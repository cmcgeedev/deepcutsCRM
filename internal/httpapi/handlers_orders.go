package httpapi

import (
	"context"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func (s *Server) ListOrders(ctx context.Context, r api.ListOrdersRequestObject) (api.ListOrdersResponseObject, error) {
	rows, err := s.d.Svc.ListOrders(ctx, service.OrderFilter{Date: deref(r.Params.Date), Status: deref(r.Params.Status), CustomerID: derefI64(r.Params.CustomerId)})
	if err != nil {
		return nil, err
	}
	out := make(api.ListOrders200JSONResponse, 0, len(rows))
	for _, o := range rows {
		out = append(out, toOrderSummary(o.ID, o.CustomerID, o.CustomerName, o.RequestedDeliveryDate, o.Status, o.NeedsReview, o.LineCount, o.Notes))
	}
	return out, nil
}

func (s *Server) CreateOrder(ctx context.Context, r api.CreateOrderRequestObject) (api.CreateOrderResponseObject, error) {
	sess, _ := auth.SessionFrom(ctx)
	o, err := s.d.Svc.CreateOrder(ctx, service.OrderInput{CustomerID: r.Body.CustomerId, RequestedDeliveryDate: r.Body.RequestedDeliveryDate, Notes: deref(r.Body.Notes), CreatedBy: sess.UserID})
	if err != nil {
		return nil, err
	}
	return api.CreateOrder201JSONResponse(toOrder(o)), nil
}

func (s *Server) GetOrder(ctx context.Context, r api.GetOrderRequestObject) (api.GetOrderResponseObject, error) {
	o, err := s.d.Svc.GetOrder(ctx, r.OrderId)
	if err != nil {
		return nil, err
	}
	return api.GetOrder200JSONResponse(toOrder(o)), nil
}

func (s *Server) UpdateOrder(ctx context.Context, r api.UpdateOrderRequestObject) (api.UpdateOrderResponseObject, error) {
	o, err := s.d.Svc.UpdateOrder(ctx, r.OrderId, r.Body.Notes, r.Body.RequestedDeliveryDate)
	if err != nil {
		return nil, err
	}
	return api.UpdateOrder200JSONResponse(toOrder(o)), nil
}

func (s *Server) AddOrderLine(ctx context.Context, r api.AddOrderLineRequestObject) (api.AddOrderLineResponseObject, error) {
	o, err := s.d.Svc.AddLine(ctx, r.OrderId, r.Body.ProductId, r.Body.OrderedQty)
	if err != nil {
		return nil, err
	}
	return api.AddOrderLine200JSONResponse(toOrder(o)), nil
}

func (s *Server) UpdateOrderLine(ctx context.Context, r api.UpdateOrderLineRequestObject) (api.UpdateOrderLineResponseObject, error) {
	b := r.Body
	o, err := s.d.Svc.UpdateLine(ctx, r.OrderId, r.LineId, service.LinePatch{OrderedQty: b.OrderedQty, UnitPriceCents: b.UnitPriceCents, ShippedWeight: b.ShippedWeight,
		DeliveredQty: b.DeliveredQty, DeliveredWeight: b.DeliveredWeight, ShortageNote: b.ShortageNote})
	if err != nil {
		return nil, err
	}
	return api.UpdateOrderLine200JSONResponse(toOrder(o)), nil
}

func (s *Server) DeleteOrderLine(ctx context.Context, r api.DeleteOrderLineRequestObject) (api.DeleteOrderLineResponseObject, error) {
	o, err := s.d.Svc.DeleteLine(ctx, r.OrderId, r.LineId)
	if err != nil {
		return nil, err
	}
	return api.DeleteOrderLine200JSONResponse(toOrder(o)), nil
}

func (s *Server) ConfirmOrder(ctx context.Context, r api.ConfirmOrderRequestObject) (api.ConfirmOrderResponseObject, error) {
	o, err := s.d.Svc.ConfirmOrder(ctx, r.OrderId)
	if err != nil {
		return nil, err
	}
	return api.ConfirmOrder200JSONResponse(toOrder(o)), nil
}

func (s *Server) UnconfirmOrder(ctx context.Context, r api.UnconfirmOrderRequestObject) (api.UnconfirmOrderResponseObject, error) {
	o, err := s.d.Svc.UnconfirmOrder(ctx, r.OrderId)
	if err != nil {
		return nil, err
	}
	return api.UnconfirmOrder200JSONResponse(toOrder(o)), nil
}

func (s *Server) CancelOrder(ctx context.Context, r api.CancelOrderRequestObject) (api.CancelOrderResponseObject, error) {
	o, err := s.d.Svc.CancelOrder(ctx, r.OrderId)
	if err != nil {
		return nil, err
	}
	return api.CancelOrder200JSONResponse(toOrder(o)), nil
}

func (s *Server) FinalizeOrder(ctx context.Context, r api.FinalizeOrderRequestObject) (api.FinalizeOrderResponseObject, error) {
	o, err := s.d.Svc.FinalizeOrder(ctx, r.OrderId)
	if err != nil {
		return nil, err
	}
	return api.FinalizeOrder200JSONResponse(toOrder(o)), nil
}
