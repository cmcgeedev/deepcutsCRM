package httpapi

import (
	"encoding/base64"
	"io"
	"testing"
)

const cidA = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const cidB = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"

var pngBytes = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0}

// outRoute schedules one confirmed order on a route for today and marks it out. Returns (routeId, stopId, orderId).
func outRoute(t *testing.T, env *testEnv) (string, string, int64) {
	t.Helper()
	cid, bid, _ := seedCatalog(t, env)
	oid := confirmedOrder(t, env, cid, bid, "2026-09-10")
	var r map[string]any
	env.do(t, env.office, "POST", "/api/office/routes", map[string]any{"routeDate": "2026-09-10", "driverUserId": env.driverID}, &r)
	rid := itoa(int64(r["id"].(float64)))
	env.do(t, env.office, "POST", "/api/office/routes/"+rid+"/stops", map[string]any{"orderId": oid}, &r)
	var o map[string]any
	env.do(t, env.office, "GET", "/api/office/orders/"+itoa(oid), nil, &o)
	lid := itoa(int64(o["lines"].([]any)[0].(map[string]any)["id"].(float64)))
	env.do(t, env.office, "PATCH", "/api/office/orders/"+itoa(oid)+"/lines/"+lid, map[string]any{"shippedWeight": 11850}, nil)
	if code := env.do(t, env.office, "POST", "/api/office/routes/"+rid+"/out", nil, &r); code != 200 {
		t.Fatalf("out: %d %v", code, r)
	}
	sid := itoa(int64(r["stops"].([]any)[0].(map[string]any)["id"].(float64)))
	return rid, sid, oid
}

func TestDriverRouteAndActions(t *testing.T) {
	env := newTestEnv(t)
	rid, sid, oid := outRoute(t, env)
	var dr map[string]any
	if code := env.do(t, env.driver, "GET", "/api/driver/route", nil, &dr); code != 200 || len(dr["stops"].([]any)) != 1 {
		t.Fatalf("driver route: %d %v", code, dr)
	}
	stop := dr["stops"].([]any)[0].(map[string]any)
	if stop["stop"].(map[string]any)["customerName"] != "Blue Plate" || len(stop["order"].(map[string]any)["lines"].([]any)) != 1 {
		t.Fatalf("driver stop shape: %v", stop)
	}
	if code := env.do(t, env.driver, "GET", "/api/driver/route?date=2026-01-01", nil, nil); code != 404 {
		t.Fatalf("no route that day: %d", code)
	}
	lineID := int64(stop["order"].(map[string]any)["lines"].([]any)[0].(map[string]any)["id"].(float64))

	var res map[string]any
	code := env.do(t, env.driver, "POST", "/api/driver/stops/"+sid+"/actions", map[string]any{
		"clientId": cidA, "type": "adjust", "lines": []map[string]any{{"lineId": lineID, "deliveredWeight": 6000, "shortageNote": "one case rejected"}},
	}, &res)
	if code != 200 || res["applied"] != true {
		t.Fatalf("adjust: %d %v", code, res)
	}
	code = env.do(t, env.driver, "POST", "/api/driver/stops/"+sid+"/actions", map[string]any{
		"clientId": cidB, "type": "deliver", "proof": map[string]any{"type": "signature", "dataBase64": base64.StdEncoding.EncodeToString(pngBytes)}, "note": "left at dock",
	}, &res)
	st := res["stop"].(map[string]any)["stop"].(map[string]any)
	if code != 200 || res["applied"] != true || st["status"] != "delivered" || st["hasProofImage"] != true || st["proofType"] != "signature" {
		t.Fatalf("deliver: %d %v", code, res)
	}
	code = env.do(t, env.driver, "POST", "/api/driver/stops/"+sid+"/actions", map[string]any{
		"clientId": cidB, "type": "deliver", "proof": map[string]any{"type": "name", "name": "Pat"},
	}, &res)
	if code != 200 || res["applied"] != false {
		t.Fatalf("replay must be 200 applied=false: %d %v", code, res)
	}
	var e map[string]any
	if code := env.do(t, env.driver, "POST", "/api/driver/stops/"+sid+"/actions", map[string]any{"clientId": "nope", "type": "skip", "skipReason": "x"}, &e); code != 422 {
		t.Fatalf("bad uuid: %d %v", code, e)
	}
	// office sees the proof image and the review flag
	res2, err := env.office.Get(env.srv.URL + "/api/office/stops/" + sid + "/proof")
	if err != nil || res2.StatusCode != 200 || res2.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("proof: %v %d", err, res2.StatusCode)
	}
	b, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	if string(b) != string(pngBytes) {
		t.Fatal("proof bytes differ")
	}
	var o map[string]any
	env.do(t, env.office, "GET", "/api/office/orders/"+itoa(oid), nil, &o)
	if o["status"] != "delivered" || o["needsReview"] != true || o["lines"].([]any)[0].(map[string]any)["deliveredWeight"].(float64) != 6000 {
		t.Fatalf("order after delivery: %v", o)
	}
	if code := env.do(t, env.driver, "POST", "/api/driver/routes/"+rid+"/complete", nil, &dr); code != 200 || dr["route"].(map[string]any)["status"] != "complete" {
		t.Fatalf("complete: %d %v", code, dr)
	}
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+itoa(oid)+"/finalize", nil, &o); code != 200 || o["status"] != "finalized" || o["needsReview"] != false {
		t.Fatalf("finalize: %d %v", code, o)
	}
}

func TestDriverProofTooLarge(t *testing.T) {
	env := newTestEnv(t)
	_, sid, _ := outRoute(t, env)
	big := make([]byte, 301*1024)
	copy(big, pngBytes)
	var e map[string]any
	code := env.do(t, env.driver, "POST", "/api/driver/stops/"+sid+"/actions", map[string]any{
		"clientId": cidA, "type": "deliver", "proof": map[string]any{"type": "photo", "dataBase64": base64.StdEncoding.EncodeToString(big)},
	}, &e)
	if code != 422 || e["fields"].(map[string]any)["proof"] == nil {
		t.Fatalf("oversized proof: %d %v", code, e)
	}
}
