// Package httpapi assembles the chi router: auth endpoints, the generated strict API, and the SPA.
package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
	"github.com/cmcgeedev/deepcutsCRM/internal/storage"
)

type Deps struct {
	Svc    *service.Service
	Auth   *auth.Auth
	Proofs *storage.Local
	Secure bool  // set Secure on cookies (TLS)
	Web    fs.FS // built SPA; index.html at the root
}

type Server struct{ d Deps }

var _ api.StrictServerInterface = (*Server)(nil)

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	// middleware.RealIP is deliberately omitted: it overwrites r.RemoteAddr from
	// X-Forwarded-For unconditionally, which would defeat auth.ClientIP's
	// trusted-proxy check and make the per-IP login limiter spoofable.
	r.Use(middleware.Logger, middleware.Recoverer, middleware.NoCache, middleware.RequestSize(1<<20))
	s := &Server{d: d}

	r.Post("/api/office/login", s.officeLogin)
	r.Post("/api/driver/login", s.driverLogin)
	r.Get("/api/driver/drivers", s.listDriverLoginNames)

	r.Group(func(g chi.Router) {
		g.Use(realmGuard(d.Auth))
		g.Post("/api/office/logout", s.logout)
		g.Post("/api/driver/logout", s.logout)
		g.Get("/api/office/stops/{stopId}/proof", s.getStopProof)
		strict := api.NewStrictHandlerWithOptions(s, nil, api.StrictHTTPServerOptions{
			RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				writeError(w, service.Invalid(map[string]string{"body": "malformed or oversized request body"}))
			},
			ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) { writeError(w, err) },
		})
		api.HandlerWithOptions(strict, api.ChiServerOptions{BaseRouter: g, ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeError(w, service.Invalid(map[string]string{"request": "invalid path or query parameter"}))
		}})
	})

	r.HandleFunc("/api/*", func(w http.ResponseWriter, r *http.Request) { writeError(w, service.NotFound("endpoint")) })
	r.NotFound(spaHandler(d.Web))
	return r
}

// realmGuard picks the realm from the path prefix and requires a matching session.
func realmGuard(a *auth.Auth) func(http.Handler) http.Handler {
	office := a.Middleware(auth.RealmOffice)
	driver := a.Middleware(auth.RealmDriver)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasPrefix(r.URL.Path, "/api/office/"):
				office(next).ServeHTTP(w, r)
			case strings.HasPrefix(r.URL.Path, "/api/driver/"):
				driver(next).ServeHTTP(w, r)
			default:
				writeError(w, service.NotFound("endpoint"))
			}
		})
	}
}

func writeError(w http.ResponseWriter, err error) {
	e, ok := service.AsError(err)
	if !ok {
		e = &service.Error{Status: http.StatusInternalServerError, Code: "internal", Message: "internal error"}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Status)
	body := api.Error{Code: e.Code, Message: e.Message}
	if len(e.Fields) > 0 {
		f := e.Fields
		body.Fields = &f
	}
	json.NewEncoder(w).Encode(body)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := dec.Decode(v); err != nil {
		return service.Invalid(map[string]string{"body": "malformed JSON"})
	}
	return nil
}

// spaHandler serves files from the embedded build and falls back to index.html for client routes.
func spaHandler(web fs.FS) http.HandlerFunc {
	files := http.FS(web)
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p != "" {
			if f, err := web.Open(p); err == nil {
				f.Close()
				http.FileServer(files).ServeHTTP(w, r)
				return
			}
		}
		idx, err := fs.ReadFile(web, "index.html")
		if err != nil {
			http.Error(w, "web app not built", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(idx)
	}
}

var errNoSession = errors.New("no session")

func sessionOf(r *http.Request) (auth.Session, error) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		return s, errNoSession
	}
	return s, nil
}
