package service

import (
	"testing"

	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
)

type memProofs struct{ saved int }

func (m *memProofs) Save(kind string, data []byte) (string, error) {
	m.saved++
	return "proofs/x.png", nil
}

const cid1 = "11111111-1111-4111-8111-111111111111"
const cid2 = "22222222-2222-4222-8222-222222222222"

func outRoute(t *testing.T) (fixture, int64, int64, int64) {
	t.Helper()
	f := newFixture(t)
	drv := f.driver(t, "Sam")
	o := f.confirmedOrder(t, "2026-09-10") // today in the fixture clock
	r, _ := f.s.CreateRoute(ctx, RouteInput{RouteDate: "2026-09-10", DriverUserID: drv})
	r, _ = f.s.AddStop(ctx, r.Route.ID, o)
	shipped := int64(5900)
	_, err := f.s.UpdateLine(ctx, o, mustLine(t, f, o), LinePatch{ShippedWeight: &shipped})
	mustNoErr(t, err)
	r, err = f.s.RouteOut(ctx, r.Route.ID)
	mustNoErr(t, err)
	return f, drv, r.Route.ID, r.Stops[0].Stop.ID
}

func TestDriverRouteForDate(t *testing.T) {
	f, drv, routeID, _ := outRoute(t)
	dr, err := f.s.DriverRouteForDate(ctx, drv, f.s.Today())
	mustNoErr(t, err)
	if dr.Route.ID != routeID || len(dr.Stops) != 1 || dr.Stops[0].Customer.Name != "Blue Plate" || len(dr.Stops[0].Order.Lines) != 1 {
		t.Fatalf("driver route: %+v", dr)
	}
	_, err = f.s.DriverRouteForDate(ctx, drv, "2026-01-01")
	wantCode(t, err, "not_found")
	other := f.driver(t, "Riley")
	_, err = f.s.DriverRouteForDate(ctx, other, f.s.Today())
	wantCode(t, err, "not_found")
}

func TestDeliverIsIdempotentAndDefaultsDelivered(t *testing.T) {
	f, drv, _, stopID := outRoute(t)
	f.s.Proofs = &memProofs{}
	a := domain.DriverAction{ClientID: cid1, Type: domain.ActionDeliver, Proof: &domain.Proof{Type: domain.ProofName, Name: "Pat"}, Note: "left at dock"}
	applied, st, err := f.s.ApplyDriverAction(ctx, drv, stopID, a)
	mustNoErr(t, err)
	if !applied || st.Stop.Status != "delivered" || st.Stop.ProofType.String != "name" || st.Stop.ProofRef.String != "Pat" || st.Stop.DriverNote != "left at dock" {
		t.Fatalf("deliver: applied=%v %+v", applied, st.Stop)
	}
	l := st.Order.Lines[0].Line
	if !l.DeliveredQty.Valid || l.DeliveredQty.Int64 != 100 || l.DeliveredWeight.Int64 != 5900 || st.Order.Order.Status != "delivered" || st.Order.Order.NeedsReview {
		t.Fatalf("defaults: %+v order=%+v", l, st.Order.Order)
	}
	applied, _, err = f.s.ApplyDriverAction(ctx, drv, stopID, a)
	mustNoErr(t, err)
	if applied {
		t.Fatal("replay must not apply")
	}
	_, _, err = f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid2, Type: domain.ActionDeliver, Proof: a.Proof})
	wantCode(t, err, "invalid_transition") // already delivered
}

func TestDeliverWithImageProofSaves(t *testing.T) {
	f, drv, _, stopID := outRoute(t)
	m := &memProofs{}
	f.s.Proofs = m
	a := domain.DriverAction{ClientID: cid1, Type: domain.ActionDeliver, Proof: &domain.Proof{Type: domain.ProofSignature, Data: []byte{0x89, 'P', 'N', 'G'}}}
	_, st, err := f.s.ApplyDriverAction(ctx, drv, stopID, a)
	mustNoErr(t, err)
	if m.saved != 1 || st.Stop.ProofRef.String != "proofs/x.png" {
		t.Fatalf("proof not saved: %+v", st.Stop)
	}
	applied, _, err := f.s.ApplyDriverAction(ctx, drv, stopID, a)
	mustNoErr(t, err)
	if applied || m.saved != 1 {
		t.Fatalf("replay must not re-save the proof: applied=%v saved=%d", applied, m.saved)
	}
}

func TestAdjustSetsNeedsReview(t *testing.T) {
	f, drv, _, stopID := outRoute(t)
	f.s.Proofs = &memProofs{}
	lineID := mustLine(t, f, f.orderOf(t, stopID))
	short := domain.Hundredths(5000)
	note := "one case rejected, temp"
	adj := domain.DriverAction{ClientID: cid1, Type: domain.ActionAdjust, Lines: []domain.LineAdjustment{{LineID: lineID, DeliveredWeight: &short, ShortageNote: &note}}}
	applied, st, err := f.s.ApplyDriverAction(ctx, drv, stopID, adj)
	mustNoErr(t, err)
	if !applied || st.Order.Lines[0].Line.DeliveredWeight.Int64 != 5000 || st.Order.Lines[0].Line.ShortageNote != note {
		t.Fatalf("adjust before deliver: %+v", st.Order.Lines[0].Line)
	}
	_, st, err = f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid2, Type: domain.ActionDeliver, Proof: &domain.Proof{Type: domain.ProofName, Name: "Pat"}})
	mustNoErr(t, err)
	if st.Order.Lines[0].Line.DeliveredWeight.Int64 != 5000 || !st.Order.Order.NeedsReview {
		t.Fatalf("deliver must keep adjustment and flag review: %+v %+v", st.Order.Lines[0].Line, st.Order.Order)
	}
}

func TestAdjustAfterDeliverFlagsReview(t *testing.T) {
	f, drv, _, stopID := outRoute(t)
	f.s.Proofs = &memProofs{}
	_, st, err := f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid1, Type: domain.ActionDeliver, Proof: &domain.Proof{Type: domain.ProofName, Name: "Pat"}})
	mustNoErr(t, err)
	if st.Order.Order.NeedsReview {
		t.Fatal("clean delivery must not need review")
	}
	lineID := st.Order.Lines[0].Line.ID
	w := domain.Hundredths(4000)
	_, st, err = f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid2, Type: domain.ActionAdjust, Lines: []domain.LineAdjustment{{LineID: lineID, DeliveredWeight: &w}}})
	mustNoErr(t, err)
	if st.Order.Lines[0].Line.DeliveredWeight.Int64 != 4000 || !st.Order.Order.NeedsReview {
		t.Fatalf("adjust after deliver must flag review: %+v", st.Order.Order)
	}
}

func (f fixture) orderOf(t *testing.T, stopID int64) int64 {
	t.Helper()
	st, err := f.s.Q.GetStop(ctx, stopID)
	mustNoErr(t, err)
	return st.OrderID
}

func TestSkipReturnsOrderToConfirmed(t *testing.T) {
	f, drv, routeID, stopID := outRoute(t)
	orderID := f.orderOf(t, stopID)
	_, st, err := f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid1, Type: domain.ActionSkip, SkipReason: "closed"})
	mustNoErr(t, err)
	if st.Stop.Status != "skipped" || st.Stop.SkipReason != "closed" || st.Order.Order.Status != "confirmed" || !st.Order.Order.NeedsReview {
		t.Fatalf("skip: %+v %+v", st.Stop, st.Order.Order)
	}
	dr, err := f.s.DriverCompleteRoute(ctx, drv, routeID)
	mustNoErr(t, err)
	if dr.Route.Status != "complete" {
		t.Fatal(dr.Route.Status)
	}
	// reschedule onto a new route works: the skipped stop stays, a new pending stop is created
	r2, _ := f.s.CreateRoute(ctx, RouteInput{RouteDate: "2026-09-10", DriverUserID: drv})
	r2, err = f.s.AddStop(ctx, r2.Route.ID, orderID)
	mustNoErr(t, err)
	if len(r2.Stops) != 1 || r2.Stops[0].Order.Status != "scheduled" {
		t.Fatalf("reschedule: %+v", r2.Stops)
	}
}

func TestDriverCannotTouchOtherRoute(t *testing.T) {
	f, _, routeID, stopID := outRoute(t)
	other := f.driver(t, "Riley")
	_, _, err := f.s.ApplyDriverAction(ctx, other, stopID, domain.DriverAction{ClientID: cid1, Type: domain.ActionSkip, SkipReason: "x"})
	wantCode(t, err, "not_found")
	_, err = f.s.DriverCompleteRoute(ctx, other, routeID)
	wantCode(t, err, "not_found")
}

func TestAdjustRejectsForeignLine(t *testing.T) {
	f, drv, _, stopID := outRoute(t)
	w := domain.Hundredths(1)
	_, _, err := f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid1, Type: domain.ActionAdjust, Lines: []domain.LineAdjustment{{LineID: 999, DeliveredWeight: &w}}})
	wantCode(t, err, "invalid")
}

func TestActionOnPlannedRouteRejected(t *testing.T) {
	f := newFixture(t)
	drv := f.driver(t, "Sam")
	o := f.confirmedOrder(t, "2026-09-10")
	r, err := f.s.CreateRoute(ctx, RouteInput{RouteDate: "2026-09-10", DriverUserID: drv})
	mustNoErr(t, err)
	r, err = f.s.AddStop(ctx, r.Route.ID, o)
	mustNoErr(t, err)
	_, _, err = f.s.ApplyDriverAction(ctx, drv, r.Stops[0].Stop.ID, domain.DriverAction{ClientID: cid1, Type: domain.ActionSkip, SkipReason: "x"})
	wantCode(t, err, "route_not_out")
}

func TestClientIDReusedOnDifferentStopRejected(t *testing.T) {
	f := newFixture(t)
	drv := f.driver(t, "Sam")
	o1 := f.confirmedOrder(t, "2026-09-10")
	o2 := f.confirmedOrder(t, "2026-09-10")
	r, err := f.s.CreateRoute(ctx, RouteInput{RouteDate: "2026-09-10", DriverUserID: drv})
	mustNoErr(t, err)
	r, err = f.s.AddStop(ctx, r.Route.ID, o1)
	mustNoErr(t, err)
	r, err = f.s.AddStop(ctx, r.Route.ID, o2)
	mustNoErr(t, err)
	r, err = f.s.RouteOut(ctx, r.Route.ID)
	mustNoErr(t, err)
	_, _, err = f.s.ApplyDriverAction(ctx, drv, r.Stops[0].Stop.ID, domain.DriverAction{ClientID: cid1, Type: domain.ActionSkip, SkipReason: "x"})
	mustNoErr(t, err)
	_, _, err = f.s.ApplyDriverAction(ctx, drv, r.Stops[1].Stop.ID, domain.DriverAction{ClientID: cid1, Type: domain.ActionSkip, SkipReason: "y"})
	wantCode(t, err, "client_id_reused")
}

func TestReplayAfterRouteCompleteIsNoop(t *testing.T) {
	f, drv, routeID, stopID := outRoute(t)
	f.s.Proofs = &memProofs{}
	a := domain.DriverAction{ClientID: cid1, Type: domain.ActionDeliver, Proof: &domain.Proof{Type: domain.ProofName, Name: "Pat"}}
	_, _, err := f.s.ApplyDriverAction(ctx, drv, stopID, a)
	mustNoErr(t, err)
	_, err = f.s.DriverCompleteRoute(ctx, drv, routeID)
	mustNoErr(t, err)
	applied, st, err := f.s.ApplyDriverAction(ctx, drv, stopID, a)
	mustNoErr(t, err)
	if applied {
		t.Fatal("replay after route complete must not apply")
	}
	if st.Stop.Status != "delivered" {
		t.Fatalf("replay must return current stop: %+v", st.Stop)
	}
}

func TestSkipClearsAbortedDeliveryAttempt(t *testing.T) {
	f, drv, _, stopID := outRoute(t)
	lineID := mustLine(t, f, f.orderOf(t, stopID))
	w := domain.Hundredths(3000)
	note := "damaged"
	_, _, err := f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid1, Type: domain.ActionAdjust, Lines: []domain.LineAdjustment{{LineID: lineID, DeliveredWeight: &w, ShortageNote: &note}}})
	mustNoErr(t, err)
	_, st, err := f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid2, Type: domain.ActionSkip, SkipReason: "closed"})
	mustNoErr(t, err)
	l := st.Order.Lines[0].Line
	if l.DeliveredWeight.Valid || l.DeliveredQty.Valid || l.ShortageNote != "" {
		t.Fatalf("skip must clear aborted delivery attempt: %+v", l)
	}
}

func TestDeliverPreservesEarlierNote(t *testing.T) {
	f, drv, _, stopID := outRoute(t)
	f.s.Proofs = &memProofs{}
	lineID := mustLine(t, f, f.orderOf(t, stopID))
	qty := domain.Hundredths(100)
	_, st, err := f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid1, Type: domain.ActionAdjust, Lines: []domain.LineAdjustment{{LineID: lineID, DeliveredQty: &qty}}, Note: "customer not answering"})
	mustNoErr(t, err)
	if st.Stop.DriverNote != "customer not answering" {
		t.Fatalf("adjust note not set: %+v", st.Stop)
	}
	_, st, err = f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid2, Type: domain.ActionDeliver, Proof: &domain.Proof{Type: domain.ProofName, Name: "Pat"}})
	mustNoErr(t, err)
	if st.Stop.DriverNote != "customer not answering" {
		t.Fatalf("deliver must preserve earlier note: %+v", st.Stop)
	}
}
