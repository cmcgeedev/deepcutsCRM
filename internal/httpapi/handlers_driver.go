package httpapi

import (
	"context"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func (s *Server) GetDriverRoute(ctx context.Context, r api.GetDriverRouteRequestObject) (api.GetDriverRouteResponseObject, error) {
	sess, _ := auth.SessionFrom(ctx)
	date := deref(r.Params.Date)
	if date == "" {
		date = s.d.Svc.Today()
	}
	if !service.ValidDate(date) {
		return nil, service.Invalid(map[string]string{"date": "must be YYYY-MM-DD"})
	}
	dr, err := s.d.Svc.DriverRouteForDate(ctx, sess.UserID, date)
	if err != nil {
		return nil, err
	}
	return api.GetDriverRoute200JSONResponse(toDriverRoute(dr, sess.DisplayName)), nil
}

func (s *Server) PostDriverAction(ctx context.Context, r api.PostDriverActionRequestObject) (api.PostDriverActionResponseObject, error) {
	sess, _ := auth.SessionFrom(ctx)
	a, err := driverActionInput(*r.Body)
	if err != nil {
		return nil, err
	}
	applied, st, err := s.d.Svc.ApplyDriverAction(ctx, sess.UserID, r.StopId, a)
	if err != nil {
		return nil, err
	}
	return api.PostDriverAction200JSONResponse(api.DriverActionResult{Applied: applied, Stop: toDriverStop(st)}), nil
}

func (s *Server) DriverCompleteRoute(ctx context.Context, r api.DriverCompleteRouteRequestObject) (api.DriverCompleteRouteResponseObject, error) {
	sess, _ := auth.SessionFrom(ctx)
	dr, err := s.d.Svc.DriverCompleteRoute(ctx, sess.UserID, r.RouteId)
	if err != nil {
		return nil, err
	}
	return api.DriverCompleteRoute200JSONResponse(toDriverRoute(dr, sess.DisplayName)), nil
}
