package auth

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"
)

const CookieName = "deepcuts_session"

type ctxKey struct{}

func SessionFrom(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(ctxKey{}).(Session)
	return s, ok
}

// Middleware requires a valid session of the given realm and stores it in the request context.
func (a *Auth) Middleware(realm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(CookieName)
			if err != nil {
				unauthorized(w)
				return
			}
			s, ok := a.Lookup(r.Context(), c.Value)
			if !ok || s.Realm != realm {
				unauthorized(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, s)))
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"code": "unauthorized", "message": "login required"})
}

func SetCookie(w http.ResponseWriter, s Session, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: s.ID, Path: "/", HttpOnly: true, Secure: secure,
		SameSite: http.SameSiteLaxMode, Expires: s.ExpiresAt,
	})
}

func ClearCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: time.Unix(0, 0), MaxAge: -1})
}

// ClientIP returns the remote IP without port. Behind Caddy, X-Forwarded-For's first entry wins.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := indexByte(xff, ','); i > 0 {
			return xff[:i]
		}
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
