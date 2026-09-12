package httpapi

import "testing"

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
