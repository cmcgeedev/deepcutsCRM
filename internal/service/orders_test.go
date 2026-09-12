package service

import "testing"

type fixture struct {
	s        *Service
	customer int64
	brisket  int64 // catch-weight, 60 lb cases, base $5.99/lb, customer price $5.49
	sausage  int64 // each, $4.00
}

func newFixture(t *testing.T) fixture {
	s := newTestService(t)
	c, err := s.CreateCustomer(ctx, CustomerInput{Name: "Blue Plate", Active: true})
	mustNoErr(t, err)
	w := int64(6000)
	b, err := s.CreateProduct(ctx, ProductInput{SKU: "BRIS", Name: "Brisket", SellUnit: "case", CatchWeight: true, ApproxCaseWeight: &w, BasePriceCents: 599, Active: true})
	mustNoErr(t, err)
	sg, err := s.CreateProduct(ctx, ProductInput{SKU: "SAUS", Name: "Sausage", SellUnit: "each", BasePriceCents: 400, Active: true})
	mustNoErr(t, err)
	_, err = s.SetCustomerPrice(ctx, c.ID, PriceInput{ProductID: b.ID, PriceCents: 549, EffectiveFrom: "2026-01-01"})
	mustNoErr(t, err)
	return fixture{s: s, customer: c.ID, brisket: b.ID, sausage: sg.ID}
}

func TestCreateOrderAndLinesFreezePrice(t *testing.T) {
	f := newFixture(t)
	o, err := f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: "2026-09-12"})
	mustNoErr(t, err)
	if o.Order.Status != "draft" || len(o.Lines) != 0 {
		t.Fatalf("new order: %+v", o.Order)
	}
	_, err = f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: "12/09/2026"})
	wantCode(t, err, "invalid")
	_, err = f.s.CreateOrder(ctx, OrderInput{CustomerID: 999, RequestedDeliveryDate: "2026-09-12"})
	wantCode(t, err, "not_found")

	o, err = f.s.AddLine(ctx, o.Order.ID, f.brisket, 200) // 2 cases
	mustNoErr(t, err)
	l := o.Lines[0]
	if l.Line.UnitPriceCents != 549 || l.Line.EstWeight.Int64 != 12000 || l.AmountSource != "estimated" || l.AmountCents != 65880 {
		t.Fatalf("brisket line: %+v amount=%d src=%s", l.Line, l.AmountCents, l.AmountSource)
	}
	o, _ = f.s.AddLine(ctx, o.Order.ID, f.sausage, 1000) // 10 each
	if o.Lines[1].Line.UnitPriceCents != 400 || o.Lines[1].AmountCents != 4000 || o.TotalCents != 69880 {
		t.Fatalf("sausage line / total: %+v total=%d", o.Lines[1], o.TotalCents)
	}
	// price change after the fact does not touch the line
	_, _ = f.s.SetCustomerPrice(ctx, f.customer, PriceInput{ProductID: f.brisket, PriceCents: 100, EffectiveFrom: "2026-01-02"})
	o, _ = f.s.GetOrder(ctx, o.Order.ID)
	if o.Lines[0].Line.UnitPriceCents != 549 {
		t.Fatal("price was not frozen")
	}
	_, err = f.s.AddLine(ctx, o.Order.ID, f.brisket, 0)
	wantCode(t, err, "invalid")
}

func TestLineEditsAndOverride(t *testing.T) {
	f := newFixture(t)
	o, _ := f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: "2026-09-12"})
	o, _ = f.s.AddLine(ctx, o.Order.ID, f.brisket, 200)
	lineID := o.Lines[0].Line.ID
	price := int64(500)
	qty := int64(300)
	o, err := f.s.UpdateLine(ctx, o.Order.ID, lineID, LinePatch{OrderedQty: &qty, UnitPriceCents: &price})
	mustNoErr(t, err)
	l := o.Lines[0].Line
	if l.OrderedQty != 300 || l.UnitPriceCents != 500 || !l.PriceOverridden || l.EstWeight.Int64 != 18000 {
		t.Fatalf("patch: %+v", l)
	}
	shipped := int64(17550)
	o, _ = f.s.UpdateLine(ctx, o.Order.ID, lineID, LinePatch{ShippedWeight: &shipped})
	if o.Lines[0].AmountSource != "shipped" || o.Lines[0].AmountCents != 87750 {
		t.Fatalf("shipped amount: %+v", o.Lines[0])
	}
	o, err = f.s.DeleteLine(ctx, o.Order.ID, lineID)
	mustNoErr(t, err)
	if len(o.Lines) != 0 {
		t.Fatal("line not deleted")
	}
	_, err = f.s.DeleteLine(ctx, o.Order.ID, 999)
	wantCode(t, err, "not_found")
}

func TestOrderTransitionsAndLocks(t *testing.T) {
	f := newFixture(t)
	o, _ := f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: "2026-09-12"})
	_, err := f.s.ConfirmOrder(ctx, o.Order.ID)
	wantCode(t, err, "no_lines")
	o, _ = f.s.AddLine(ctx, o.Order.ID, f.sausage, 100)
	o, err = f.s.ConfirmOrder(ctx, o.Order.ID)
	mustNoErr(t, err)
	if o.Order.Status != "confirmed" {
		t.Fatal(o.Order.Status)
	}
	// still editable while confirmed
	o, err = f.s.AddLine(ctx, o.Order.ID, f.brisket, 100)
	mustNoErr(t, err)
	_, err = f.s.FinalizeOrder(ctx, o.Order.ID)
	wantCode(t, err, "invalid_transition")
	o, err = f.s.UnconfirmOrder(ctx, o.Order.ID)
	mustNoErr(t, err)
	if o.Order.Status != "draft" {
		t.Fatal(o.Order.Status)
	}
	o, _ = f.s.CancelOrder(ctx, o.Order.ID)
	if o.Order.Status != "cancelled" {
		t.Fatal(o.Order.Status)
	}
	_, err = f.s.AddLine(ctx, o.Order.ID, f.sausage, 100)
	wantCode(t, err, "locked")
	_, err = f.s.ConfirmOrder(ctx, o.Order.ID)
	wantCode(t, err, "invalid_transition")
	// cancelling an already-cancelled order is an invalid transition, not "locked"
	_, err = f.s.CancelOrder(ctx, o.Order.ID)
	wantCode(t, err, "invalid_transition")
}

func TestListOrdersFilters(t *testing.T) {
	f := newFixture(t)
	a, _ := f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: "2026-09-12"})
	_, _ = f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: "2026-09-13"})
	_, _ = f.s.AddLine(ctx, a.Order.ID, f.sausage, 100)
	_, _ = f.s.ConfirmOrder(ctx, a.Order.ID)
	all, _ := f.s.ListOrders(ctx, OrderFilter{})
	byDate, _ := f.s.ListOrders(ctx, OrderFilter{Date: "2026-09-12"})
	byStatus, _ := f.s.ListOrders(ctx, OrderFilter{Status: "confirmed"})
	if len(all) != 2 || len(byDate) != 1 || len(byStatus) != 1 || byStatus[0].LineCount != 1 || byStatus[0].CustomerName != "Blue Plate" {
		t.Fatalf("all=%d date=%d status=%d", len(all), len(byDate), len(byStatus))
	}
}
