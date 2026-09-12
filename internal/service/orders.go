package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
)

type OrderInput struct {
	CustomerID            int64
	RequestedDeliveryDate string
	Notes                 string
	CreatedBy             int64
}

type OrderFilter struct {
	Date       string
	Status     string
	CustomerID int64
}

type LineDetail struct {
	Line         queries.OrderLine
	Product      queries.Product
	AmountCents  int64
	AmountSource string
}

type OrderDetail struct {
	Order       queries.Order
	Customer    queries.Customer
	Lines       []LineDetail
	TotalCents  int64
	RouteID     *int64
	StopID      *int64
	RouteStatus string // "" when not scheduled
}

type LinePatch struct {
	OrderedQty, UnitPriceCents, ShippedWeight, DeliveredQty, DeliveredWeight *int64
	ShortageNote                                                             *string
}

func (s *Service) CreateOrder(ctx context.Context, in OrderInput) (OrderDetail, error) {
	if !ValidDate(in.RequestedDeliveryDate) {
		return OrderDetail{}, Invalid(map[string]string{"requestedDeliveryDate": "must be YYYY-MM-DD"})
	}
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		if _, err := q.GetCustomer(ctx, in.CustomerID); err != nil {
			return notFoundIf(err, "customer")
		}
		now := s.now()
		o, err := q.CreateOrder(ctx, queries.CreateOrderParams{
			CustomerID: in.CustomerID, RequestedDeliveryDate: in.RequestedDeliveryDate, Notes: in.Notes,
			CreatedBy: nullInt(nonZero(in.CreatedBy)), CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, o.ID)
		return err
	})
	return out, err
}

func nonZero(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

func (s *Service) GetOrder(ctx context.Context, id int64) (OrderDetail, error) {
	return s.loadOrder(ctx, s.Q, id)
}

func (s *Service) ListOrders(ctx context.Context, f OrderFilter) ([]queries.ListOrdersRow, error) {
	return s.Q.ListOrders(ctx, queries.ListOrdersParams{Date: f.Date, Status: f.Status, CustomerID: f.CustomerID})
}

func (s *Service) UpdateOrder(ctx context.Context, id int64, notes, date *string) (OrderDetail, error) {
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		o, err := q.GetOrder(ctx, id)
		if err != nil {
			return notFoundIf(err, "order")
		}
		if scope, _ := s.editScope(ctx, q, o); scope != domain.EditAll {
			return Conflict("locked", "order can no longer be edited")
		}
		if notes != nil {
			o.Notes = *notes
		}
		if date != nil {
			if !ValidDate(*date) {
				return Invalid(map[string]string{"requestedDeliveryDate": "must be YYYY-MM-DD"})
			}
			if o.Status == "scheduled" {
				return Conflict("locked", "unschedule the order before changing its date")
			}
			o.RequestedDeliveryDate = *date
		}
		if err := q.UpdateOrderMeta(ctx, queries.UpdateOrderMetaParams{Notes: o.Notes, RequestedDeliveryDate: o.RequestedDeliveryDate, UpdatedAt: s.now(), ID: id}); err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, id)
		return err
	})
	return out, err
}

// editScope returns what may be edited on the order's lines, and the active route status if any.
func (s *Service) editScope(ctx context.Context, q *queries.Queries, o queries.Order) (domain.LineEditScope, string) {
	routeStatus := ""
	if stop, err := q.GetActiveStopForOrder(ctx, o.ID); err == nil {
		routeStatus = stop.RouteStatus
	}
	return domain.LineEditScopeFor(domain.OrderStatus(o.Status), domain.RouteStatus(routeStatus)), routeStatus
}

func (s *Service) AddLine(ctx context.Context, orderID, productID, orderedQty int64) (OrderDetail, error) {
	if orderedQty <= 0 {
		return OrderDetail{}, Invalid(map[string]string{"orderedQty": "must be positive"})
	}
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		o, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return notFoundIf(err, "order")
		}
		if scope, _ := s.editScope(ctx, q, o); scope != domain.EditAll {
			return Conflict("locked", "lines cannot be added to this order")
		}
		p, err := q.GetProduct(ctx, productID)
		if err != nil {
			return notFoundIf(err, "product")
		}
		dp := productOf(p)
		price, _, err := s.resolvePrice(ctx, q, o.CustomerID, productID, dp.BasePrice, o.RequestedDeliveryDate)
		if err != nil {
			return err
		}
		var est sql.NullInt64
		if w, ok := domain.EstimateWeight(dp, domain.Hundredths(orderedQty)); ok {
			est = sql.NullInt64{Int64: int64(w), Valid: true}
		}
		if _, err := q.CreateOrderLine(ctx, queries.CreateOrderLineParams{
			OrderID: orderID, ProductID: productID, OrderedQty: orderedQty, UnitPriceCents: int64(price), EstWeight: est,
		}); err != nil {
			return err
		}
		if err := q.UpdateOrderMeta(ctx, queries.UpdateOrderMetaParams{Notes: o.Notes, RequestedDeliveryDate: o.RequestedDeliveryDate, UpdatedAt: s.now(), ID: orderID}); err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, orderID)
		return err
	})
	return out, err
}

func (s *Service) UpdateLine(ctx context.Context, orderID, lineID int64, patch LinePatch) (OrderDetail, error) {
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		o, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return notFoundIf(err, "order")
		}
		l, err := q.GetOrderLine(ctx, queries.GetOrderLineParams{ID: lineID, OrderID: orderID})
		if err != nil {
			return notFoundIf(err, "line")
		}
		p, err := q.GetProduct(ctx, l.ProductID)
		if err != nil {
			return err
		}
		scope, _ := s.editScope(ctx, q, o)
		if err := applyLinePatch(&l, productOf(p), patch, scope); err != nil {
			return err
		}
		if err := q.UpdateOrderLine(ctx, queries.UpdateOrderLineParams{
			OrderedQty: l.OrderedQty, UnitPriceCents: l.UnitPriceCents, PriceOverridden: l.PriceOverridden, EstWeight: l.EstWeight,
			ShippedWeight: l.ShippedWeight, DeliveredQty: l.DeliveredQty, DeliveredWeight: l.DeliveredWeight, ShortageNote: l.ShortageNote, ID: l.ID,
		}); err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, orderID)
		return err
	})
	return out, err
}

// applyLinePatch mutates l according to the patch, enforcing the edit scope.
func applyLinePatch(l *queries.OrderLine, p domain.Product, patch LinePatch, scope domain.LineEditScope) error {
	fields := map[string]string{}
	locked := func(f string) { fields[f] = "cannot be changed in the order's current state" }
	if patch.OrderedQty != nil {
		if scope != domain.EditAll {
			locked("orderedQty")
		} else if *patch.OrderedQty <= 0 {
			fields["orderedQty"] = "must be positive"
		} else {
			l.OrderedQty = *patch.OrderedQty
			l.EstWeight = sql.NullInt64{}
			if w, ok := domain.EstimateWeight(p, domain.Hundredths(l.OrderedQty)); ok {
				l.EstWeight = sql.NullInt64{Int64: int64(w), Valid: true}
			}
		}
	}
	if patch.UnitPriceCents != nil {
		if scope != domain.EditAll && scope != domain.EditShippedAndPrice {
			locked("unitPriceCents")
		} else if *patch.UnitPriceCents < 0 {
			fields["unitPriceCents"] = "must not be negative"
		} else {
			l.UnitPriceCents = *patch.UnitPriceCents
			l.PriceOverridden = true
		}
	}
	if patch.ShippedWeight != nil {
		if scope != domain.EditAll && scope != domain.EditShippedAndPrice {
			locked("shippedWeight")
		} else if !p.CatchWeight {
			fields["shippedWeight"] = "only catch-weight lines have a shipped weight"
		} else if *patch.ShippedWeight < 0 {
			fields["shippedWeight"] = "must not be negative"
		} else {
			l.ShippedWeight = sql.NullInt64{Int64: *patch.ShippedWeight, Valid: true}
		}
	}
	for name, v := range map[string]*int64{"deliveredQty": patch.DeliveredQty, "deliveredWeight": patch.DeliveredWeight} {
		if v == nil {
			continue
		}
		if scope != domain.EditDelivered {
			locked(name)
		} else if *v < 0 {
			fields[name] = "must not be negative"
		} else if name == "deliveredQty" {
			l.DeliveredQty = sql.NullInt64{Int64: *v, Valid: true}
		} else if !p.CatchWeight {
			fields[name] = "only catch-weight lines have a delivered weight"
		} else {
			l.DeliveredWeight = sql.NullInt64{Int64: *v, Valid: true}
		}
	}
	if patch.ShortageNote != nil {
		if scope == domain.EditNone {
			locked("shortageNote")
		} else {
			l.ShortageNote = *patch.ShortageNote
		}
	}
	if len(fields) > 0 {
		return Invalid(fields)
	}
	return nil
}

func (s *Service) DeleteLine(ctx context.Context, orderID, lineID int64) (OrderDetail, error) {
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		o, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return notFoundIf(err, "order")
		}
		if scope, _ := s.editScope(ctx, q, o); scope != domain.EditAll {
			return Conflict("locked", "lines cannot be removed from this order")
		}
		if _, err := q.GetOrderLine(ctx, queries.GetOrderLineParams{ID: lineID, OrderID: orderID}); err != nil {
			return notFoundIf(err, "line")
		}
		if err := q.DeleteOrderLine(ctx, queries.DeleteOrderLineParams{ID: lineID, OrderID: orderID}); err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, orderID)
		return err
	})
	return out, err
}

// transitionOrder checks the state machine and writes the new status.
func (s *Service) transitionOrder(ctx context.Context, q *queries.Queries, o queries.Order, to domain.OrderStatus) error {
	from := domain.OrderStatus(o.Status)
	if !from.CanTransition(to) {
		return Conflict("invalid_transition", fmt.Sprintf("order cannot go from %s to %s", from, to))
	}
	return q.UpdateOrderStatus(ctx, queries.UpdateOrderStatusParams{Status: string(to), UpdatedAt: s.now(), ID: o.ID})
}

func (s *Service) simpleTransition(ctx context.Context, id int64, to domain.OrderStatus, pre func(q *queries.Queries, o queries.Order) error) (OrderDetail, error) {
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		o, err := q.GetOrder(ctx, id)
		if err != nil {
			return notFoundIf(err, "order")
		}
		if pre != nil {
			if err := pre(q, o); err != nil {
				return err
			}
		}
		if err := s.transitionOrder(ctx, q, o, to); err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, id)
		return err
	})
	return out, err
}

func (s *Service) ConfirmOrder(ctx context.Context, id int64) (OrderDetail, error) {
	return s.simpleTransition(ctx, id, domain.OrderConfirmed, func(q *queries.Queries, o queries.Order) error {
		if !domain.OrderStatus(o.Status).CanTransition(domain.OrderConfirmed) {
			return Conflict("invalid_transition", fmt.Sprintf("order cannot go from %s to confirmed", o.Status))
		}
		n, err := q.CountOrderLines(ctx, id)
		if err != nil {
			return err
		}
		if n == 0 {
			return Conflict("no_lines", "add at least one line before confirming")
		}
		return nil
	})
}

func (s *Service) UnconfirmOrder(ctx context.Context, id int64) (OrderDetail, error) {
	return s.simpleTransition(ctx, id, domain.OrderDraft, nil)
}

// CancelOrder cancels a draft, confirmed or scheduled order. A scheduled order's stop is removed
// only while the route is planned; once the route is out the office must skip it via the driver flow.
func (s *Service) CancelOrder(ctx context.Context, id int64) (OrderDetail, error) {
	return s.simpleTransition(ctx, id, domain.OrderCancelled, func(q *queries.Queries, o queries.Order) error {
		stop, err := q.GetActiveStopForOrder(ctx, o.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if stop.RouteStatus != string(domain.RoutePlanned) {
			return Conflict("locked", "route is already out; skip the stop from the driver app instead")
		}
		return q.DeleteStop(ctx, stop.StopID)
	})
}

// FinalizeOrder locks a delivered order after checking every line has what it needs to be billed.
func (s *Service) FinalizeOrder(ctx context.Context, id int64) (OrderDetail, error) {
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		o, err := q.GetOrder(ctx, id)
		if err != nil {
			return notFoundIf(err, "order")
		}
		if !domain.OrderStatus(o.Status).CanTransition(domain.OrderFinalized) {
			return Conflict("invalid_transition", fmt.Sprintf("order cannot go from %s to finalized", o.Status))
		}
		lines, err := q.ListOrderLines(ctx, id)
		if err != nil {
			return err
		}
		fields := map[string]string{}
		for _, l := range lines {
			key := fmt.Sprintf("line:%d", l.ID)
			if !l.DeliveredQty.Valid {
				fields[key] = "missing delivered quantity"
			} else if l.CatchWeight && !l.DeliveredWeight.Valid {
				fields[key] = "missing delivered weight"
			}
		}
		if len(fields) > 0 {
			return &Error{Status: 409, Code: "incomplete", Message: "some lines are missing delivered quantities", Fields: fields}
		}
		now := s.now()
		if err := q.FinalizeOrder(ctx, queries.FinalizeOrderParams{FinalizedAt: sql.NullTime{Time: now, Valid: true}, UpdatedAt: now, ID: id}); err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, id)
		return err
	})
	return out, err
}

// loadOrder assembles an OrderDetail with computed amounts.
func (s *Service) loadOrder(ctx context.Context, q *queries.Queries, id int64) (OrderDetail, error) {
	o, err := q.GetOrder(ctx, id)
	if err != nil {
		return OrderDetail{}, notFoundIf(err, "order")
	}
	c, err := q.GetCustomer(ctx, o.CustomerID)
	if err != nil {
		return OrderDetail{}, err
	}
	rows, err := q.ListOrderLines(ctx, id)
	if err != nil {
		return OrderDetail{}, err
	}
	d := OrderDetail{Order: o, Customer: c, Lines: make([]LineDetail, 0, len(rows))}
	for _, r := range rows {
		p := queries.Product{ID: r.ProductID, Sku: r.Sku, Name: r.ProductName, Category: r.Category, SellUnit: r.SellUnit,
			CatchWeight: r.CatchWeight, ApproxCaseWeight: r.ApproxCaseWeight, BasePriceCents: r.BasePriceCents}
		line := queries.OrderLine{ID: r.ID, OrderID: r.OrderID, ProductID: r.ProductID, OrderedQty: r.OrderedQty,
			UnitPriceCents: r.UnitPriceCents, PriceOverridden: r.PriceOverridden, EstWeight: r.EstWeight, ShippedWeight: r.ShippedWeight,
			DeliveredQty: r.DeliveredQty, DeliveredWeight: r.DeliveredWeight, ShortageNote: r.ShortageNote}
		dl := domain.Line{Product: productOf(p), OrderedQty: domain.Hundredths(line.OrderedQty), UnitPrice: domain.Cents(line.UnitPriceCents),
			EstWeight: hPtr(line.EstWeight), ShippedWeight: hPtr(line.ShippedWeight), DeliveredQty: hPtr(line.DeliveredQty), DeliveredWeight: hPtr(line.DeliveredWeight)}
		_, src := dl.BillableQty()
		amt := int64(domain.LineAmount(dl))
		d.Lines = append(d.Lines, LineDetail{Line: line, Product: p, AmountCents: amt, AmountSource: src})
		d.TotalCents += amt
	}
	if stop, err := q.GetActiveStopForOrder(ctx, id); err == nil {
		d.RouteID, d.StopID, d.RouteStatus = &stop.RouteID, &stop.StopID, stop.RouteStatus
	}
	return d, nil
}
