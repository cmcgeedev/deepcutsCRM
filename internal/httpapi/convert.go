package httpapi

import (
	"database/sql"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
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
