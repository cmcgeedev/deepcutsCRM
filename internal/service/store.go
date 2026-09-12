package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
)

type Service struct {
	DB  *sql.DB
	Q   *queries.Queries
	Now func() time.Time
	Loc *time.Location
}

func New(d *sql.DB, loc *time.Location) *Service {
	return &Service{DB: d, Q: queries.New(d), Now: time.Now, Loc: loc}
}

// Tx runs fn inside a transaction. All queries inside fn must go through q.
func (s *Service) Tx(ctx context.Context, fn func(q *queries.Queries) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(s.Q.WithTx(tx)); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Service) now() time.Time { return s.Now().UTC() }

// Today is the business-local date.
func (s *Service) Today() string { return s.Now().In(s.Loc).Format("2006-01-02") }

func ValidDate(d string) bool {
	_, err := time.Parse("2006-01-02", d)
	return err == nil
}

var weekdays = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

func ParseDeliveryDays(days []string) (string, error) {
	var out []string
	for _, d := range days {
		d = strings.ToLower(strings.TrimSpace(d))
		ok := false
		for _, w := range weekdays {
			if w == d {
				ok = true
			}
		}
		if !ok {
			return "", fmt.Errorf("unknown weekday %q", d)
		}
		out = append(out, d)
	}
	return strings.Join(out, ","), nil
}

func SplitDeliveryDays(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

func nullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }
func strOf(n sql.NullString) string   { return n.String }
func nullInt(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}
func intPtr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}
func hPtr(n sql.NullInt64) *domain.Hundredths {
	if !n.Valid {
		return nil
	}
	v := domain.Hundredths(n.Int64)
	return &v
}
func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func productOf(p queries.Product) domain.Product {
	return domain.Product{
		ID: p.ID, SKU: p.Sku, Name: p.Name, SellUnit: domain.Unit(p.SellUnit),
		CatchWeight: p.CatchWeight, ApproxCaseWeight: domain.Hundredths(p.ApproxCaseWeight.Int64),
		BasePrice: domain.Cents(p.BasePriceCents),
	}
}
