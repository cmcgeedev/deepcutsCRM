package service

import (
	"context"
	"strings"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
)

type CustomerInput struct {
	Name, BillingAddress, DeliveryAddress, ContactName, Phone, Email, DeliveryNotes string
	DeliveryDays                                                                    []string
	QBOCustomerID                                                                   string
	Active                                                                          bool
}

func (in CustomerInput) validate() (days string, err error) {
	fields := map[string]string{}
	if strings.TrimSpace(in.Name) == "" {
		fields["name"] = "required"
	}
	days, derr := ParseDeliveryDays(in.DeliveryDays)
	if derr != nil {
		fields["deliveryDays"] = derr.Error()
	}
	if len(fields) > 0 {
		return "", Invalid(fields)
	}
	return days, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (s *Service) CreateCustomer(ctx context.Context, in CustomerInput) (queries.Customer, error) {
	days, err := in.validate()
	if err != nil {
		return queries.Customer{}, err
	}
	now := s.now()
	c, err := s.Q.CreateCustomer(ctx, queries.CreateCustomerParams{
		Name: strings.TrimSpace(in.Name), BillingAddress: in.BillingAddress, DeliveryAddress: in.DeliveryAddress,
		ContactName: in.ContactName, Phone: in.Phone, Email: in.Email, DeliveryNotes: in.DeliveryNotes,
		DeliveryDays: days, QboCustomerID: nullStr(in.QBOCustomerID), CreatedAt: now, UpdatedAt: now,
	})
	if isUniqueViolation(err) {
		return queries.Customer{}, Conflict("duplicate", "a customer with that QuickBooks id already exists")
	}
	return c, err
}

func (s *Service) UpdateCustomer(ctx context.Context, id int64, in CustomerInput) (queries.Customer, error) {
	days, err := in.validate()
	if err != nil {
		return queries.Customer{}, err
	}
	if _, err := s.Q.GetCustomer(ctx, id); err != nil {
		return queries.Customer{}, notFoundIf(err, "customer")
	}
	c, err := s.Q.UpdateCustomer(ctx, queries.UpdateCustomerParams{
		Name: strings.TrimSpace(in.Name), BillingAddress: in.BillingAddress, DeliveryAddress: in.DeliveryAddress,
		ContactName: in.ContactName, Phone: in.Phone, Email: in.Email, DeliveryNotes: in.DeliveryNotes,
		DeliveryDays: days, QboCustomerID: nullStr(in.QBOCustomerID), Active: in.Active, UpdatedAt: s.now(), ID: id,
	})
	if isUniqueViolation(err) {
		return queries.Customer{}, Conflict("duplicate", "a customer with that QuickBooks id already exists")
	}
	return c, err
}

func (s *Service) GetCustomer(ctx context.Context, id int64) (queries.Customer, error) {
	c, err := s.Q.GetCustomer(ctx, id)
	return c, notFoundIf(err, "customer")
}

func (s *Service) ListCustomers(ctx context.Context, includeInactive bool) ([]queries.Customer, error) {
	return s.Q.ListCustomers(ctx, includeInactive)
}

type ProductInput struct {
	SKU, Name, Category, SellUnit string
	CatchWeight                   bool
	ApproxCaseWeight              *int64
	BasePriceCents                int64
	QBOItemID                     string
	Active                        bool
}

func (in ProductInput) domain() (domain.Product, error) {
	p := domain.Product{
		SKU: strings.TrimSpace(in.SKU), Name: strings.TrimSpace(in.Name), SellUnit: domain.Unit(in.SellUnit),
		CatchWeight: in.CatchWeight, BasePrice: domain.Cents(in.BasePriceCents),
	}
	if in.ApproxCaseWeight != nil {
		p.ApproxCaseWeight = domain.Hundredths(*in.ApproxCaseWeight)
	}
	if errs := p.Validate(); len(errs) > 0 {
		return p, Invalid(errs)
	}
	return p, nil
}

func (s *Service) CreateProduct(ctx context.Context, in ProductInput) (queries.Product, error) {
	p, err := in.domain()
	if err != nil {
		return queries.Product{}, err
	}
	now := s.now()
	row, err := s.Q.CreateProduct(ctx, queries.CreateProductParams{
		Sku: p.SKU, Name: p.Name, Category: in.Category, SellUnit: string(p.SellUnit), CatchWeight: p.CatchWeight,
		ApproxCaseWeight: nullInt(in.ApproxCaseWeight), BasePriceCents: in.BasePriceCents, QboItemID: nullStr(in.QBOItemID),
		CreatedAt: now, UpdatedAt: now,
	})
	if isUniqueViolation(err) {
		return queries.Product{}, Conflict("duplicate", "a product with that SKU already exists")
	}
	return row, err
}

func (s *Service) UpdateProduct(ctx context.Context, id int64, in ProductInput) (queries.Product, error) {
	p, err := in.domain()
	if err != nil {
		return queries.Product{}, err
	}
	if _, err := s.Q.GetProduct(ctx, id); err != nil {
		return queries.Product{}, notFoundIf(err, "product")
	}
	row, err := s.Q.UpdateProduct(ctx, queries.UpdateProductParams{
		Sku: p.SKU, Name: p.Name, Category: in.Category, SellUnit: string(p.SellUnit), CatchWeight: p.CatchWeight,
		ApproxCaseWeight: nullInt(in.ApproxCaseWeight), BasePriceCents: in.BasePriceCents, QboItemID: nullStr(in.QBOItemID),
		Active: in.Active, UpdatedAt: s.now(), ID: id,
	})
	if isUniqueViolation(err) {
		return queries.Product{}, Conflict("duplicate", "a product with that SKU already exists")
	}
	return row, err
}

func (s *Service) GetProduct(ctx context.Context, id int64) (queries.Product, error) {
	p, err := s.Q.GetProduct(ctx, id)
	return p, notFoundIf(err, "product")
}

func (s *Service) ListProducts(ctx context.Context, includeInactive bool) ([]queries.Product, error) {
	return s.Q.ListProducts(ctx, includeInactive)
}

type PriceInput struct {
	ProductID     int64
	PriceCents    int64
	EffectiveFrom string
}

func (s *Service) SetCustomerPrice(ctx context.Context, customerID int64, in PriceInput) (queries.CustomerPrice, error) {
	fields := map[string]string{}
	if !ValidDate(in.EffectiveFrom) {
		fields["effectiveFrom"] = "must be YYYY-MM-DD"
	}
	if in.PriceCents < 0 {
		fields["priceCents"] = "must not be negative"
	}
	if len(fields) > 0 {
		return queries.CustomerPrice{}, Invalid(fields)
	}
	if _, err := s.Q.GetCustomer(ctx, customerID); err != nil {
		return queries.CustomerPrice{}, notFoundIf(err, "customer")
	}
	if _, err := s.Q.GetProduct(ctx, in.ProductID); err != nil {
		return queries.CustomerPrice{}, notFoundIf(err, "product")
	}
	return s.Q.CreateCustomerPrice(ctx, queries.CreateCustomerPriceParams{
		CustomerID: customerID, ProductID: in.ProductID, PriceCents: in.PriceCents, EffectiveFrom: in.EffectiveFrom, CreatedAt: s.now(),
	})
}

func (s *Service) ListCustomerPrices(ctx context.Context, customerID int64, asOf string) ([]queries.ListCurrentCustomerPricesRow, error) {
	if _, err := s.Q.GetCustomer(ctx, customerID); err != nil {
		return nil, notFoundIf(err, "customer")
	}
	return s.Q.ListCurrentCustomerPrices(ctx, queries.ListCurrentCustomerPricesParams{CustomerID: customerID, AsOf: asOf})
}

// resolvePrice applies domain.ResolvePrice using the customer's price rows. q may be transaction-bound.
func (s *Service) resolvePrice(ctx context.Context, q *queries.Queries, customerID, productID int64, base domain.Cents, date string) (domain.Cents, bool, error) {
	rows, err := q.ListCustomerPriceRows(ctx, queries.ListCustomerPriceRowsParams{CustomerID: customerID, ProductID: productID})
	if err != nil {
		return 0, false, err
	}
	prs := make([]domain.PriceRow, len(rows))
	for i, r := range rows {
		prs[i] = domain.PriceRow{Price: domain.Cents(r.PriceCents), EffectiveFrom: r.EffectiveFrom}
	}
	p, custom := domain.ResolvePrice(base, prs, date)
	return p, custom, nil
}
