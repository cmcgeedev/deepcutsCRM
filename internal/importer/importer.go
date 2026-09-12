// Package importer loads customers, products and prices from CSV. It never deletes and is re-run safe.
package importer

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

type Summary struct {
	Created, Updated, Rejected int
	Errors                     []string
}

func (s Summary) String() string {
	out := fmt.Sprintf("created %d, updated %d, rejected %d", s.Created, s.Updated, s.Rejected)
	for _, e := range s.Errors {
		out += "\n  " + e
	}
	return out
}

func (s *Summary) reject(line int, msg string) {
	s.Rejected++
	s.Errors = append(s.Errors, fmt.Sprintf("line %d: %s", line, msg))
}

// table reads a CSV into rows keyed by lower-cased header.
type table struct {
	headers []string
	rows    []map[string]string
}

func readTable(r io.Reader, required ...string) (table, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	recs, err := cr.ReadAll()
	if err != nil {
		return table{}, err
	}
	if len(recs) == 0 {
		return table{}, errors.New("empty file")
	}
	t := table{}
	for _, h := range recs[0] {
		t.headers = append(t.headers, strings.ToLower(strings.TrimSpace(h)))
	}
	for _, req := range required {
		found := false
		for _, h := range t.headers {
			if h == req {
				found = true
			}
		}
		if !found {
			return table{}, fmt.Errorf("missing required column %q", req)
		}
	}
	for _, rec := range recs[1:] {
		row := map[string]string{}
		for i, h := range t.headers {
			if i < len(rec) {
				row[h] = strings.TrimSpace(rec[i])
			}
		}
		t.rows = append(t.rows, row)
	}
	return t, nil
}

func (t table) has(col string) bool {
	for _, h := range t.headers {
		if h == col {
			return true
		}
	}
	return false
}

// decimalToHundredths parses "5.99" → 599 (also used for weights: "60" → 6000).
func decimalToHundredths(s string) (int64, error) {
	f, err := strconv.ParseFloat(strings.TrimPrefix(s, "$"), 64)
	if err != nil || f < 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("%q is not a non-negative decimal", s)
	}
	return int64(math.Round(f * 100)), nil
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "yes", "y", "1":
		return true, nil
	case "false", "no", "n", "0", "":
		return false, nil
	}
	return false, fmt.Errorf("%q is not a boolean", s)
}

func nullable(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func Customers(ctx context.Context, svc *service.Service, r io.Reader) (Summary, error) {
	t, err := readTable(r, "name")
	if err != nil {
		return Summary{}, err
	}
	var sum Summary
	for i, row := range t.rows {
		line := i + 2
		if row["name"] == "" {
			sum.reject(line, "name is required")
			continue
		}
		var existing *queries.Customer
		if q := row["qbo_customer_id"]; q != "" {
			if c, err := svc.Q.GetCustomerByQBO(ctx, nullable(q)); err == nil {
				existing = &c
			}
		}
		if existing == nil {
			if c, err := svc.Q.GetCustomerByName(ctx, row["name"]); err == nil {
				existing = &c
			}
		}
		in := service.CustomerInput{Active: true}
		if existing != nil {
			in = service.CustomerInput{
				Name: existing.Name, BillingAddress: existing.BillingAddress, DeliveryAddress: existing.DeliveryAddress, ContactName: existing.ContactName,
				Phone: existing.Phone, Email: existing.Email, DeliveryNotes: existing.DeliveryNotes, DeliveryDays: service.SplitDeliveryDays(existing.DeliveryDays),
				QBOCustomerID: existing.QboCustomerID.String, Active: existing.Active,
			}
		}
		in.Name = row["name"]
		set := func(col string, dst *string) {
			if t.has(col) {
				*dst = row[col]
			}
		}
		set("billing_address", &in.BillingAddress)
		set("delivery_address", &in.DeliveryAddress)
		set("contact_name", &in.ContactName)
		set("phone", &in.Phone)
		set("email", &in.Email)
		set("delivery_notes", &in.DeliveryNotes)
		set("qbo_customer_id", &in.QBOCustomerID)
		if t.has("delivery_days") && row["delivery_days"] != "" {
			in.DeliveryDays = strings.Split(strings.ReplaceAll(row["delivery_days"], ",", ";"), ";")
		}
		if existing == nil {
			_, err = svc.CreateCustomer(ctx, in)
		} else {
			_, err = svc.UpdateCustomer(ctx, existing.ID, in)
		}
		if err != nil {
			sum.reject(line, err.Error())
			continue
		}
		if existing == nil {
			sum.Created++
		} else {
			sum.Updated++
		}
	}
	return sum, nil
}

func Products(ctx context.Context, svc *service.Service, r io.Reader) (Summary, error) {
	t, err := readTable(r, "sku", "name")
	if err != nil {
		return Summary{}, err
	}
	var sum Summary
	for i, row := range t.rows {
		line := i + 2
		var existing *queries.Product
		if p, err := svc.Q.GetProductBySKU(ctx, row["sku"]); err == nil {
			existing = &p
		}
		in := service.ProductInput{SKU: row["sku"], Active: true}
		if existing != nil {
			in = service.ProductInput{SKU: existing.Sku, Name: existing.Name, Category: existing.Category, SellUnit: existing.SellUnit, CatchWeight: existing.CatchWeight,
				BasePriceCents: existing.BasePriceCents, QBOItemID: existing.QboItemID.String, Active: existing.Active}
			if existing.ApproxCaseWeight.Valid {
				w := existing.ApproxCaseWeight.Int64
				in.ApproxCaseWeight = &w
			}
		}
		in.Name = row["name"]
		if t.has("category") {
			in.Category = row["category"]
		}
		if t.has("sell_unit") {
			in.SellUnit = strings.ToLower(row["sell_unit"])
		}
		if t.has("qbo_item_id") {
			in.QBOItemID = row["qbo_item_id"]
		}
		var perr error
		if t.has("catch_weight") {
			in.CatchWeight, perr = parseBool(row["catch_weight"])
		}
		if perr == nil && t.has("approx_case_weight_lb") && row["approx_case_weight_lb"] != "" {
			var w int64
			w, perr = decimalToHundredths(row["approx_case_weight_lb"])
			in.ApproxCaseWeight = &w
		}
		if perr == nil && t.has("base_price") && row["base_price"] != "" {
			in.BasePriceCents, perr = decimalToHundredths(row["base_price"])
		}
		if perr != nil {
			sum.reject(line, perr.Error())
			continue
		}
		if existing == nil {
			_, err = svc.CreateProduct(ctx, in)
		} else {
			_, err = svc.UpdateProduct(ctx, existing.ID, in)
		}
		if err != nil {
			sum.reject(line, err.Error())
			continue
		}
		if existing == nil {
			sum.Created++
		} else {
			sum.Updated++
		}
	}
	return sum, nil
}

func Prices(ctx context.Context, svc *service.Service, r io.Reader, effectiveFrom string) (Summary, error) {
	if !service.ValidDate(effectiveFrom) {
		return Summary{}, fmt.Errorf("effective date %q must be YYYY-MM-DD", effectiveFrom)
	}
	t, err := readTable(r, "customer", "sku", "price")
	if err != nil {
		return Summary{}, err
	}
	var sum Summary
	for i, row := range t.rows {
		line := i + 2
		c, err := svc.Q.GetCustomerByQBO(ctx, nullable(row["customer"]))
		if err != nil {
			c, err = svc.Q.GetCustomerByName(ctx, row["customer"])
		}
		if err != nil {
			sum.reject(line, fmt.Sprintf("customer %q not found", row["customer"]))
			continue
		}
		p, err := svc.Q.GetProductBySKU(ctx, row["sku"])
		if err != nil {
			sum.reject(line, fmt.Sprintf("sku %q not found", row["sku"]))
			continue
		}
		cents, err := decimalToHundredths(row["price"])
		if err != nil {
			sum.reject(line, err.Error())
			continue
		}
		if _, err := svc.SetCustomerPrice(ctx, c.ID, service.PriceInput{ProductID: p.ID, PriceCents: cents, EffectiveFrom: effectiveFrom}); err != nil {
			sum.reject(line, err.Error())
			continue
		}
		sum.Created++
	}
	return sum, nil
}
