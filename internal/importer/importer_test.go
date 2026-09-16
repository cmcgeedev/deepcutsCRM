package importer

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

var ctx = context.Background()

func sqlStr(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }

func newSvc(t *testing.T) *service.Service {
	t.Helper()
	d, err := db.OpenAndMigrate(ctx, filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	loc, _ := time.LoadLocation("America/New_York")
	return service.New(d, loc)
}

func TestCustomersImportAndRerun(t *testing.T) {
	s := newSvc(t)
	csv := "Name,Delivery_Address,delivery_days,qbo_customer_id,phone\nBlue Plate,1 Main St,mon;thu,q1,555\nNo QBO,2 Side St,,,\n,3 Nowhere,,,\n"
	sum, err := Customers(ctx, s, strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if sum.Created != 2 || sum.Rejected != 1 || len(sum.Errors) != 1 {
		t.Fatalf("first run: %+v", sum)
	}
	csv2 := "name,delivery_address,qbo_customer_id\nBlue Plate Diner,9 New St,q1\nNo QBO,2 Side St,\n"
	sum, err = Customers(ctx, s, strings.NewReader(csv2))
	if err != nil {
		t.Fatal(err)
	}
	if sum.Created != 0 || sum.Updated != 2 {
		t.Fatalf("rerun: %+v", sum)
	}
	all, _ := s.ListCustomers(ctx, true)
	if len(all) != 2 {
		t.Fatalf("expected 2 customers, got %d", len(all))
	}
	for _, c := range all {
		if c.QboCustomerID.String == "q1" && (c.Name != "Blue Plate Diner" || c.DeliveryAddress != "9 New St" || c.DeliveryDays != "mon,thu") {
			t.Fatalf("q1 not updated in place, or delivery days lost: %+v", c)
		}
	}
	if _, err := Customers(ctx, s, strings.NewReader("phone\n555\n")); err == nil {
		t.Fatal("missing name column must be an error")
	}
}

func TestProductsImport(t *testing.T) {
	s := newSvc(t)
	csv := "sku,name,category,sell_unit,catch_weight,approx_case_weight_lb,base_price,qbo_item_id\n" +
		"BRIS,Brisket,Beef,case,yes,60,5.99,i1\n" +
		"SAUS,Sausage,Pork,each,no,,4,i2\n" +
		"BAD,Bad,Beef,kg,no,,1,\n" +
		"CW,Catch no weight,Beef,case,true,,1,\n"
	sum, err := Products(ctx, s, strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if sum.Created != 2 || sum.Rejected != 2 {
		t.Fatalf("products: %+v", sum)
	}
	p, err := s.Q.GetProductBySKU(ctx, "BRIS")
	if err != nil || !p.CatchWeight || p.ApproxCaseWeight.Int64 != 6000 || p.BasePriceCents != 599 || p.QboItemID.String != "i1" {
		t.Fatalf("brisket: %+v %v", p, err)
	}
	sum, _ = Products(ctx, s, strings.NewReader("sku,name,sell_unit,base_price\nBRIS,Whole Brisket,case,6.49\n"))
	if sum.Updated != 1 {
		t.Fatalf("rerun: %+v", sum)
	}
	p, _ = s.Q.GetProductBySKU(ctx, "BRIS")
	if p.Name != "Whole Brisket" || p.BasePriceCents != 649 || !p.CatchWeight || p.ApproxCaseWeight.Int64 != 6000 {
		t.Fatalf("partial update must keep unspecified fields: %+v", p)
	}
}

func TestPricesImport(t *testing.T) {
	s := newSvc(t)
	Customers(ctx, s, strings.NewReader("name,qbo_customer_id\nBlue Plate,q1\nRed Barn,\n"))
	Products(ctx, s, strings.NewReader("sku,name,sell_unit,base_price\nBRIS,Brisket,lb,5.99\n"))
	csv := "customer,sku,price\nq1,BRIS,5.49\nRed Barn,BRIS,5.79\nNobody,BRIS,1\nq1,NOPE,1\nq1,BRIS,abc\n"
	sum, err := Prices(ctx, s, strings.NewReader(csv), "2026-09-10")
	if err != nil {
		t.Fatal(err)
	}
	if sum.Created != 2 || sum.Rejected != 3 {
		t.Fatalf("prices: %+v", sum)
	}
	c, _ := s.Q.GetCustomerByQBO(ctx, sqlStr("q1"))
	rows, _ := s.ListCustomerPrices(ctx, c.ID, "2026-09-10")
	if len(rows) != 1 || rows[0].PriceCents != 549 {
		t.Fatalf("q1 price: %+v", rows)
	}
}
