package seed

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
	"github.com/cmcgeedev/deepcutsCRM/internal/storage"
)

func TestDemoSeed(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.OpenAndMigrate(ctx, filepath.Join(dir, "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	loc, _ := time.LoadLocation("America/New_York")
	svc := service.New(d, loc)
	svc.Proofs, _ = storage.NewLocal(filepath.Join(dir, "uploads"))
	a := auth.New(queries.New(d))
	info, err := Demo(ctx, svc, a)
	if err != nil {
		t.Fatal(err)
	}
	if info.Customers != 12 || info.Products != 40 || len(info.Drivers) != 2 || info.Orders < 12 {
		t.Fatalf("info: %+v", info)
	}
	if _, err := a.LoginOffice(ctx, info.OfficeEmail, info.OfficePassword, "t"); err != nil {
		t.Fatal("office login failed")
	}
	today := svc.Today()
	dv, err := svc.DayView(ctx, today)
	if err != nil || len(dv.Routes) != 1 || dv.Routes[0].Route.Status != "out" || len(dv.Routes[0].Stops) != 4 || len(dv.Unscheduled) != 2 {
		t.Fatalf("today: %v %+v", err, dv)
	}
	delivered := 0
	for _, st := range dv.Routes[0].Stops {
		if st.Stop.Status == "delivered" {
			delivered++
		}
	}
	if delivered != 1 {
		t.Fatalf("expected one delivered stop today, got %d", delivered)
	}
	drivers, _ := svc.ListDrivers(ctx)
	var sam int64
	for _, d := range drivers {
		if d.DisplayName == "Sam" {
			sam = d.ID
		}
	}
	dr, err := svc.DriverRouteForDate(ctx, sam, today)
	if err != nil || len(dr.Stops) != 4 {
		t.Fatalf("driver route: %v", err)
	}
	fin, _ := svc.ListOrders(ctx, service.OrderFilter{Status: "finalized"})
	rev, _ := svc.ListOrders(ctx, service.OrderFilter{Status: "delivered"})
	if len(fin) < 1 || len(rev) < 1 {
		t.Fatalf("yesterday: finalized=%d delivered=%d", len(fin), len(rev))
	}
	if _, err := Demo(ctx, svc, a); err == nil {
		t.Fatal("second seed must refuse")
	}
}
