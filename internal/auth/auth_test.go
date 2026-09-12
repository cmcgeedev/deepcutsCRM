package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
)

var ctx = context.Background()

func newAuth(t *testing.T) *Auth {
	t.Helper()
	d, err := db.OpenAndMigrate(ctx, filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	a := New(queries.New(d))
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	a.Now = func() time.Time { return now }
	return a
}

func TestOfficeLogin(t *testing.T) {
	a := newAuth(t)
	_, err := a.CreateOfficeUser(ctx, "o@x.com", "Olive", "short")
	if err == nil {
		t.Fatal("password under 8 chars must be rejected")
	}
	u, err := a.CreateOfficeUser(ctx, "o@x.com", "Olive", "correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.CreateOfficeUser(ctx, "o@x.com", "Dup", "correct horse"); err == nil {
		t.Fatal("duplicate email must fail")
	}
	if _, err := a.LoginOffice(ctx, "o@x.com", "wrong", "1.1.1.1"); err == nil {
		t.Fatal("wrong password must fail")
	}
	s, err := a.LoginOffice(ctx, "O@X.com", "correct horse", "1.1.1.1")
	if err != nil || s.UserID != u.ID || s.Realm != "office" || s.DisplayName != "Olive" {
		t.Fatalf("login: %v %+v", err, s)
	}
	got, ok := a.Lookup(ctx, s.ID)
	if !ok || got.UserID != u.ID {
		t.Fatal("lookup failed")
	}
	if err := a.Logout(ctx, s.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.Lookup(ctx, s.ID); ok {
		t.Fatal("session should be gone")
	}
}

func TestDriverPinAndRateLimit(t *testing.T) {
	a := newAuth(t)
	if _, err := a.CreateDriver(ctx, "Sam", "12345"); err == nil {
		t.Fatal("5 digit pin must be rejected")
	}
	u, err := a.CreateDriver(ctx, "Sam", "123456")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := a.LoginDriver(ctx, u.ID, "000000", "9.9.9.9"); err == nil {
			t.Fatal("wrong pin must fail")
		}
	}
	if _, err := a.LoginDriver(ctx, u.ID, "123456", "9.9.9.9"); err == nil {
		t.Fatal("11th attempt must be rate limited even with the right pin")
	}
	a.Limiter.Reset("user:" + itoa(u.ID))
	a.Limiter.Reset("ip:9.9.9.9")
	s, err := a.LoginDriver(ctx, u.ID, "123456", "9.9.9.9")
	if err != nil || s.Realm != "driver" {
		t.Fatalf("login after reset: %v", err)
	}
	if err := a.SetDriverPIN(ctx, "Sam", "654321"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.LoginDriver(ctx, u.ID, "123456", "8.8.8.8"); err == nil {
		t.Fatal("old pin must fail after reset")
	}
	if err := a.DeactivateUser(ctx, "Sam"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.LoginDriver(ctx, u.ID, "654321", "7.7.7.7"); err == nil {
		t.Fatal("inactive user must not log in")
	}
}

func TestSessionExpiry(t *testing.T) {
	a := newAuth(t)
	u, _ := a.CreateDriver(ctx, "Sam", "123456")
	s, _ := a.LoginDriver(ctx, u.ID, "123456", "1.1.1.1")
	a.Now = func() time.Time { return time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) }
	if _, ok := a.Lookup(ctx, s.ID); ok {
		t.Fatal("expired session must not resolve")
	}
}

func TestMiddlewareRealms(t *testing.T) {
	a := newAuth(t)
	u, _ := a.CreateDriver(ctx, "Sam", "123456")
	s, _ := a.LoginDriver(ctx, u.ID, "123456", "1.1.1.1")
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, _ := SessionFrom(r.Context())
		w.Write([]byte(sess.DisplayName))
	})
	call := func(realm, cookie string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/", nil)
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: CookieName, Value: cookie})
		}
		rec := httptest.NewRecorder()
		a.Middleware(realm)(ok).ServeHTTP(rec, req)
		return rec
	}
	if rec := call("driver", s.ID); rec.Code != 200 || rec.Body.String() != "Sam" {
		t.Fatalf("driver realm: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call("office", s.ID); rec.Code != 401 {
		t.Fatalf("cross realm must be 401, got %d", rec.Code)
	}
	if rec := call("driver", ""); rec.Code != 401 {
		t.Fatalf("no cookie must be 401, got %d", rec.Code)
	}
	if rec := call("driver", "garbage"); rec.Code != 401 {
		t.Fatalf("bad cookie must be 401, got %d", rec.Code)
	}
}

func itoa(i int64) string { return strconv.FormatInt(i, 10) }
