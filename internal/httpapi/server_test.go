package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
	"github.com/cmcgeedev/deepcutsCRM/internal/storage"
)

func TestRealmGuards(t *testing.T) {
	env := newTestEnv(t)
	var me map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/me", nil, &me); code != 200 || me["displayName"] != "Olive" || me["realm"] != "office" {
		t.Fatalf("office me: %d %v", code, me)
	}
	if code := env.do(t, env.driver, "GET", "/api/driver/me", nil, &me); code != 200 || me["realm"] != "driver" {
		t.Fatalf("driver me: %d %v", code, me)
	}
	if code := env.do(t, env.driver, "GET", "/api/office/me", nil, nil); code != 401 {
		t.Fatalf("driver on office path: %d", code)
	}
	if code := env.do(t, env.office, "GET", "/api/driver/route", nil, nil); code != 401 {
		t.Fatalf("office on driver path: %d", code)
	}
	if code := env.do(t, env.anon, "GET", "/api/office/customers", nil, nil); code != 401 {
		t.Fatalf("anon: %d", code)
	}
	if code := env.do(t, env.anon, "GET", "/api/customer/anything", nil, nil); code != 404 {
		t.Fatalf("customer realm reserved: %d", code)
	}
}

func TestLoginErrorsAndLogout(t *testing.T) {
	env := newTestEnv(t)
	var e map[string]any
	if code := env.do(t, env.anon, "POST", "/api/office/login", map[string]any{"email": "o@x.com", "password": "nope"}, &e); code != 401 || e["code"] != "unauthorized" {
		t.Fatalf("bad password: %d %v", code, e)
	}
	if code := env.do(t, env.anon, "POST", "/api/office/login", map[string]any{"email": "o@x.com"}, &e); code != 422 || e["code"] != "invalid" {
		t.Fatalf("missing field: %d %v", code, e)
	}
	var drivers []map[string]any
	if code := env.do(t, env.anon, "GET", "/api/driver/drivers", nil, &drivers); code != 200 || len(drivers) != 1 || drivers[0]["displayName"] != "Sam" {
		t.Fatalf("driver names: %d %v", code, drivers)
	}
	if code := env.do(t, env.office, "POST", "/api/office/logout", nil, nil); code != 204 {
		t.Fatalf("logout: %d", code)
	}
	if code := env.do(t, env.office, "GET", "/api/office/me", nil, nil); code != 401 {
		t.Fatalf("after logout: %d", code)
	}
}

func TestSPAFallbackAndErrorShape(t *testing.T) {
	env := newTestEnv(t)
	res, err := env.anon.Get(env.srv.URL + "/office/orders/12")
	if err != nil || res.StatusCode != 200 || res.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("spa fallback: %v %d %s", err, res.StatusCode, res.Header.Get("Content-Type"))
	}
	var e map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/customers/999", nil, &e); code != 404 || e["code"] != "not_found" {
		t.Fatalf("404 shape: %d %v", code, e)
	}
	if code := env.do(t, env.office, "POST", "/api/office/customers", map[string]any{"name": ""}, &e); code != 422 || e["fields"].(map[string]any)["name"] == nil {
		t.Fatalf("422 shape: %d %v", code, e)
	}
}

// newSecureTestEnv is like newTestEnv but sets Deps.Secure: true, to check the
// cookie flags NewRouter applies when the app runs behind TLS.
func newSecureTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dir := t.TempDir()
	d, err := db.OpenAndMigrate(context.Background(), filepath.Join(dir, "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	loc, _ := time.LoadLocation("America/New_York")
	svc := service.New(d, loc)
	svc.Now = func() time.Time { return time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC) }
	proofs, _ := storage.NewLocal(filepath.Join(dir, "uploads"))
	svc.Proofs = proofs
	a := auth.New(queries.New(d))
	a.Now = svc.Now
	web := fstest.MapFS{"index.html": {Data: []byte("<html>spa</html>")}}
	srv := httptest.NewServer(NewRouter(Deps{Svc: svc, Auth: a, Proofs: proofs, Secure: true, Web: web}))
	t.Cleanup(srv.Close)

	env := &testEnv{srv: srv, svc: svc, auth: a, office: jarClient(), driver: jarClient(), anon: jarClient()}
	ctx := context.Background()
	if _, err := a.CreateOfficeUser(ctx, "o@x.com", "Olive", "correct horse"); err != nil {
		t.Fatal(err)
	}
	return env
}

func TestCookieFlags(t *testing.T) {
	env := newSecureTestEnv(t)
	req, _ := http.NewRequest("POST", env.srv.URL+"/api/office/login", bytes.NewReader([]byte(`{"email":"o@x.com","password":"correct horse"}`)))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("login: %d", res.StatusCode)
	}
	var found *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == auth.CookieName {
			found = c
		}
	}
	if found == nil {
		t.Fatal("no session cookie set")
	}
	if !found.HttpOnly || !found.Secure || found.SameSite != http.SameSiteLaxMode || found.Path != "/" {
		t.Fatalf("cookie flags: HttpOnly=%v Secure=%v SameSite=%v Path=%q", found.HttpOnly, found.Secure, found.SameSite, found.Path)
	}
}

func TestOversizedBodyRejected(t *testing.T) {
	env := newTestEnv(t)
	big := make([]byte, 2<<20)
	for i := range big {
		big[i] = 'a'
	}
	body := []byte(`{"email":"` + string(big) + `","password":"x"}`)
	req, _ := http.NewRequest("POST", env.srv.URL+"/api/office/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := env.anon.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 422 {
		t.Fatalf("oversized body: %d", res.StatusCode)
	}
	var e map[string]any
	if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
		t.Fatal(err)
	}
	if e["code"] != "invalid" {
		t.Fatalf("oversized body code: %v", e)
	}
}

func TestInternalErrorShape(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, errors.New("boom"))
	if rec.Code != 500 {
		t.Fatalf("status: %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "boom") {
		t.Fatalf("body leaked internal error text: %s", body)
	}
	var e map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatal(err)
	}
	if e["code"] != "internal" {
		t.Fatalf("code: %v", e)
	}
}
