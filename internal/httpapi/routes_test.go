package httpapi

import "testing"

// confirmedOrder creates a confirmed order for the date with one brisket line (2 cases) and returns its id.
func confirmedOrder(t *testing.T, env *testEnv, cid, bid int64, date string) int64 {
	t.Helper()
	var o map[string]any
	env.do(t, env.office, "POST", "/api/office/orders", map[string]any{"customerId": cid, "requestedDeliveryDate": date}, &o)
	oid := int64(o["id"].(float64))
	env.do(t, env.office, "POST", "/api/office/orders/"+itoa(oid)+"/lines", map[string]any{"productId": bid, "orderedQty": 200}, &o)
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+itoa(oid)+"/confirm", nil, &o); code != 200 {
		t.Fatalf("confirm: %d %v", code, o)
	}
	return oid
}

func TestDayViewAndRoutes(t *testing.T) {
	env := newTestEnv(t)
	cid, bid, _ := seedCatalog(t, env)
	o1 := confirmedOrder(t, env, cid, bid, "2026-09-10")
	o2 := confirmedOrder(t, env, cid, bid, "2026-09-10")

	var day map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/day?date=2026-09-10", nil, &day); code != 200 || len(day["unscheduled"].([]any)) != 2 || len(day["routes"].([]any)) != 0 {
		t.Fatalf("day: %d %v", code, day)
	}
	var drivers []map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/drivers", nil, &drivers); code != 200 || len(drivers) != 1 {
		t.Fatalf("drivers: %d %v", code, drivers)
	}
	var r map[string]any
	if code := env.do(t, env.office, "POST", "/api/office/routes", map[string]any{"routeDate": "2026-09-10", "driverUserId": env.driverID, "truckLabel": "Reefer 1"}, &r); code != 201 || r["driverName"] != "Sam" {
		t.Fatalf("create route: %d %v", code, r)
	}
	rid := itoa(int64(r["id"].(float64)))
	env.do(t, env.office, "POST", "/api/office/routes/"+rid+"/stops", map[string]any{"orderId": o1}, &r)
	if code := env.do(t, env.office, "POST", "/api/office/routes/"+rid+"/stops", map[string]any{"orderId": o2}, &r); code != 200 || len(r["stops"].([]any)) != 2 {
		t.Fatalf("add stops: %d %v", code, r)
	}
	stops := r["stops"].([]any)
	s1 := itoa(int64(stops[0].(map[string]any)["id"].(float64)))
	s2 := itoa(int64(stops[1].(map[string]any)["id"].(float64)))
	if code := env.do(t, env.office, "PUT", "/api/office/routes/"+rid+"/sequence", map[string]any{"stopIds": []string{s2, s1}}, &r); code != 422 {
		t.Fatalf("sequence with strings must be 422: %d", code)
	}
	if code := env.do(t, env.office, "PUT", "/api/office/routes/"+rid+"/sequence", map[string]any{"stopIds": []int64{int64(stops[1].(map[string]any)["id"].(float64)), int64(stops[0].(map[string]any)["id"].(float64))}}, &r); code != 200 || r["stops"].([]any)[0].(map[string]any)["orderId"].(float64) != float64(o2) {
		t.Fatalf("reorder: %d %v", code, r)
	}
	if code := env.do(t, env.office, "DELETE", "/api/office/routes/"+rid+"/stops/"+s1, nil, &r); code != 200 || len(r["stops"].([]any)) != 1 {
		t.Fatalf("remove stop: %d %v", code, r)
	}
	env.do(t, env.office, "GET", "/api/office/day?date=2026-09-10", nil, &day)
	if len(day["unscheduled"].([]any)) != 1 || len(day["routes"].([]any)) != 1 {
		t.Fatalf("day after: %v", day)
	}
	if code := env.do(t, env.office, "POST", "/api/office/routes/"+rid+"/out", nil, &r); code != 200 || r["status"] != "out" {
		t.Fatalf("out: %d %v", code, r)
	}
	var e map[string]any
	if code := env.do(t, env.office, "POST", "/api/office/routes/"+rid+"/complete", nil, &e); code != 409 || e["code"] != "stops_pending" {
		t.Fatalf("complete with pending: %d %v", code, e)
	}
	if code := env.do(t, env.office, "GET", "/api/office/routes/"+rid, nil, &r); code != 200 || r["stops"].([]any)[0].(map[string]any)["customerName"] != "Blue Plate" {
		t.Fatalf("get route: %d %v", code, r)
	}
}
