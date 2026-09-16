package domain

import "testing"

func TestResolvePrice(t *testing.T) {
	rows := []PriceRow{
		{Price: 500, EffectiveFrom: "2026-01-01"},
		{Price: 550, EffectiveFrom: "2026-06-01"},
		{Price: 600, EffectiveFrom: "2026-12-01"},
	}
	if p, ok := ResolvePrice(700, nil, "2026-09-10"); p != 700 || ok {
		t.Errorf("no rows: got %d %v", p, ok)
	}
	if p, ok := ResolvePrice(700, rows, "2025-12-31"); p != 700 || ok {
		t.Errorf("before all rows: got %d %v", p, ok)
	}
	if p, ok := ResolvePrice(700, rows, "2026-06-01"); p != 550 || !ok {
		t.Errorf("on effective date: got %d %v", p, ok)
	}
	if p, ok := ResolvePrice(700, rows, "2026-09-10"); p != 550 || !ok {
		t.Errorf("between: got %d %v", p, ok)
	}
	if p, _ := ResolvePrice(700, rows, "2027-01-01"); p != 600 {
		t.Errorf("latest: got %d", p)
	}
}

func TestEstimateWeight(t *testing.T) {
	cw := Product{SellUnit: UnitCase, CatchWeight: true, ApproxCaseWeight: 6000} // 60 lb cases
	if w, ok := EstimateWeight(cw, 300); !ok || w != 18000 {
		t.Errorf("3 cases × 60 lb: got %d %v", w, ok)
	}
	if w, ok := EstimateWeight(cw, 150); !ok || w != 9000 {
		t.Errorf("1.5 cases: got %d %v", w, ok)
	}
	if _, ok := EstimateWeight(Product{SellUnit: UnitEach}, 100); ok {
		t.Error("non catch-weight must return false")
	}
}

func h(v int64) *Hundredths { x := Hundredths(v); return &x }

func TestLineAmountCatchWeight(t *testing.T) {
	p := Product{SellUnit: UnitCase, CatchWeight: true, ApproxCaseWeight: 6000, BasePrice: 599}
	l := Line{Product: p, OrderedQty: 200, UnitPrice: 599, EstWeight: h(12000)}
	if got := LineAmount(l); got != 71880 { // 120.00 lb × $5.99
		t.Errorf("estimated: %d", got)
	}
	if _, src := l.BillableQty(); src != "estimated" {
		t.Errorf("src=%s", src)
	}
	l.ShippedWeight = h(11850)
	if got := LineAmount(l); got != 70982 { // 118.50 × 5.99 = 709.815 → 709.82
		t.Errorf("shipped: %d", got)
	}
	l.DeliveredWeight = h(6000)
	if got := LineAmount(l); got != 35940 {
		t.Errorf("delivered: %d", got)
	}
	if _, src := l.BillableQty(); src != "delivered" {
		t.Errorf("src=%s", src)
	}
}

func TestLineAmountUnitPriced(t *testing.T) {
	p := Product{SellUnit: UnitEach, BasePrice: 1250}
	l := Line{Product: p, OrderedQty: 400, UnitPrice: 1250}
	if got := LineAmount(l); got != 5000 {
		t.Errorf("ordered: %d", got)
	}
	l.DeliveredQty = h(300)
	if got := LineAmount(l); got != 3750 {
		t.Errorf("delivered: %d", got)
	}
	l.DeliveredQty = h(0)
	if got := LineAmount(l); got != 0 {
		t.Errorf("rejected line must be zero, got %d", got)
	}
}
