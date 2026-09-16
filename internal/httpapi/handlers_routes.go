package httpapi

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func (s *Server) GetDay(ctx context.Context, r api.GetDayRequestObject) (api.GetDayResponseObject, error) {
	dv, err := s.d.Svc.DayView(ctx, r.Params.Date)
	if err != nil {
		return nil, err
	}
	out := api.DayView{Date: dv.Date, Unscheduled: []api.OrderSummary{}, Routes: []api.Route{}}
	for _, o := range dv.Unscheduled {
		out.Unscheduled = append(out.Unscheduled, toOrderSummary(o.ID, o.CustomerID, o.CustomerName, o.RequestedDeliveryDate, o.Status, o.NeedsReview, o.LineCount, o.Notes))
	}
	for _, rt := range dv.Routes {
		out.Routes = append(out.Routes, toRoute(rt))
	}
	return api.GetDay200JSONResponse(out), nil
}

func (s *Server) ListDrivers(ctx context.Context, _ api.ListDriversRequestObject) (api.ListDriversResponseObject, error) {
	users, err := s.d.Svc.ListDrivers(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListDrivers200JSONResponse, 0, len(users))
	for _, u := range users {
		out = append(out, api.Driver{Id: u.ID, DisplayName: u.DisplayName})
	}
	return out, nil
}

func (s *Server) CreateRoute(ctx context.Context, r api.CreateRouteRequestObject) (api.CreateRouteResponseObject, error) {
	rd, err := s.d.Svc.CreateRoute(ctx, service.RouteInput{RouteDate: r.Body.RouteDate, DriverUserID: r.Body.DriverUserId, TruckLabel: deref(r.Body.TruckLabel)})
	if err != nil {
		return nil, err
	}
	return api.CreateRoute201JSONResponse(toRoute(rd)), nil
}

func (s *Server) GetRoute(ctx context.Context, r api.GetRouteRequestObject) (api.GetRouteResponseObject, error) {
	rd, err := s.d.Svc.GetRoute(ctx, r.RouteId)
	if err != nil {
		return nil, err
	}
	return api.GetRoute200JSONResponse(toRoute(rd)), nil
}

func (s *Server) AddStop(ctx context.Context, r api.AddStopRequestObject) (api.AddStopResponseObject, error) {
	rd, err := s.d.Svc.AddStop(ctx, r.RouteId, r.Body.OrderId)
	if err != nil {
		return nil, err
	}
	return api.AddStop200JSONResponse(toRoute(rd)), nil
}

func (s *Server) RemoveStop(ctx context.Context, r api.RemoveStopRequestObject) (api.RemoveStopResponseObject, error) {
	rd, err := s.d.Svc.RemoveStop(ctx, r.RouteId, r.StopId)
	if err != nil {
		return nil, err
	}
	return api.RemoveStop200JSONResponse(toRoute(rd)), nil
}

func (s *Server) ReorderStops(ctx context.Context, r api.ReorderStopsRequestObject) (api.ReorderStopsResponseObject, error) {
	rd, err := s.d.Svc.ReorderStops(ctx, r.RouteId, r.Body.StopIds)
	if err != nil {
		return nil, err
	}
	return api.ReorderStops200JSONResponse(toRoute(rd)), nil
}

func (s *Server) RouteOut(ctx context.Context, r api.RouteOutRequestObject) (api.RouteOutResponseObject, error) {
	rd, err := s.d.Svc.RouteOut(ctx, r.RouteId)
	if err != nil {
		return nil, err
	}
	return api.RouteOut200JSONResponse(toRoute(rd)), nil
}

func (s *Server) RouteComplete(ctx context.Context, r api.RouteCompleteRequestObject) (api.RouteCompleteResponseObject, error) {
	rd, err := s.d.Svc.RouteComplete(ctx, r.RouteId)
	if err != nil {
		return nil, err
	}
	return api.RouteComplete200JSONResponse(toRoute(rd)), nil
}

// getStopProof streams the signature/photo for a stop (plain handler: binary response).
func (s *Server) getStopProof(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "stopId"), 10, 64)
	if err != nil {
		writeError(w, service.Invalid(map[string]string{"stopId": "must be an integer"}))
		return
	}
	st, err := s.d.Svc.Q.GetStop(r.Context(), id)
	if err != nil || !st.ProofRef.Valid || st.ProofType.String == "name" {
		writeError(w, service.NotFound("proof"))
		return
	}
	rc, ct, err := s.d.Proofs.Open(st.ProofRef.String)
	if err != nil {
		writeError(w, service.NotFound("proof"))
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", ct)
	io.Copy(w, rc)
}
