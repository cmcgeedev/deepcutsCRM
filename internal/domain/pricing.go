package domain

// PriceRow is one negotiated customer price with the date it takes effect (YYYY-MM-DD).
type PriceRow struct {
	Price         Cents
	EffectiveFrom string
}

// ResolvePrice picks the customer price with the latest EffectiveFrom on or before orderDate,
// falling back to base. Dates compare lexically because they are YYYY-MM-DD.
func ResolvePrice(base Cents, rows []PriceRow, orderDate string) (Cents, bool) {
	best := -1
	for i, r := range rows {
		if r.EffectiveFrom > orderDate {
			continue
		}
		if best == -1 || r.EffectiveFrom > rows[best].EffectiveFrom {
			best = i
		}
	}
	if best == -1 {
		return base, false
	}
	return rows[best].Price, true
}

// EstimateWeight returns ordered cases × approximate case weight for catch-weight products.
func EstimateWeight(p Product, orderedQty Hundredths) (Hundredths, bool) {
	if !p.CatchWeight {
		return 0, false
	}
	return Hundredths(int64(orderedQty) * int64(p.ApproxCaseWeight) / 100), true
}

type Line struct {
	Product         Product
	OrderedQty      Hundredths
	UnitPrice       Cents
	EstWeight       *Hundredths
	ShippedWeight   *Hundredths
	DeliveredQty    *Hundredths
	DeliveredWeight *Hundredths
}

// BillableQty returns the quantity the amount is computed from and where it came from.
// Catch-weight: delivered weight, else shipped, else estimated. Otherwise: delivered qty, else ordered.
func (l Line) BillableQty() (Hundredths, string) {
	if l.Product.CatchWeight {
		switch {
		case l.DeliveredWeight != nil:
			return *l.DeliveredWeight, "delivered"
		case l.ShippedWeight != nil:
			return *l.ShippedWeight, "shipped"
		case l.EstWeight != nil:
			return *l.EstWeight, "estimated"
		default:
			return 0, "estimated"
		}
	}
	if l.DeliveredQty != nil {
		return *l.DeliveredQty, "delivered"
	}
	return l.OrderedQty, "ordered"
}

func LineAmount(l Line) Cents {
	q, _ := l.BillableQty()
	return Extended(q, l.UnitPrice)
}
