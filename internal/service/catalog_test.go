package service

import (
	"context"
	"testing"
)

var ctx = context.Background()

func TestCustomerCRUD(t *testing.T) {
	s := newTestService(t)
	c, err := s.CreateCustomer(ctx, CustomerInput{Name: "Blue Plate Diner", DeliveryDays: []string{"Mon", "thu"}, QBOCustomerID: "qbo-1", Active: true})
	mustNoErr(t, err)
	if c.DeliveryDays != "mon,thu" || c.QboCustomerID.String != "qbo-1" {
		t.Fatalf("bad row: %+v", c)
	}
	_, err = s.CreateCustomer(ctx, CustomerInput{Name: "", Active: true})
	wantCode(t, err, "invalid")
	_, err = s.CreateCustomer(ctx, CustomerInput{Name: "X", DeliveryDays: []string{"funday"}, Active: true})
	wantCode(t, err, "invalid")
	_, err = s.CreateCustomer(ctx, CustomerInput{Name: "Dup", QBOCustomerID: "qbo-1", Active: true})
	wantCode(t, err, "duplicate")

	u, err := s.UpdateCustomer(ctx, c.ID, CustomerInput{Name: "Blue Plate", Active: false})
	mustNoErr(t, err)
	if u.Name != "Blue Plate" || u.Active {
		t.Fatalf("update failed: %+v", u)
	}
	active, _ := s.ListCustomers(ctx, false)
	all, _ := s.ListCustomers(ctx, true)
	if len(active) != 0 || len(all) != 1 {
		t.Fatalf("list: active=%d all=%d", len(active), len(all))
	}
	_, err = s.GetCustomer(ctx, 999)
	wantCode(t, err, "not_found")
}

func TestProductCRUD(t *testing.T) {
	s := newTestService(t)
	w := int64(6000)
	p, err := s.CreateProduct(ctx, ProductInput{SKU: "BRIS", Name: "Brisket", SellUnit: "case", CatchWeight: true, ApproxCaseWeight: &w, BasePriceCents: 599, Active: true})
	mustNoErr(t, err)
	if !p.CatchWeight || p.ApproxCaseWeight.Int64 != 6000 {
		t.Fatalf("bad row: %+v", p)
	}
	_, err = s.CreateProduct(ctx, ProductInput{SKU: "BRIS", Name: "Dup", SellUnit: "each", BasePriceCents: 1, Active: true})
	wantCode(t, err, "duplicate")
	_, err = s.CreateProduct(ctx, ProductInput{SKU: "BAD", Name: "Bad", SellUnit: "lb", CatchWeight: true, ApproxCaseWeight: &w, BasePriceCents: 1, Active: true})
	wantCode(t, err, "invalid")
	u, err := s.UpdateProduct(ctx, p.ID, ProductInput{SKU: "BRIS", Name: "Whole Brisket", SellUnit: "case", CatchWeight: true, ApproxCaseWeight: &w, BasePriceCents: 649, Active: true})
	mustNoErr(t, err)
	if u.Name != "Whole Brisket" || u.BasePriceCents != 649 {
		t.Fatalf("update failed: %+v", u)
	}
}

func TestCustomerPrices(t *testing.T) {
	s := newTestService(t)
	c, _ := s.CreateCustomer(ctx, CustomerInput{Name: "A", Active: true})
	p, _ := s.CreateProduct(ctx, ProductInput{SKU: "S1", Name: "Sausage", SellUnit: "each", BasePriceCents: 400, Active: true})
	_, err := s.SetCustomerPrice(ctx, c.ID, PriceInput{ProductID: p.ID, PriceCents: 350, EffectiveFrom: "2026-01-01"})
	mustNoErr(t, err)
	_, err = s.SetCustomerPrice(ctx, c.ID, PriceInput{ProductID: p.ID, PriceCents: 375, EffectiveFrom: "2026-10-01"})
	mustNoErr(t, err)
	_, err = s.SetCustomerPrice(ctx, c.ID, PriceInput{ProductID: p.ID, PriceCents: 1, EffectiveFrom: "not-a-date"})
	wantCode(t, err, "invalid")
	_, err = s.SetCustomerPrice(ctx, c.ID, PriceInput{ProductID: 999, PriceCents: 1, EffectiveFrom: "2026-01-01"})
	wantCode(t, err, "not_found")

	rows, err := s.ListCustomerPrices(ctx, c.ID, "2026-09-10")
	mustNoErr(t, err)
	if len(rows) != 1 || rows[0].PriceCents != 350 {
		t.Fatalf("as of sept: %+v", rows)
	}
	rows, _ = s.ListCustomerPrices(ctx, c.ID, "2026-10-01")
	if len(rows) != 1 || rows[0].PriceCents != 375 {
		t.Fatalf("as of oct: %+v", rows)
	}
	price, custom, err := s.resolvePrice(ctx, s.Q, c.ID, p.ID, 400, "2025-06-01")
	mustNoErr(t, err)
	if price != 400 || custom {
		t.Fatalf("fallback to base: %d %v", price, custom)
	}
	price, custom, _ = s.resolvePrice(ctx, s.Q, c.ID, p.ID, 400, "2026-12-01")
	if price != 375 || !custom {
		t.Fatalf("latest custom: %d %v", price, custom)
	}
}
