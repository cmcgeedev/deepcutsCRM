package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
	"github.com/cmcgeedev/deepcutsCRM/internal/storage"
)

type testEnv struct {
	srv      *httptest.Server
	svc      *service.Service
	auth     *auth.Auth
	office   *http.Client
	driver   *http.Client
	anon     *http.Client
	driverID int64
}

func newTestEnv(t *testing.T) *testEnv {
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
	srv := httptest.NewServer(NewRouter(Deps{Svc: svc, Auth: a, Proofs: proofs, Secure: false, Web: web}))
	t.Cleanup(srv.Close)

	env := &testEnv{srv: srv, svc: svc, auth: a, office: jarClient(), driver: jarClient(), anon: jarClient()}
	ctx := context.Background()
	if _, err := a.CreateOfficeUser(ctx, "o@x.com", "Olive", "correct horse"); err != nil {
		t.Fatal(err)
	}
	drv, err := a.CreateDriver(ctx, "Sam", "123456")
	if err != nil {
		t.Fatal(err)
	}
	env.driverID = drv.ID
	if code := env.do(t, env.office, "POST", "/api/office/login", map[string]any{"email": "o@x.com", "password": "correct horse"}, nil); code != 200 {
		t.Fatalf("office login: %d", code)
	}
	if code := env.do(t, env.driver, "POST", "/api/driver/login", map[string]any{"userId": drv.ID, "pin": "123456"}, nil); code != 200 {
		t.Fatalf("driver login: %d", code)
	}
	return env
}

func jarClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

// do sends JSON and decodes the JSON reply into out (may be nil). Returns the status code.
func (e *testEnv) do(t *testing.T, c *http.Client, method, path string, body any, out any) int {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("decode %s %s: %v: %s", method, path, err, raw)
		}
	}
	return res.StatusCode
}
