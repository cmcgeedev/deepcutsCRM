package httpapi

import "testing"

func TestCustomerEndpoints(t *testing.T) {
	env := newTestEnv(t)
	var c map[string]any
	code := env.do(t, env.office, "POST", "/api/office/customers", map[string]any{
		"name": "Blue Plate", "deliveryAddress": "1 Main St", "deliveryDays": []string{"mon", "thu"}, "qboCustomerId": "q1",
	}, &c)
	if code != 201 || c["name"] != "Blue Plate" || c["active"] != true || len(c["deliveryDays"].([]any)) != 2 {
		t.Fatalf("create: %d %v", code, c)
	}
	id := int64(c["id"].(float64))
	var list []map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/customers", nil, &list); code != 200 || len(list) != 1 {
		t.Fatalf("list: %d %v", code, list)
	}
	code = env.do(t, env.office, "PUT", "/api/office/customers/"+itoa(id), map[string]any{"name": "Blue Plate Diner", "active": false, "qboCustomerId": "q1"}, &c)
	if code != 200 || c["name"] != "Blue Plate Diner" || c["active"] != false {
		t.Fatalf("update: %d %v", code, c)
	}
	if code := env.do(t, env.office, "GET", "/api/office/customers", nil, &list); code != 200 || len(list) != 0 {
		t.Fatalf("inactive hidden: %d %v", code, list)
	}
	if code := env.do(t, env.office, "GET", "/api/office/customers?includeInactive=true", nil, &list); code != 200 || len(list) != 1 {
		t.Fatalf("includeInactive: %d %v", code, list)
	}
	var e map[string]any
	if code := env.do(t, env.office, "POST", "/api/office/customers", map[string]any{"name": "Dup", "qboCustomerId": "q1"}, &e); code != 409 || e["code"] != "duplicate" {
		t.Fatalf("duplicate: %d %v", code, e)
	}
}

func TestProductAndPriceEndpoints(t *testing.T) {
	env := newTestEnv(t)
	var p map[string]any
	code := env.do(t, env.office, "POST", "/api/office/products", map[string]any{
		"sku": "BRIS", "name": "Brisket", "sellUnit": "case", "catchWeight": true, "approxCaseWeight": 6000, "basePriceCents": 599,
	}, &p)
	if code != 201 || p["catchWeight"] != true || p["approxCaseWeight"].(float64) != 6000 {
		t.Fatalf("create product: %d %v", code, p)
	}
	pid := int64(p["id"].(float64))
	var e map[string]any
	if code := env.do(t, env.office, "POST", "/api/office/products", map[string]any{"sku": "X", "name": "X", "sellUnit": "lb", "catchWeight": true, "basePriceCents": 1}, &e); code != 422 {
		t.Fatalf("invalid product: %d %v", code, e)
	}
	var c map[string]any
	env.do(t, env.office, "POST", "/api/office/customers", map[string]any{"name": "A"}, &c)
	cid := int64(c["id"].(float64))
	var pr map[string]any
	code = env.do(t, env.office, "POST", "/api/office/customers/"+itoa(cid)+"/prices", map[string]any{"productId": pid, "priceCents": 549, "effectiveFrom": "2026-01-01"}, &pr)
	if code != 201 || pr["sku"] != "BRIS" || pr["priceCents"].(float64) != 549 {
		t.Fatalf("set price: %d %v", code, pr)
	}
	var prices []map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/customers/"+itoa(cid)+"/prices", nil, &prices); code != 200 || len(prices) != 1 {
		t.Fatalf("list prices: %d %v", code, prices)
	}
	if code := env.do(t, env.office, "GET", "/api/office/customers/"+itoa(cid)+"/prices?asOf=2025-01-01", nil, &prices); code != 200 || len(prices) != 0 {
		t.Fatalf("list prices before effective: %d %v", code, prices)
	}
}
