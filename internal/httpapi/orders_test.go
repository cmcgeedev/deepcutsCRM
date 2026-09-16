package httpapi

import "testing"

// seedCatalog creates a customer with a custom brisket price and returns (customerId, brisketId, sausageId).
func seedCatalog(t *testing.T, env *testEnv) (int64, int64, int64) {
	t.Helper()
	var c, b, s map[string]any
	env.do(t, env.office, "POST", "/api/office/customers", map[string]any{"name": "Blue Plate", "deliveryAddress": "1 Main St"}, &c)
	env.do(t, env.office, "POST", "/api/office/products", map[string]any{"sku": "BRIS", "name": "Brisket", "sellUnit": "case", "catchWeight": true, "approxCaseWeight": 6000, "basePriceCents": 599}, &b)
	env.do(t, env.office, "POST", "/api/office/products", map[string]any{"sku": "SAUS", "name": "Sausage", "sellUnit": "each", "basePriceCents": 400}, &s)
	cid, bid, sid := int64(c["id"].(float64)), int64(b["id"].(float64)), int64(s["id"].(float64))
	env.do(t, env.office, "POST", "/api/office/customers/"+itoa(cid)+"/prices", map[string]any{"productId": bid, "priceCents": 549, "effectiveFrom": "2026-01-01"}, nil)
	return cid, bid, sid
}

func TestOrderEndpoints(t *testing.T) {
	env := newTestEnv(t)
	cid, bid, sid := seedCatalog(t, env)
	var o map[string]any
	code := env.do(t, env.office, "POST", "/api/office/orders", map[string]any{"customerId": cid, "requestedDeliveryDate": "2026-09-12", "notes": "back door"}, &o)
	if code != 201 || o["status"] != "draft" || o["customer"].(map[string]any)["name"] != "Blue Plate" {
		t.Fatalf("create: %d %v", code, o)
	}
	oid := itoa(int64(o["id"].(float64)))
	code = env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/lines", map[string]any{"productId": bid, "orderedQty": 200}, &o)
	lines := o["lines"].([]any)
	l0 := lines[0].(map[string]any)
	if code != 200 || len(lines) != 1 || l0["unitPriceCents"].(float64) != 549 || l0["estWeight"].(float64) != 12000 || l0["amountSource"] != "estimated" || o["totalCents"].(float64) != 65880 {
		t.Fatalf("add line: %d %v", code, o)
	}
	env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/lines", map[string]any{"productId": sid, "orderedQty": 1000}, &o)
	lid := itoa(int64(l0["id"].(float64)))
	code = env.do(t, env.office, "PATCH", "/api/office/orders/"+oid+"/lines/"+lid, map[string]any{"shippedWeight": 11850, "unitPriceCents": 500}, &o)
	l0 = o["lines"].([]any)[0].(map[string]any)
	if code != 200 || l0["priceOverridden"] != true || l0["amountSource"] != "shipped" || l0["amountCents"].(float64) != 59250 {
		t.Fatalf("patch line: %d %v", code, l0)
	}
	var e map[string]any
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/finalize", nil, &e); code != 409 || e["code"] != "invalid_transition" {
		t.Fatalf("finalize draft: %d %v", code, e)
	}
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/confirm", nil, &o); code != 200 || o["status"] != "confirmed" {
		t.Fatalf("confirm: %d %v", code, o)
	}
	var list []map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/orders?status=confirmed", nil, &list); code != 200 || len(list) != 1 || list[0]["lineCount"].(float64) != 2 {
		t.Fatalf("list: %d %v", code, list)
	}
	if code := env.do(t, env.office, "PATCH", "/api/office/orders/"+oid, map[string]any{"notes": "side door"}, &o); code != 200 || o["notes"] != "side door" {
		t.Fatalf("patch order: %d %v", code, o)
	}
	if code := env.do(t, env.office, "DELETE", "/api/office/orders/"+oid+"/lines/"+lid, nil, &o); code != 200 || len(o["lines"].([]any)) != 1 {
		t.Fatalf("delete line: %d %v", code, o)
	}
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/unconfirm", nil, &o); code != 200 || o["status"] != "draft" {
		t.Fatalf("unconfirm: %d %v", code, o)
	}
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/cancel", nil, &o); code != 200 || o["status"] != "cancelled" {
		t.Fatalf("cancel: %d %v", code, o)
	}
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/lines", map[string]any{"productId": sid, "orderedQty": 100}, &e); code != 409 || e["code"] != "locked" {
		t.Fatalf("locked: %d %v", code, e)
	}
}
