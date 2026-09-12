package httpapi

import (
	"context"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func (s *Server) ListCustomers(ctx context.Context, r api.ListCustomersRequestObject) (api.ListCustomersResponseObject, error) {
	rows, err := s.d.Svc.ListCustomers(ctx, derefBool(r.Params.IncludeInactive, false))
	if err != nil {
		return nil, err
	}
	out := make(api.ListCustomers200JSONResponse, 0, len(rows))
	for _, c := range rows {
		out = append(out, toCustomer(c))
	}
	return out, nil
}

func (s *Server) CreateCustomer(ctx context.Context, r api.CreateCustomerRequestObject) (api.CreateCustomerResponseObject, error) {
	c, err := s.d.Svc.CreateCustomer(ctx, customerInput(*r.Body))
	if err != nil {
		return nil, err
	}
	return api.CreateCustomer201JSONResponse(toCustomer(c)), nil
}

func (s *Server) GetCustomer(ctx context.Context, r api.GetCustomerRequestObject) (api.GetCustomerResponseObject, error) {
	c, err := s.d.Svc.GetCustomer(ctx, r.CustomerId)
	if err != nil {
		return nil, err
	}
	return api.GetCustomer200JSONResponse(toCustomer(c)), nil
}

func (s *Server) UpdateCustomer(ctx context.Context, r api.UpdateCustomerRequestObject) (api.UpdateCustomerResponseObject, error) {
	c, err := s.d.Svc.UpdateCustomer(ctx, r.CustomerId, customerInput(*r.Body))
	if err != nil {
		return nil, err
	}
	return api.UpdateCustomer200JSONResponse(toCustomer(c)), nil
}

func (s *Server) ListCustomerPrices(ctx context.Context, r api.ListCustomerPricesRequestObject) (api.ListCustomerPricesResponseObject, error) {
	asOf := deref(r.Params.AsOf)
	if asOf == "" {
		asOf = s.d.Svc.Today()
	}
	if !service.ValidDate(asOf) {
		return nil, service.Invalid(map[string]string{"asOf": "must be YYYY-MM-DD"})
	}
	rows, err := s.d.Svc.ListCustomerPrices(ctx, r.CustomerId, asOf)
	if err != nil {
		return nil, err
	}
	out := make(api.ListCustomerPrices200JSONResponse, 0, len(rows))
	for _, p := range rows {
		out = append(out, toPriceRow(p.ID, p.ProductID, p.Sku, p.ProductName, p.PriceCents, p.EffectiveFrom))
	}
	return out, nil
}

func (s *Server) SetCustomerPrice(ctx context.Context, r api.SetCustomerPriceRequestObject) (api.SetCustomerPriceResponseObject, error) {
	p, err := s.d.Svc.SetCustomerPrice(ctx, r.CustomerId, service.PriceInput{ProductID: r.Body.ProductId, PriceCents: r.Body.PriceCents, EffectiveFrom: r.Body.EffectiveFrom})
	if err != nil {
		return nil, err
	}
	prod, err := s.d.Svc.GetProduct(ctx, p.ProductID)
	if err != nil {
		return nil, err
	}
	return api.SetCustomerPrice201JSONResponse(toPriceRow(p.ID, p.ProductID, prod.Sku, prod.Name, p.PriceCents, p.EffectiveFrom)), nil
}

func (s *Server) ListProducts(ctx context.Context, r api.ListProductsRequestObject) (api.ListProductsResponseObject, error) {
	rows, err := s.d.Svc.ListProducts(ctx, derefBool(r.Params.IncludeInactive, false))
	if err != nil {
		return nil, err
	}
	out := make(api.ListProducts200JSONResponse, 0, len(rows))
	for _, p := range rows {
		out = append(out, toProduct(p))
	}
	return out, nil
}

func (s *Server) CreateProduct(ctx context.Context, r api.CreateProductRequestObject) (api.CreateProductResponseObject, error) {
	p, err := s.d.Svc.CreateProduct(ctx, productInput(*r.Body))
	if err != nil {
		return nil, err
	}
	return api.CreateProduct201JSONResponse(toProduct(p)), nil
}

func (s *Server) GetProduct(ctx context.Context, r api.GetProductRequestObject) (api.GetProductResponseObject, error) {
	p, err := s.d.Svc.GetProduct(ctx, r.ProductId)
	if err != nil {
		return nil, err
	}
	return api.GetProduct200JSONResponse(toProduct(p)), nil
}

func (s *Server) UpdateProduct(ctx context.Context, r api.UpdateProductRequestObject) (api.UpdateProductResponseObject, error) {
	p, err := s.d.Svc.UpdateProduct(ctx, r.ProductId, productInput(*r.Body))
	if err != nil {
		return nil, err
	}
	return api.UpdateProduct200JSONResponse(toProduct(p)), nil
}
