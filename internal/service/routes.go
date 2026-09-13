package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
)

type RouteInput struct {
	RouteDate    string
	DriverUserID int64
	TruckLabel   string
}

type StopDetail struct {
	Stop      queries.DeliveryStop
	Order     queries.Order
	Customer  queries.Customer
	LineCount int64
}

type RouteDetail struct {
	Route      queries.DeliveryRoute
	DriverName string
	Stops      []StopDetail
}

type DayView struct {
	Date        string
	Unscheduled []queries.ListUnscheduledOrdersForDateRow
	Routes      []RouteDetail
}

func (s *Service) ListDrivers(ctx context.Context) ([]queries.User, error) {
	return s.Q.ListUsersByRealm(ctx, "driver")
}

func (s *Service) CreateRoute(ctx context.Context, in RouteInput) (RouteDetail, error) {
	if !ValidDate(in.RouteDate) {
		return RouteDetail{}, Invalid(map[string]string{"routeDate": "must be YYYY-MM-DD"})
	}
	var out RouteDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		u, err := q.GetUser(ctx, in.DriverUserID)
		if err != nil || u.Realm != "driver" || !u.Active {
			return NotFound("driver")
		}
		_, err = q.GetRouteForDriverDate(ctx, queries.GetRouteForDriverDateParams{DriverUserID: in.DriverUserID, RouteDate: in.RouteDate})
		if err == nil {
			return Conflict("duplicate_route", "this driver already has a route for that date")
		}
		if err != sql.ErrNoRows {
			return err
		}
		r, err := q.CreateRoute(ctx, queries.CreateRouteParams{RouteDate: in.RouteDate, DriverUserID: in.DriverUserID, TruckLabel: in.TruckLabel, CreatedAt: s.now()})
		if err != nil {
			return err
		}
		out, err = s.loadRoute(ctx, q, r.ID)
		return err
	})
	return out, err
}

func (s *Service) GetRoute(ctx context.Context, id int64) (RouteDetail, error) {
	return s.loadRoute(ctx, s.Q, id)
}

func (s *Service) DayView(ctx context.Context, date string) (DayView, error) {
	if !ValidDate(date) {
		return DayView{}, Invalid(map[string]string{"date": "must be YYYY-MM-DD"})
	}
	un, err := s.Q.ListUnscheduledOrdersForDate(ctx, date)
	if err != nil {
		return DayView{}, err
	}
	routes, err := s.Q.ListRoutesByDate(ctx, date)
	if err != nil {
		return DayView{}, err
	}
	dv := DayView{Date: date, Unscheduled: un, Routes: []RouteDetail{}}
	for _, r := range routes {
		rd, err := s.loadRoute(ctx, s.Q, r.ID)
		if err != nil {
			return DayView{}, err
		}
		dv.Routes = append(dv.Routes, rd)
	}
	return dv, nil
}

func (s *Service) AddStop(ctx context.Context, routeID, orderID int64) (RouteDetail, error) {
	var out RouteDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		r, err := q.GetRoute(ctx, routeID)
		if err != nil {
			return notFoundIf(err, "route")
		}
		if r.Status != string(domain.RoutePlanned) {
			return Conflict("locked", "stops cannot be added once the route is out")
		}
		o, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return notFoundIf(err, "order")
		}
		if o.RequestedDeliveryDate != r.RouteDate {
			return Conflict("date_mismatch", "order is for a different delivery date than the route")
		}
		if err := s.transitionOrder(ctx, q, o, domain.OrderScheduled); err != nil {
			return err
		}
		maxSeq, err := q.MaxStopSequence(ctx, routeID)
		if err != nil {
			return err
		}
		if _, err := q.CreateStop(ctx, queries.CreateStopParams{RouteID: routeID, OrderID: orderID, Sequence: maxSeq + 1}); err != nil {
			return err
		}
		out, err = s.loadRoute(ctx, q, routeID)
		return err
	})
	return out, err
}

func (s *Service) RemoveStop(ctx context.Context, routeID, stopID int64) (RouteDetail, error) {
	var out RouteDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		r, err := q.GetRoute(ctx, routeID)
		if err != nil {
			return notFoundIf(err, "route")
		}
		if r.Status != string(domain.RoutePlanned) {
			return Conflict("locked", "stops cannot be removed once the route is out")
		}
		st, err := q.GetStop(ctx, stopID)
		if err != nil || st.RouteID != routeID {
			return NotFound("stop")
		}
		o, err := q.GetOrder(ctx, st.OrderID)
		if err != nil {
			return err
		}
		if err := s.transitionOrder(ctx, q, o, domain.OrderConfirmed); err != nil {
			return err
		}
		if err := q.DeleteStop(ctx, stopID); err != nil {
			return err
		}
		out, err = s.loadRoute(ctx, q, routeID)
		return err
	})
	return out, err
}

func (s *Service) ReorderStops(ctx context.Context, routeID int64, stopIDs []int64) (RouteDetail, error) {
	var out RouteDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		r, err := q.GetRoute(ctx, routeID)
		if err != nil {
			return notFoundIf(err, "route")
		}
		if r.Status != string(domain.RoutePlanned) {
			return Conflict("locked", "stop order is fixed once the route is out")
		}
		existing, err := q.ListStopsForRoute(ctx, routeID)
		if err != nil {
			return err
		}
		have := map[int64]bool{}
		for _, st := range existing {
			have[st.ID] = true
		}
		seen := map[int64]bool{}
		for _, id := range stopIDs {
			if !have[id] || seen[id] {
				return Invalid(map[string]string{"stopIds": "must list each stop on the route exactly once"})
			}
			seen[id] = true
		}
		if len(seen) != len(have) {
			return Invalid(map[string]string{"stopIds": "must list each stop on the route exactly once"})
		}
		for i, id := range stopIDs {
			if err := q.UpdateStopSequence(ctx, queries.UpdateStopSequenceParams{Sequence: int64(i + 1), ID: id, RouteID: routeID}); err != nil {
				return err
			}
		}
		out, err = s.loadRoute(ctx, q, routeID)
		return err
	})
	return out, err
}

func (s *Service) RouteOut(ctx context.Context, routeID int64) (RouteDetail, error) {
	var out RouteDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		r, err := q.GetRoute(ctx, routeID)
		if err != nil {
			return notFoundIf(err, "route")
		}
		if !domain.RouteStatus(r.Status).CanTransition(domain.RouteOut) {
			return Conflict("invalid_transition", fmt.Sprintf("route cannot go from %s to out", r.Status))
		}
		n, err := q.MaxStopSequence(ctx, routeID)
		if err != nil {
			return err
		}
		if n == 0 {
			return Conflict("empty_route", "add at least one stop before marking the route out")
		}
		now := s.now()
		if err := q.UpdateRouteStatus(ctx, queries.UpdateRouteStatusParams{Status: string(domain.RouteOut), OutAt: sql.NullTime{Time: now, Valid: true}, ID: routeID}); err != nil {
			return err
		}
		out, err = s.loadRoute(ctx, q, routeID)
		return err
	})
	return out, err
}

func (s *Service) RouteComplete(ctx context.Context, routeID int64) (RouteDetail, error) {
	var out RouteDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		r, err := q.GetRoute(ctx, routeID)
		if err != nil {
			return notFoundIf(err, "route")
		}
		if !domain.RouteStatus(r.Status).CanTransition(domain.RouteComplete) {
			return Conflict("invalid_transition", fmt.Sprintf("route cannot go from %s to complete", r.Status))
		}
		pending, err := q.CountPendingStops(ctx, routeID)
		if err != nil {
			return err
		}
		if pending > 0 {
			return Conflict("stops_pending", fmt.Sprintf("%d stops are still pending", pending))
		}
		now := s.now()
		if err := q.UpdateRouteStatus(ctx, queries.UpdateRouteStatusParams{Status: string(domain.RouteComplete), OutAt: r.OutAt, CompletedAt: sql.NullTime{Time: now, Valid: true}, ID: routeID}); err != nil {
			return err
		}
		out, err = s.loadRoute(ctx, q, routeID)
		return err
	})
	return out, err
}

func (s *Service) loadRoute(ctx context.Context, q *queries.Queries, id int64) (RouteDetail, error) {
	r, err := q.GetRoute(ctx, id)
	if err != nil {
		return RouteDetail{}, notFoundIf(err, "route")
	}
	u, err := q.GetUser(ctx, r.DriverUserID)
	if err != nil {
		return RouteDetail{}, err
	}
	rows, err := q.ListStopsForRoute(ctx, id)
	if err != nil {
		return RouteDetail{}, err
	}
	rd := RouteDetail{Route: r, DriverName: u.DisplayName, Stops: make([]StopDetail, 0, len(rows))}
	for _, x := range rows {
		rd.Stops = append(rd.Stops, StopDetail{
			Stop: queries.DeliveryStop{ID: x.ID, RouteID: x.RouteID, OrderID: x.OrderID, Sequence: x.Sequence, Status: x.Status,
				DeliveredAt: x.DeliveredAt, ProofType: x.ProofType, ProofRef: x.ProofRef, SkipReason: x.SkipReason, DriverNote: x.DriverNote},
			Order: queries.Order{ID: x.OrderID, CustomerID: x.CustomerID, Status: x.OrderStatus, NeedsReview: x.NeedsReview,
				RequestedDeliveryDate: x.RequestedDeliveryDate, Notes: x.OrderNotes},
			Customer: queries.Customer{ID: x.CustomerID, Name: x.CustomerName, DeliveryAddress: x.DeliveryAddress, Phone: x.Phone,
				ContactName: x.ContactName, DeliveryNotes: x.DeliveryNotes},
			LineCount: x.LineCount,
		})
	}
	return rd, nil
}
