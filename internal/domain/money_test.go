package domain

import "testing"

func TestExtendedRoundsHalfUp(t *testing.T) {
	cases := []struct {
		qty   Hundredths
		price Cents
		want  Cents
	}{
		{100, 1000, 1000},  // 1.00 × $10.00
		{250, 1000, 2500},  // 2.50 × $10.00
		{333, 1000, 3330},  // 3.33 × $10.00
		{1, 1, 0},          // 0.01 × $0.01 = 0.0001 → 0
		{50, 1, 1},         // 0.50 × $0.01 = 0.005 → rounds up to 1
		{49, 1, 0},         // 0.49 × $0.01 = 0.0049 → 0
		{1234, 799, 9860},  // 12.34 × $7.99 = 98.5966 → 98.60
		{0, 999, 0},
	}
	for _, c := range cases {
		if got := Extended(c.qty, c.price); got != c.want {
			t.Errorf("Extended(%d,%d)=%d want %d", c.qty, c.price, got, c.want)
		}
	}
}

func TestParseUnit(t *testing.T) {
	for _, s := range []string{"lb", "case", "each"} {
		if _, ok := ParseUnit(s); !ok {
			t.Errorf("%q should parse", s)
		}
	}
	if _, ok := ParseUnit("kg"); ok {
		t.Error("kg should not parse")
	}
}

func TestProductValidate(t *testing.T) {
	good := Product{SKU: "BRIS-01", Name: "Brisket", SellUnit: UnitCase, CatchWeight: true, ApproxCaseWeight: 6000, BasePrice: 599}
	if errs := good.Validate(); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	bad := Product{SKU: "", Name: "", SellUnit: "kg", CatchWeight: true, ApproxCaseWeight: 0, BasePrice: -1}
	errs := bad.Validate()
	for _, f := range []string{"sku", "name", "sellUnit", "approxCaseWeight", "basePriceCents"} {
		if errs[f] == "" {
			t.Errorf("expected error for %s: %v", f, errs)
		}
	}
	cwLb := Product{SKU: "X", Name: "X", SellUnit: UnitLb, CatchWeight: true, ApproxCaseWeight: 100, BasePrice: 1}
	if cwLb.Validate()["sellUnit"] == "" {
		t.Error("catch-weight products must be sold by the case")
	}
}
