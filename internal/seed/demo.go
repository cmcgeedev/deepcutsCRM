// Package seed loads a realistic demo data set through the service layer so every state machine rule holds.
package seed

import (
	"context"
	"fmt"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

type DriverLogin struct{ Name, PIN string }

type Info struct {
	OfficeEmail, OfficePassword string
	Drivers                     []DriverLogin
	Customers, Products, Orders int
}

type product struct {
	sku, name, category, unit string
	catch                     bool
	caseLb                    int64 // whole pounds
	price                     int64 // cents per unit (per lb when catch)
}

var products = []product{
	{"BF-BRIS", "Brisket, packer", "Beef", "case", true, 60, 599}, {"BF-RIB", "Ribeye, boneless", "Beef", "case", true, 45, 1499},
	{"BF-STRIP", "Strip loin", "Beef", "case", true, 50, 1199}, {"BF-TEND", "Tenderloin PSMO", "Beef", "case", true, 30, 1899},
	{"BF-CHUCK", "Chuck roll", "Beef", "case", true, 70, 449}, {"BF-RND", "Top round", "Beef", "case", true, 60, 479},
	{"BF-SHRT", "Short ribs", "Beef", "case", true, 40, 899}, {"BF-FLNK", "Flank steak", "Beef", "case", true, 25, 999},
	{"BF-SKRT", "Skirt steak, outside", "Beef", "case", true, 25, 1099}, {"BF-TRI", "Tri-tip", "Beef", "case", true, 30, 799},
	{"PK-BLLY", "Pork belly, skinless", "Pork", "case", true, 55, 389}, {"PK-SHLD", "Pork shoulder, bone-in", "Pork", "case", true, 65, 219},
	{"PK-LOIN", "Pork loin, boneless", "Pork", "case", true, 50, 279}, {"PK-RIBS", "St. Louis ribs", "Pork", "case", true, 30, 349},
	{"PK-BBR", "Baby back ribs", "Pork", "case", true, 30, 429}, {"PK-TEND", "Pork tenderloin", "Pork", "case", true, 20, 399},
	{"LM-LEG", "Lamb leg, bone-in", "Lamb", "case", true, 40, 749}, {"LM-RACK", "Lamb rack, frenched", "Lamb", "case", true, 20, 1999},
	{"CH-WHOLE", "Whole chicken, WOG", "Poultry", "case", true, 40, 189}, {"CH-THIGH", "Chicken thighs, boneless", "Poultry", "case", true, 40, 259},
	{"GB-80", "Ground beef 80/20, 10 lb tube", "Ground", "each", false, 0, 3490}, {"GB-90", "Ground beef 90/10, 10 lb tube", "Ground", "each", false, 0, 4290},
	{"GP-PORK", "Ground pork, 10 lb tube", "Ground", "each", false, 0, 2490}, {"SS-BRAT", "Bratwurst, 5 lb", "Sausage", "each", false, 0, 2250},
	{"SS-ITAL", "Italian sausage, 5 lb", "Sausage", "each", false, 0, 2250}, {"SS-CHOR", "Chorizo, 5 lb", "Sausage", "each", false, 0, 2450},
	{"SS-BKFT", "Breakfast links, 5 lb", "Sausage", "each", false, 0, 2150}, {"BC-SLAB", "Bacon, sliced, 15 lb", "Cured", "each", false, 0, 6750},
	{"BC-THK", "Bacon, thick cut, 15 lb", "Cured", "each", false, 0, 6950}, {"HM-BONE", "Ham, bone-in, each", "Cured", "each", false, 0, 3900},
	{"CH-WING", "Chicken wings, 40 lb", "Poultry", "each", false, 0, 9800}, {"CH-BRST", "Chicken breast, 40 lb", "Poultry", "each", false, 0, 11200},
	{"TK-BRST", "Turkey breast, each", "Poultry", "each", false, 0, 2800}, {"BF-PATTY", "Beef patties 6 oz, 40 ct", "Ground", "each", false, 0, 7200},
	{"BF-STEW", "Stew meat, 10 lb", "Beef", "each", false, 0, 5900}, {"BF-BONE", "Beef bones, 30 lb", "Beef", "each", false, 0, 3000},
	{"PK-FAT", "Pork fatback, 10 lb", "Pork", "each", false, 0, 1500}, {"DL-ROAST", "Deli roast beef, 8 lb", "Deli", "each", false, 0, 6400},
	{"DL-TURK", "Deli turkey, 8 lb", "Deli", "each", false, 0, 5600}, {"DL-HAM", "Deli ham, 8 lb", "Deli", "each", false, 0, 4800},
}

type customer struct{ name, addr, contact, phone, notes, days string }

var customers = []customer{
	{"Blue Plate Diner", "112 Main St, Ferndale", "Marcy", "555-0101", "Back door by the dumpsters, ring twice", "mon,thu"},
	{"The Butcher's Table", "9 Depot Rd, Ferndale", "Luis", "555-0102", "Walk-in cooler is left of the loading dock", "mon,wed,fri"},
	{"Red Barn Market", "4400 County Rd 12", "Dana", "555-0103", "Gate code 4471. Dock hours 6-10am", "tue,fri"},
	{"Hilltop Steakhouse", "1 Summit Ave", "Chef Ana", "555-0104", "Deliveries before 11am only", "mon,thu"},
	{"Corner Tavern", "88 Front St", "Mike", "555-0105", "Side alley, knock hard", "wed"},
	{"Greenfield Grocery", "210 Elm St, Greenfield", "Priya", "555-0106", "Receiving at rear, ask for Priya", "mon,wed,fri"},
	{"Smokehouse BBQ", "7 Pit Rd", "Earl", "555-0107", "Leave in the walk-in if nobody's there", "tue,thu"},
	{"Riverside Cafe", "300 River Rd", "Jo", "555-0108", "", "thu"},
	{"Lakeview Country Club", "1 Fairway Dr", "Kitchen", "555-0109", "Use the service road, security will wave you through", "wed,sat"},
	{"Mama Rosa's", "56 Union St", "Rosa", "555-0110", "Ring the bell at the kitchen door", "mon,thu"},
	{"Pinecrest School District", "1000 School Ln", "Cafeteria Mgr", "555-0111", "Dock hours 5:30-8am. No deliveries on holidays", "tue"},
	{"Two Rivers Brewing", "12 Brewery Way", "Sam K", "555-0112", "Kitchen is upstairs; use the freight elevator", "fri"},
}

func Demo(ctx context.Context, svc *service.Service, a *auth.Auth) (Info, error) {
	if existing, err := svc.ListCustomers(ctx, true); err != nil {
		return Info{}, err
	} else if len(existing) > 0 {
		return Info{}, service.Conflict("seed_not_empty", "database already has customers; seed only runs on an empty database")
	}
	info := Info{OfficeEmail: "office@demo.local", OfficePassword: "demo1234", Drivers: []DriverLogin{{"Sam", "111111"}, {"Riley", "222222"}}}
	if _, err := a.CreateOfficeUser(ctx, info.OfficeEmail, "Demo Office", info.OfficePassword); err != nil {
		return info, err
	}
	var driverIDs []int64
	for _, d := range info.Drivers {
		u, err := a.CreateDriver(ctx, d.Name, d.PIN)
		if err != nil {
			return info, err
		}
		driverIDs = append(driverIDs, u.ID)
	}
	var custIDs []int64
	for _, c := range customers {
		row, err := svc.CreateCustomer(ctx, service.CustomerInput{Name: c.name, BillingAddress: c.addr, DeliveryAddress: c.addr, ContactName: c.contact,
			Phone: c.phone, DeliveryNotes: c.notes, DeliveryDays: service.SplitDeliveryDays(c.days), Active: true})
		if err != nil {
			return info, err
		}
		custIDs = append(custIDs, row.ID)
	}
	info.Customers = len(custIDs)
	var prodIDs []int64
	for _, p := range products {
		in := service.ProductInput{SKU: p.sku, Name: p.name, Category: p.category, SellUnit: p.unit, CatchWeight: p.catch, BasePriceCents: p.price, Active: true}
		if p.catch {
			w := p.caseLb * 100
			in.ApproxCaseWeight = &w
		}
		row, err := svc.CreateProduct(ctx, in)
		if err != nil {
			return info, err
		}
		prodIDs = append(prodIDs, row.ID)
	}
	info.Products = len(prodIDs)
	// negotiated prices: first five customers get ~8% off eight products
	for ci := 0; ci < 5; ci++ {
		for pi := 0; pi < 8; pi++ {
			p := products[pi*3%len(products)]
			if _, err := svc.SetCustomerPrice(ctx, custIDs[ci], service.PriceInput{ProductID: prodIDs[pi*3%len(prodIDs)], PriceCents: p.price * 92 / 100, EffectiveFrom: "2026-01-01"}); err != nil {
				return info, err
			}
		}
	}
	today := svc.Now().In(svc.Loc)
	day := func(offset int) string { return today.AddDate(0, 0, offset).Format("2006-01-02") }

	// helper: confirmed order with n lines, deterministic product choice
	mk := func(ci int, date string, n int, confirm bool) (int64, error) {
		o, err := svc.CreateOrder(ctx, service.OrderInput{CustomerID: custIDs[ci], RequestedDeliveryDate: date})
		if err != nil {
			return 0, err
		}
		for k := 0; k < n; k++ {
			pi := (ci*7 + k*5) % len(prodIDs)
			qty := int64(100 * (1 + (ci+k)%3))
			if _, err := svc.AddLine(ctx, o.Order.ID, prodIDs[pi], qty); err != nil {
				return 0, err
			}
		}
		if confirm {
			if _, err := svc.ConfirmOrder(ctx, o.Order.ID); err != nil {
				return 0, err
			}
		}
		info.Orders++
		return o.Order.ID, nil
	}
	// fill shipped weights on catch-weight lines: approx weight ± a little
	ship := func(orderID int64) error {
		od, err := svc.GetOrder(ctx, orderID)
		if err != nil {
			return err
		}
		for i, l := range od.Lines {
			if !l.Product.CatchWeight {
				continue
			}
			w := l.Line.EstWeight.Int64 * int64(97+i*2) / 100
			if _, err := svc.UpdateLine(ctx, orderID, l.Line.ID, service.LinePatch{ShippedWeight: &w}); err != nil {
				return err
			}
		}
		return nil
	}
	deliver := func(driverID, stopID int64, clientID string, adjust bool) error {
		st, err := svc.Q.GetStop(ctx, stopID)
		if err != nil {
			return err
		}
		od, _ := svc.GetOrder(ctx, st.OrderID)
		act := domain.DriverAction{ClientID: clientID, Type: domain.ActionDeliver, Proof: &domain.Proof{Type: domain.ProofName, Name: od.Customer.ContactName}}
		if adjust && len(od.Lines) > 0 {
			short := domain.Hundredths(0)
			note := "customer refused, temperature"
			act.Lines = []domain.LineAdjustment{{LineID: od.Lines[0].Line.ID, DeliveredQty: &short, ShortageNote: &note}}
			if od.Lines[0].Product.CatchWeight {
				act.Lines[0].DeliveredWeight = &short
			}
		}
		_, _, err = svc.ApplyDriverAction(ctx, driverID, stopID, act)
		return err
	}

	// yesterday: complete route, one finalized order, one needing review
	y := day(-1)
	r, err := svc.CreateRoute(ctx, service.RouteInput{RouteDate: y, DriverUserID: driverIDs[0], TruckLabel: "Reefer 1"})
	if err != nil {
		return info, err
	}
	var yStops []int64
	for ci := 0; ci < 3; ci++ {
		oid, err := mk(ci, y, 3, true)
		if err != nil {
			return info, err
		}
		if err := ship(oid); err != nil {
			return info, err
		}
		if r, err = svc.AddStop(ctx, r.Route.ID, oid); err != nil {
			return info, err
		}
		yStops = append(yStops, r.Stops[len(r.Stops)-1].Stop.ID)
	}
	if _, err := svc.RouteOut(ctx, r.Route.ID); err != nil {
		return info, err
	}
	for i, sid := range yStops {
		if err := deliver(driverIDs[0], sid, fmt.Sprintf("00000000-0000-4000-8000-0000000000%02d", i), i == 1); err != nil {
			return info, err
		}
	}
	if _, err := svc.RouteComplete(ctx, r.Route.ID); err != nil {
		return info, err
	}
	first, _ := svc.Q.GetStop(ctx, yStops[0])
	if _, err := svc.FinalizeOrder(ctx, first.OrderID); err != nil {
		return info, err
	}

	// today: route out with 4 stops (one delivered), plus 2 unscheduled confirmed orders
	t := day(0)
	r, err = svc.CreateRoute(ctx, service.RouteInput{RouteDate: t, DriverUserID: driverIDs[0], TruckLabel: "Reefer 1"})
	if err != nil {
		return info, err
	}
	var tStops []int64
	for ci := 3; ci < 7; ci++ {
		oid, err := mk(ci, t, 4, true)
		if err != nil {
			return info, err
		}
		if err := ship(oid); err != nil {
			return info, err
		}
		if r, err = svc.AddStop(ctx, r.Route.ID, oid); err != nil {
			return info, err
		}
		tStops = append(tStops, r.Stops[len(r.Stops)-1].Stop.ID)
	}
	if _, err := svc.RouteOut(ctx, r.Route.ID); err != nil {
		return info, err
	}
	if err := deliver(driverIDs[0], tStops[0], "00000000-0000-4000-8000-0000000000aa", false); err != nil {
		return info, err
	}
	for ci := 7; ci < 9; ci++ {
		if _, err := mk(ci, t, 2, true); err != nil {
			return info, err
		}
	}
	// tomorrow and later: drafts and confirmed
	for ci := 0; ci < 12; ci++ {
		if _, err := mk(ci, day(1+ci%3), 2+ci%3, ci%2 == 0); err != nil {
			return info, err
		}
	}
	return info, nil
}
