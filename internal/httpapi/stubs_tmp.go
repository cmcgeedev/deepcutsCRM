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

func (s *Server) GetDay(ctx context.Context, request api.GetDayRequestObject) (api.GetDayResponseObject, error) {
	return nil, service.Conflict("not_implemented", "not implemented")
}

func (s *Server) ListDrivers(ctx context.Context, request api.ListDriversRequestObject) (api.ListDriversResponseObject, error) {
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
