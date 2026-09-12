package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
)

// ProofStore persists signature/photo images and returns a reference.
type ProofStore interface {
	Save(kind string, data []byte) (string, error)
}

type DriverStop struct {
	Stop     queries.DeliveryStop
	Customer queries.Customer
	Order    OrderDetail
}

type DriverRoute struct {
	Route queries.DeliveryRoute
	Stops []DriverStop
}

func (s *Service) DriverRouteForDate(ctx context.Context, driverUserID int64, date string) (DriverRoute, error) {
	r, err := s.Q.GetRouteForDriverDate(ctx, queries.GetRouteForDriverDateParams{DriverUserID: driverUserID, RouteDate: date})
	if err != nil {
		return DriverRoute{}, notFoundIf(err, "route")
	}
	return s.loadDriverRoute(ctx, s.Q, r)
}

func (s *Service) loadDriverRoute(ctx context.Context, q *queries.Queries, r queries.DeliveryRoute) (DriverRoute, error) {
	stops, err := q.ListStopsForRoute(ctx, r.ID)
	if err != nil {
		return DriverRoute{}, err
	}
	out := DriverRoute{Route: r, Stops: make([]DriverStop, 0, len(stops))}
	for _, st := range stops {
		ds, err := s.loadDriverStop(ctx, q, st.ID)
		if err != nil {
			return DriverRoute{}, err
		}
		out.Stops = append(out.Stops, ds)
	}
	return out, nil
}

func (s *Service) loadDriverStop(ctx context.Context, q *queries.Queries, stopID int64) (DriverStop, error) {
	st, err := q.GetStop(ctx, stopID)
	if err != nil {
		return DriverStop{}, notFoundIf(err, "stop")
	}
	od, err := s.loadOrder(ctx, q, st.OrderID)
	if err != nil {
		return DriverStop{}, err
	}
	return DriverStop{Stop: st, Customer: od.Customer, Order: od}, nil
}

// ownedStop loads a stop and verifies it belongs to a route driven by driverUserID.
func (s *Service) ownedStop(ctx context.Context, q *queries.Queries, driverUserID, stopID int64) (queries.DeliveryStop, queries.DeliveryRoute, error) {
	st, err := q.GetStop(ctx, stopID)
	if err != nil {
		return st, queries.DeliveryRoute{}, notFoundIf(err, "stop")
	}
	r, err := q.GetRoute(ctx, st.RouteID)
	if err != nil {
		return st, r, err
	}
	if r.DriverUserID != driverUserID {
		return st, r, NotFound("stop")
	}
	return st, r, nil
}

// ApplyDriverAction applies a deliver/adjust/skip action exactly once per ClientID.
func (s *Service) ApplyDriverAction(ctx context.Context, driverUserID, stopID int64, a domain.DriverAction) (bool, DriverStop, error) {
	if errs := a.Validate(); len(errs) > 0 {
		return false, DriverStop{}, Invalid(errs)
	}
	applied := false
	var out DriverStop
	err := s.Tx(ctx, func(q *queries.Queries) error {
		st, r, err := s.ownedStop(ctx, q, driverUserID, stopID)
		if err != nil {
			return err
		}
		if prev, err := q.GetDriverAction(ctx, a.ClientID); err == nil {
			if prev.StopID != stopID {
				return Conflict("client_id_reused", "this action id was already used for another stop")
			}
			out, err = s.loadDriverStop(ctx, q, stopID)
			return err
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if r.Status != string(domain.RouteOut) {
			return Conflict("route_not_out", "the route is not out")
		}
		o, err := q.GetOrder(ctx, st.OrderID)
		if err != nil {
			return err
		}
		switch a.Type {
		case domain.ActionAdjust:
			if st.Status == string(domain.StopSkipped) {
				return Conflict("invalid_transition", "stop was skipped")
			}
			if err := s.applyAdjustments(ctx, q, o, a.Lines); err != nil {
				return err
			}
			if a.Note != "" {
				if err := q.UpdateStopNote(ctx, queries.UpdateStopNoteParams{DriverNote: a.Note, ID: stopID}); err != nil {
					return err
				}
			}
			// an adjustment after delivery changes what will be billed: the office must look at it
			if st.Status == string(domain.StopDelivered) {
				if err := q.SetOrderNeedsReview(ctx, queries.SetOrderNeedsReviewParams{NeedsReview: true, UpdatedAt: s.now(), ID: o.ID}); err != nil {
					return err
				}
			}
		case domain.ActionDeliver:
			if st.Status != string(domain.StopPending) {
				return Conflict("invalid_transition", "stop is already "+st.Status)
			}
			ref := a.Proof.Name
			if a.Proof.Type != domain.ProofName {
				if s.Proofs == nil {
					return errors.New("proof storage not configured")
				}
				ref, err = s.Proofs.Save("proofs", a.Proof.Data)
				if err != nil {
					return Invalid(map[string]string{"proof": err.Error()})
				}
			}
			if err := s.applyAdjustments(ctx, q, o, a.Lines); err != nil {
				return err
			}
			if err := s.defaultDelivered(ctx, q, o.ID); err != nil {
				return err
			}
			if err := q.UpdateStopDelivered(ctx, queries.UpdateStopDeliveredParams{
				DeliveredAt: sql.NullTime{Time: s.now(), Valid: true}, ProofType: nullStr(string(a.Proof.Type)), ProofRef: nullStr(ref), DriverNote: a.Note, ID: stopID,
			}); err != nil {
				return err
			}
			if err := s.transitionOrder(ctx, q, o, domain.OrderDelivered); err != nil {
				return err
			}
		case domain.ActionSkip:
			if st.Status != string(domain.StopPending) {
				return Conflict("invalid_transition", "stop is already "+st.Status)
			}
			if err := q.UpdateStopSkipped(ctx, queries.UpdateStopSkippedParams{SkipReason: a.SkipReason, DriverNote: a.Note, ID: stopID}); err != nil {
				return err
			}
			if err := s.transitionOrder(ctx, q, o, domain.OrderConfirmed); err != nil {
				return err
			}
			if err := q.SetOrderNeedsReview(ctx, queries.SetOrderNeedsReviewParams{NeedsReview: true, UpdatedAt: s.now(), ID: o.ID}); err != nil {
				return err
			}
		}
		payload, _ := json.Marshal(map[string]any{"type": a.Type, "note": a.Note, "skipReason": a.SkipReason, "lines": len(a.Lines), "proof": proofType(a.Proof)})
		if err := q.CreateDriverAction(ctx, queries.CreateDriverActionParams{ClientID: a.ClientID, StopID: stopID, ActionType: string(a.Type), Payload: string(payload), ReceivedAt: s.now()}); err != nil {
			return err
		}
		applied = true
		out, err = s.loadDriverStop(ctx, q, stopID)
		return err
	})
	return applied, out, err
}

func proofType(p *domain.Proof) string {
	if p == nil {
		return ""
	}
	return string(p.Type)
}

// applyAdjustments writes delivered quantities/weights and notes for the given lines.
func (s *Service) applyAdjustments(ctx context.Context, q *queries.Queries, o queries.Order, adj []domain.LineAdjustment) error {
	for _, a := range adj {
		l, err := q.GetOrderLine(ctx, queries.GetOrderLineParams{ID: a.LineID, OrderID: o.ID})
		if err != nil {
			return Invalid(map[string]string{"lines": fmt.Sprintf("line %d is not on this order", a.LineID)})
		}
		p, err := q.GetProduct(ctx, l.ProductID)
		if err != nil {
			return err
		}
		if a.DeliveredQty != nil {
			l.DeliveredQty = sql.NullInt64{Int64: int64(*a.DeliveredQty), Valid: true}
		}
		if a.DeliveredWeight != nil {
			if !p.CatchWeight {
				return Invalid(map[string]string{"lines": fmt.Sprintf("line %d is not catch-weight", a.LineID)})
			}
			l.DeliveredWeight = sql.NullInt64{Int64: int64(*a.DeliveredWeight), Valid: true}
		}
		if a.ShortageNote != nil {
			l.ShortageNote = *a.ShortageNote
		}
		if err := q.UpdateOrderLine(ctx, queries.UpdateOrderLineParams{
			OrderedQty: l.OrderedQty, UnitPriceCents: l.UnitPriceCents, PriceOverridden: l.PriceOverridden, EstWeight: l.EstWeight,
			ShippedWeight: l.ShippedWeight, DeliveredQty: l.DeliveredQty, DeliveredWeight: l.DeliveredWeight, ShortageNote: l.ShortageNote, ID: l.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}

// defaultDelivered fills delivered_qty from ordered_qty and delivered_weight from shipped_weight
// where the driver did not adjust, and flags the order for review if anything differs.
func (s *Service) defaultDelivered(ctx context.Context, q *queries.Queries, orderID int64) error {
	lines, err := q.ListOrderLines(ctx, orderID)
	if err != nil {
		return err
	}
	review := false
	for _, r := range lines {
		dq, dw := r.DeliveredQty, r.DeliveredWeight
		if !dq.Valid {
			dq = sql.NullInt64{Int64: r.OrderedQty, Valid: true}
		}
		if r.CatchWeight && !dw.Valid && r.ShippedWeight.Valid {
			dw = r.ShippedWeight
		}
		if dq.Int64 != r.OrderedQty || (r.CatchWeight && r.ShippedWeight.Valid && dw.Int64 != r.ShippedWeight.Int64) || r.ShortageNote != "" {
			review = true
		}
		if err := q.UpdateOrderLine(ctx, queries.UpdateOrderLineParams{
			OrderedQty: r.OrderedQty, UnitPriceCents: r.UnitPriceCents, PriceOverridden: r.PriceOverridden, EstWeight: r.EstWeight,
			ShippedWeight: r.ShippedWeight, DeliveredQty: dq, DeliveredWeight: dw, ShortageNote: r.ShortageNote, ID: r.ID,
		}); err != nil {
			return err
		}
	}
	if review {
		return q.SetOrderNeedsReview(ctx, queries.SetOrderNeedsReviewParams{NeedsReview: true, UpdatedAt: s.now(), ID: orderID})
	}
	return nil
}

func (s *Service) DriverCompleteRoute(ctx context.Context, driverUserID, routeID int64) (DriverRoute, error) {
	r, err := s.Q.GetRoute(ctx, routeID)
	if err != nil || r.DriverUserID != driverUserID {
		return DriverRoute{}, NotFound("route")
	}
	if _, err := s.RouteComplete(ctx, routeID); err != nil {
		return DriverRoute{}, err
	}
	r, _ = s.Q.GetRoute(ctx, routeID)
	return s.loadDriverRoute(ctx, s.Q, r)
}
