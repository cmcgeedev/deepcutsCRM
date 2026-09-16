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
	now := svc.Now().In(svc.Loc)
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

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

	fin, err := svc.ListOrders(ctx, service.OrderFilter{Date: yesterday, Status: "finalized"})
	if err != nil || len(fin) != 1 {
		t.Fatalf("yesterday finalized: err=%v got=%d", err, len(fin))
	}
	rev, err := svc.ListOrders(ctx, service.OrderFilter{Date: yesterday, Status: "delivered"})
	if err != nil || len(rev) != 2 {
		t.Fatalf("yesterday delivered: err=%v got=%d", err, len(rev))
	}
	needsReview := 0
	for _, o := range rev {
		if o.NeedsReview {
			needsReview++
		}
	}
	if needsReview != 1 {
		t.Fatalf("expected exactly one delivered order needing review, got %d", needsReview)
	}

	prods, err := svc.ListProducts(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	catchCount, plainCount := 0, 0
	for _, p := range prods {
		if p.CatchWeight {
			catchCount++
		} else {
			plainCount++
		}
	}
	if catchCount != 20 || plainCount != 20 {
		t.Fatalf("expected 20 catch-weight and 20 non-catch-weight products, got %d/%d", catchCount, plainCount)
	}

	blue, err := svc.Q.GetCustomerByName(ctx, "Blue Plate Diner")
	if err != nil {
		t.Fatal(err)
	}
	bluePrices, err := svc.ListCustomerPrices(ctx, blue.ID, today)
	if err != nil || len(bluePrices) != 8 {
		t.Fatalf("Blue Plate Diner prices: err=%v got=%d", err, len(bluePrices))
	}
	tr, err := svc.Q.GetCustomerByName(ctx, "Two Rivers Brewing")
	if err != nil {
		t.Fatal(err)
	}
	trPrices, err := svc.ListCustomerPrices(ctx, tr.ID, today)
	if err != nil || len(trPrices) != 0 {
		t.Fatalf("Two Rivers Brewing prices: err=%v got=%d", err, len(trPrices))
	}

	drafts, err := svc.ListOrders(ctx, service.OrderFilter{Status: "draft"})
	if err != nil {
		t.Fatal(err)
	}
	hasFutureDraft := false
	for _, o := range drafts {
		if o.RequestedDeliveryDate > today {
			hasFutureDraft = true
		}
	}
	if !hasFutureDraft {
		t.Fatal("expected at least one draft order after today")
	}
	confirmed, err := svc.ListOrders(ctx, service.OrderFilter{Status: "confirmed"})
	if err != nil {
		t.Fatal(err)
	}
	hasFutureConfirmed := false
	for _, o := range confirmed {
		if o.RequestedDeliveryDate > today {
			hasFutureConfirmed = true
		}
	}
	if !hasFutureConfirmed {
		t.Fatal("expected at least one confirmed order after today")
	}

	if _, err := Demo(ctx, svc, a); err == nil {
		t.Fatal("second seed must refuse")
	}
}
