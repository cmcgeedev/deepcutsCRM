package service

import (
	"testing"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
)

func (f fixture) confirmedOrder(t *testing.T, date string) int64 {
	t.Helper()
	o, err := f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: date})
	mustNoErr(t, err)
	_, err = f.s.AddLine(ctx, o.Order.ID, f.brisket, 100)
	mustNoErr(t, err)
	_, err = f.s.ConfirmOrder(ctx, o.Order.ID)
	mustNoErr(t, err)
	return o.Order.ID
}

func (f fixture) driver(t *testing.T, name string) int64 {
	t.Helper()
	u, err := f.s.Q.CreateUser(ctx, queries.CreateUserParams{Realm: "driver", DisplayName: name, CreatedAt: f.s.now()})
	mustNoErr(t, err)
	return u.ID
}

func TestRouteSchedulingAndDayView(t *testing.T) {
	f := newFixture(t)
	drv := f.driver(t, "Sam")
	o1 := f.confirmedOrder(t, "2026-09-12")
	o2 := f.confirmedOrder(t, "2026-09-12")
	o3 := f.confirmedOrder(t, "2026-09-13")

	_, err := f.s.CreateRoute(ctx, RouteInput{RouteDate: "bad", DriverUserID: drv})
	wantCode(t, err, "invalid")
	_, err = f.s.CreateRoute(ctx, RouteInput{RouteDate: "2026-09-12", DriverUserID: 999})
	wantCode(t, err, "not_found")
	r, err := f.s.CreateRoute(ctx, RouteInput{RouteDate: "2026-09-12", DriverUserID: drv, TruckLabel: "Reefer 1"})
	mustNoErr(t, err)
	if r.Route.Status != "planned" || r.DriverName != "Sam" {
		t.Fatalf("route: %+v", r)
	}

	r, err = f.s.AddStop(ctx, r.Route.ID, o1)
	mustNoErr(t, err)
	r, err = f.s.AddStop(ctx, r.Route.ID, o2)
	mustNoErr(t, err)
	if len(r.Stops) != 2 || r.Stops[0].Stop.Sequence != 1 || r.Stops[1].Stop.Sequence != 2 || r.Stops[0].Order.Status != "scheduled" {
		t.Fatalf("stops: %+v", r.Stops)
	}
	_, err = f.s.AddStop(ctx, r.Route.ID, o1)
	wantCode(t, err, "invalid_transition") // already scheduled
	_, err = f.s.AddStop(ctx, r.Route.ID, o3)
	wantCode(t, err, "date_mismatch")

	dv, err := f.s.DayView(ctx, "2026-09-12")
	mustNoErr(t, err)
	if len(dv.Unscheduled) != 0 || len(dv.Routes) != 1 || len(dv.Routes[0].Stops) != 2 {
		t.Fatalf("day view: %+v", dv)
	}

	r, err = f.s.ReorderStops(ctx, r.Route.ID, []int64{r.Stops[1].Stop.ID, r.Stops[0].Stop.ID})
	mustNoErr(t, err)
	if r.Stops[0].Order.ID != o2 || r.Stops[0].Stop.Sequence != 1 {
		t.Fatalf("reorder: %+v", r.Stops)
	}
	_, err = f.s.ReorderStops(ctx, r.Route.ID, []int64{r.Stops[0].Stop.ID})
	wantCode(t, err, "invalid")

	r, err = f.s.RemoveStop(ctx, r.Route.ID, r.Stops[1].Stop.ID)
	mustNoErr(t, err)
	if len(r.Stops) != 1 {
		t.Fatal("stop not removed")
	}
	od, _ := f.s.GetOrder(ctx, o1)
	if od.Order.Status != "confirmed" || od.RouteID != nil {
		t.Fatalf("unscheduled order: %+v", od.Order)
	}
	dv, _ = f.s.DayView(ctx, "2026-09-12")
	if len(dv.Unscheduled) != 1 {
		t.Fatalf("unscheduled should list o1: %+v", dv.Unscheduled)
	}
}

func TestRouteOutLocksAndComplete(t *testing.T) {
	f := newFixture(t)
	drv := f.driver(t, "Sam")
	o1 := f.confirmedOrder(t, "2026-09-12")
	r, _ := f.s.CreateRoute(ctx, RouteInput{RouteDate: "2026-09-12", DriverUserID: drv})
	_, err := f.s.RouteOut(ctx, r.Route.ID)
	wantCode(t, err, "empty_route")
	r, _ = f.s.AddStop(ctx, r.Route.ID, o1)
	r, err = f.s.RouteOut(ctx, r.Route.ID)
	mustNoErr(t, err)
	if r.Route.Status != "out" || !r.Route.OutAt.Valid {
		t.Fatalf("out: %+v", r.Route)
	}
	_, err = f.s.RemoveStop(ctx, r.Route.ID, r.Stops[0].Stop.ID)
	wantCode(t, err, "locked")
	_, err = f.s.AddStop(ctx, r.Route.ID, f.confirmedOrder(t, "2026-09-12"))
	wantCode(t, err, "locked")
	qty := int64(200)
	_, err = f.s.UpdateLine(ctx, o1, mustLine(t, f, o1), LinePatch{OrderedQty: &qty})
	wantCode(t, err, "invalid") // ordered qty locked once out
	shipped := int64(5900)
	_, err = f.s.UpdateLine(ctx, o1, mustLine(t, f, o1), LinePatch{ShippedWeight: &shipped})
	mustNoErr(t, err)
	_, err = f.s.CancelOrder(ctx, o1)
	wantCode(t, err, "locked")
	_, err = f.s.RouteComplete(ctx, r.Route.ID)
	wantCode(t, err, "stops_pending")
}

func mustLine(t *testing.T, f fixture, orderID int64) int64 {
	t.Helper()
	o, err := f.s.GetOrder(ctx, orderID)
	mustNoErr(t, err)
	return o.Lines[0].Line.ID
}
