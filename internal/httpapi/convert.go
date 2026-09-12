package httpapi

import (
	"database/sql"
	"encoding/base64"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func toUser(s auth.Session) api.User {
	return api.User{Id: s.UserID, Realm: api.UserRealm(s.Realm), DisplayName: s.DisplayName}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefBool(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func derefI64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func nullI64(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func nullT(n sql.NullTime) *time.Time {
	if !n.Valid {
		return nil
	}
	v := n.Time
	return &v
}

func toCustomer(c queries.Customer) api.Customer {
	return api.Customer{
		Id: c.ID, Name: c.Name, BillingAddress: c.BillingAddress, DeliveryAddress: c.DeliveryAddress, ContactName: c.ContactName,
		Phone: c.Phone, Email: c.Email, DeliveryNotes: c.DeliveryNotes, DeliveryDays: service.SplitDeliveryDays(c.DeliveryDays),
		QboCustomerId: strPtr(c.QboCustomerID.String), Active: c.Active,
	}
}

func customerInput(in api.CustomerInput) service.CustomerInput {
	days := []string{}
	if in.DeliveryDays != nil {
		days = *in.DeliveryDays
	}
	return service.CustomerInput{
		Name: in.Name, BillingAddress: deref(in.BillingAddress), DeliveryAddress: deref(in.DeliveryAddress), ContactName: deref(in.ContactName),
		Phone: deref(in.Phone), Email: deref(in.Email), DeliveryNotes: deref(in.DeliveryNotes), DeliveryDays: days,
		QBOCustomerID: deref(in.QboCustomerId), Active: derefBool(in.Active, true),
	}
}

func toProduct(p queries.Product) api.Product {
	return api.Product{
		Id: p.ID, Sku: p.Sku, Name: p.Name, Category: p.Category, SellUnit: api.ProductSellUnit(p.SellUnit), CatchWeight: p.CatchWeight,
		ApproxCaseWeight: nullI64(p.ApproxCaseWeight), BasePriceCents: p.BasePriceCents, QboItemId: strPtr(p.QboItemID.String), Active: p.Active,
	}
}

func productInput(in api.ProductInput) service.ProductInput {
	return service.ProductInput{
		SKU: in.Sku, Name: in.Name, Category: deref(in.Category), SellUnit: string(in.SellUnit), CatchWeight: derefBool(in.CatchWeight, false),
		ApproxCaseWeight: in.ApproxCaseWeight, BasePriceCents: in.BasePriceCents, QBOItemID: deref(in.QboItemId), Active: derefBool(in.Active, true),
	}
}

func toPriceRow(id, productID int64, sku, name string, cents int64, from string) api.CustomerPrice {
	return api.CustomerPrice{Id: id, ProductId: productID, Sku: sku, ProductName: name, PriceCents: cents, EffectiveFrom: from}
}

func toOrderSummary(id, customerID int64, customerName, date, status string, needsReview bool, lineCount int64, notes string) api.OrderSummary {
	return api.OrderSummary{Id: id, CustomerId: customerID, CustomerName: customerName, RequestedDeliveryDate: date,
		Status: api.OrderStatus(status), NeedsReview: needsReview, LineCount: lineCount, Notes: notes}
}

func toLine(l service.LineDetail) api.OrderLine {
	return api.OrderLine{
		Id: l.Line.ID, ProductId: l.Product.ID, Sku: l.Product.Sku, ProductName: l.Product.Name, SellUnit: l.Product.SellUnit,
		CatchWeight: l.Product.CatchWeight, OrderedQty: l.Line.OrderedQty, UnitPriceCents: l.Line.UnitPriceCents, PriceOverridden: l.Line.PriceOverridden,
		EstWeight: nullI64(l.Line.EstWeight), ShippedWeight: nullI64(l.Line.ShippedWeight), DeliveredQty: nullI64(l.Line.DeliveredQty),
		DeliveredWeight: nullI64(l.Line.DeliveredWeight), ShortageNote: l.Line.ShortageNote, AmountCents: l.AmountCents,
		AmountSource: api.OrderLineAmountSource(l.AmountSource),
	}
}

func toOrder(d service.OrderDetail) api.Order {
	lines := make([]api.OrderLine, 0, len(d.Lines))
	for _, l := range d.Lines {
		lines = append(lines, toLine(l))
	}
	return api.Order{
		Id: d.Order.ID, Customer: toCustomer(d.Customer), RequestedDeliveryDate: d.Order.RequestedDeliveryDate, Status: api.OrderStatus(d.Order.Status),
		Notes: d.Order.Notes, NeedsReview: d.Order.NeedsReview, Lines: lines, TotalCents: d.TotalCents, RouteId: d.RouteID, StopId: d.StopID,
		RouteStatus: strPtr(d.RouteStatus), CreatedAt: d.Order.CreatedAt, FinalizedAt: nullT(d.Order.FinalizedAt),
	}
}

func toStop(sd service.StopDetail) api.Stop {
	st := sd.Stop
	out := api.Stop{
		Id: st.ID, RouteId: st.RouteID, OrderId: st.OrderID, Sequence: st.Sequence, Status: api.StopStatus(st.Status), DeliveredAt: nullT(st.DeliveredAt),
		SkipReason: st.SkipReason, DriverNote: st.DriverNote, CustomerName: sd.Customer.Name, DeliveryAddress: sd.Customer.DeliveryAddress,
		Phone: sd.Customer.Phone, ContactName: sd.Customer.ContactName, DeliveryNotes: sd.Customer.DeliveryNotes,
		OrderStatus: api.OrderStatus(sd.Order.Status), NeedsReview: sd.Order.NeedsReview, LineCount: sd.LineCount,
	}
	if st.ProofType.Valid {
		pt := api.StopProofType(st.ProofType.String)
		out.ProofType = &pt
		if st.ProofType.String == "name" {
			out.ProofName = strPtr(st.ProofRef.String)
		} else {
			out.HasProofImage = st.ProofRef.Valid
		}
	}
	return out
}

func toRoute(rd service.RouteDetail) api.Route {
	stops := make([]api.Stop, 0, len(rd.Stops))
	for _, s := range rd.Stops {
		stops = append(stops, toStop(s))
	}
	r := rd.Route
	return api.Route{Id: r.ID, RouteDate: r.RouteDate, DriverUserId: r.DriverUserID, DriverName: rd.DriverName, TruckLabel: r.TruckLabel,
		Status: api.RouteStatus(r.Status), OutAt: nullT(r.OutAt), CompletedAt: nullT(r.CompletedAt), Stops: stops}
}

func toDriverStop(ds service.DriverStop) api.DriverStop {
	sd := service.StopDetail{Stop: ds.Stop, Order: ds.Order.Order, Customer: ds.Customer, LineCount: int64(len(ds.Order.Lines))}
	return api.DriverStop{Stop: toStop(sd), Order: toOrder(ds.Order)}
}

func toDriverRoute(dr service.DriverRoute, driverName string) api.DriverRoute {
	stops := make([]api.DriverStop, 0, len(dr.Stops))
	for _, s := range dr.Stops {
		stops = append(stops, toDriverStop(s))
	}
	r := dr.Route
	return api.DriverRoute{
		Route: api.Route{Id: r.ID, RouteDate: r.RouteDate, DriverUserId: r.DriverUserID, DriverName: driverName, TruckLabel: r.TruckLabel,
			Status: api.RouteStatus(r.Status), OutAt: nullT(r.OutAt), CompletedAt: nullT(r.CompletedAt), Stops: []api.Stop{}},
		Stops: stops,
	}
}

func driverActionInput(in api.DriverAction) (domain.DriverAction, error) {
	a := domain.DriverAction{ClientID: in.ClientId, Type: domain.ActionType(in.Type), Note: deref(in.Note), SkipReason: deref(in.SkipReason)}
	if in.Proof != nil {
		p := &domain.Proof{Type: domain.ProofType(in.Proof.Type), Name: deref(in.Proof.Name)}
		if in.Proof.DataBase64 != nil {
			if len(*in.Proof.DataBase64) > (domain.MaxProofBytes*4/3)+4 {
				return a, service.Invalid(map[string]string{"proof": "image larger than 300 KB"})
			}
			data, err := base64.StdEncoding.DecodeString(*in.Proof.DataBase64)
			if err != nil {
				return a, service.Invalid(map[string]string{"proof": "dataBase64 is not valid base64"})
			}
			p.Data = data
		}
		a.Proof = p
	}
	if in.Lines != nil {
		for _, l := range *in.Lines {
			adj := domain.LineAdjustment{LineID: l.LineId, ShortageNote: l.ShortageNote}
			if l.DeliveredQty != nil {
				v := domain.Hundredths(*l.DeliveredQty)
				adj.DeliveredQty = &v
			}
			if l.DeliveredWeight != nil {
				v := domain.Hundredths(*l.DeliveredWeight)
				adj.DeliveredWeight = &v
			}
			a.Lines = append(a.Lines, adj)
		}
	}
	return a, nil
}
