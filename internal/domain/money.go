// Package domain holds business rules with no database or HTTP dependencies.
package domain

// Cents is a money amount in integer cents.
type Cents int64

// Hundredths is a quantity or weight in hundredths of a unit (1.00 lb == 100).
type Hundredths int64

// Extended multiplies a quantity in hundredths by a unit price in cents and rounds half-up to cents.
func Extended(qty Hundredths, unitPrice Cents) Cents {
	n := int64(qty) * int64(unitPrice)
	if n < 0 {
		return Cents(-((-n + 50) / 100))
	}
	return Cents((n + 50) / 100)
}

type Unit string

const (
	UnitLb   Unit = "lb"
	UnitCase Unit = "case"
	UnitEach Unit = "each"
)

func ParseUnit(s string) (Unit, bool) {
	switch Unit(s) {
	case UnitLb, UnitCase, UnitEach:
		return Unit(s), true
	}
	return "", false
}

type Product struct {
	ID               int64
	SKU              string
	Name             string
	SellUnit         Unit
	CatchWeight      bool
	ApproxCaseWeight Hundredths
	BasePrice        Cents
}

// Validate returns field-name → message for every invalid field; empty when valid.
func (p Product) Validate() map[string]string {
	errs := map[string]string{}
	if p.SKU == "" {
		errs["sku"] = "required"
	}
	if p.Name == "" {
		errs["name"] = "required"
	}
	if _, ok := ParseUnit(string(p.SellUnit)); !ok {
		errs["sellUnit"] = "must be lb, case or each"
	} else if p.CatchWeight && p.SellUnit != UnitCase {
		errs["sellUnit"] = "catch-weight products are sold by the case"
	}
	if p.CatchWeight && p.ApproxCaseWeight <= 0 {
		errs["approxCaseWeight"] = "required for catch-weight products"
	}
	if p.BasePrice < 0 {
		errs["basePriceCents"] = "must not be negative"
	}
	return errs
}
