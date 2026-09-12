package httpapi

import (
	"context"
	"net/http"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

// officeLoginRequest and driverLoginRequest mirror the OfficeLogin/DriverLogin
// schemas in api/openapi.yaml. oapi-codegen prunes schemas that are referenced
// only by an excluded operation's requestBody, so those two model types are not
// generated into internal/api; these local structs keep the same JSON shape.
type officeLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type driverLoginRequest struct {
	UserId int64  `json:"userId"`
	Pin    string `json:"pin"`
}

func (s *Server) officeLogin(w http.ResponseWriter, r *http.Request) {
	var in officeLoginRequest
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if in.Email == "" || in.Password == "" {
		writeError(w, service.Invalid(map[string]string{"email": "required", "password": "required"}))
		return
	}
	sess, err := s.d.Auth.LoginOffice(r.Context(), in.Email, in.Password, auth.ClientIP(r))
	if err != nil {
		writeError(w, err)
		return
	}
	auth.SetCookie(w, sess, s.d.Secure)
	writeJSON(w, http.StatusOK, toUser(sess))
}

func (s *Server) driverLogin(w http.ResponseWriter, r *http.Request) {
	var in driverLoginRequest
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if in.UserId == 0 || in.Pin == "" {
		writeError(w, service.Invalid(map[string]string{"userId": "required", "pin": "required"}))
		return
	}
	sess, err := s.d.Auth.LoginDriver(r.Context(), in.UserId, in.Pin, auth.ClientIP(r))
	if err != nil {
		writeError(w, err)
		return
	}
	auth.SetCookie(w, sess, s.d.Secure)
	writeJSON(w, http.StatusOK, toUser(sess))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if sess, err := sessionOf(r); err == nil {
		_ = s.d.Auth.Logout(r.Context(), sess.ID)
	}
	auth.ClearCookie(w, s.d.Secure)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listDriverLoginNames(w http.ResponseWriter, r *http.Request) {
	users, err := s.d.Svc.ListDrivers(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]api.Driver, 0, len(users))
	for _, u := range users {
		out = append(out, api.Driver{Id: u.ID, DisplayName: u.DisplayName})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) OfficeMe(ctx context.Context, _ api.OfficeMeRequestObject) (api.OfficeMeResponseObject, error) {
	sess, ok := auth.SessionFrom(ctx)
	if !ok {
		return nil, service.Unauthorized("login required")
	}
	return api.OfficeMe200JSONResponse(toUser(sess)), nil
}

func (s *Server) DriverMe(ctx context.Context, _ api.DriverMeRequestObject) (api.DriverMeResponseObject, error) {
	sess, ok := auth.SessionFrom(ctx)
	if !ok {
		return nil, service.Unauthorized("login required")
	}
	return api.DriverMe200JSONResponse(toUser(sess)), nil
}
