package httpapi

// stubs_tmp.go: temporary not-implemented stubs for strict handler methods that
// Tasks 11 and 12 implement. Each task deletes the stubs it replaces; Task 12
// deletes this file.

import (
	"context"
	"net/http"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func (s *Server) GetDriverRoute(ctx context.Context, request api.GetDriverRouteRequestObject) (api.GetDriverRouteResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) DriverCompleteRoute(ctx context.Context, request api.DriverCompleteRouteRequestObject) (api.DriverCompleteRouteResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) PostDriverAction(ctx context.Context, request api.PostDriverActionRequestObject) (api.PostDriverActionResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) ListCustomers(ctx context.Context, request api.ListCustomersRequestObject) (api.ListCustomersResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) CreateCustomer(ctx context.Context, request api.CreateCustomerRequestObject) (api.CreateCustomerResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) GetCustomer(ctx context.Context, request api.GetCustomerRequestObject) (api.GetCustomerResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) UpdateCustomer(ctx context.Context, request api.UpdateCustomerRequestObject) (api.UpdateCustomerResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) ListCustomerPrices(ctx context.Context, request api.ListCustomerPricesRequestObject) (api.ListCustomerPricesResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) SetCustomerPrice(ctx context.Context, request api.SetCustomerPriceRequestObject) (api.SetCustomerPriceResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) GetDay(ctx context.Context, request api.GetDayRequestObject) (api.GetDayResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) ListDrivers(ctx context.Context, request api.ListDriversRequestObject) (api.ListDriversResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) ListOrders(ctx context.Context, request api.ListOrdersRequestObject) (api.ListOrdersResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) CreateOrder(ctx context.Context, request api.CreateOrderRequestObject) (api.CreateOrderResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) GetOrder(ctx context.Context, request api.GetOrderRequestObject) (api.GetOrderResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) UpdateOrder(ctx context.Context, request api.UpdateOrderRequestObject) (api.UpdateOrderResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) CancelOrder(ctx context.Context, request api.CancelOrderRequestObject) (api.CancelOrderResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) ConfirmOrder(ctx context.Context, request api.ConfirmOrderRequestObject) (api.ConfirmOrderResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) FinalizeOrder(ctx context.Context, request api.FinalizeOrderRequestObject) (api.FinalizeOrderResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) AddOrderLine(ctx context.Context, request api.AddOrderLineRequestObject) (api.AddOrderLineResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) DeleteOrderLine(ctx context.Context, request api.DeleteOrderLineRequestObject) (api.DeleteOrderLineResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) UpdateOrderLine(ctx context.Context, request api.UpdateOrderLineRequestObject) (api.UpdateOrderLineResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) UnconfirmOrder(ctx context.Context, request api.UnconfirmOrderRequestObject) (api.UnconfirmOrderResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) ListProducts(ctx context.Context, request api.ListProductsRequestObject) (api.ListProductsResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) CreateProduct(ctx context.Context, request api.CreateProductRequestObject) (api.CreateProductResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) GetProduct(ctx context.Context, request api.GetProductRequestObject) (api.GetProductResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) UpdateProduct(ctx context.Context, request api.UpdateProductRequestObject) (api.UpdateProductResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) CreateRoute(ctx context.Context, request api.CreateRouteRequestObject) (api.CreateRouteResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) GetRoute(ctx context.Context, request api.GetRouteRequestObject) (api.GetRouteResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) RouteComplete(ctx context.Context, request api.RouteCompleteRequestObject) (api.RouteCompleteResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) RouteOut(ctx context.Context, request api.RouteOutRequestObject) (api.RouteOutResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) ReorderStops(ctx context.Context, request api.ReorderStopsRequestObject) (api.ReorderStopsResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) AddStop(ctx context.Context, request api.AddStopRequestObject) (api.AddStopResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) RemoveStop(ctx context.Context, request api.RemoveStopRequestObject) (api.RemoveStopResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

// getStopProof streams the signature/photo for a stop; implemented in Task 12.
func (s *Server) getStopProof(w http.ResponseWriter, r *http.Request) {
	writeError(w, service.Conflict("not_implemented", "not implemented"))
}
