package db

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestConstraints(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.sqlite")
	d, err := OpenAndMigrate(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	must := func(q string) {
		t.Helper()
		if _, err := d.Exec(q); err != nil {
			t.Fatalf("seed %q: %v", q, err)
		}
	}
	must(`INSERT INTO customers (id, name, created_at, updated_at) VALUES (1, 'Acme', '2026-01-01', '2026-01-01')`)
	must(`INSERT INTO users (id, realm, display_name, created_at) VALUES (1, 'driver', 'Bob', '2026-01-01')`)
	must(`INSERT INTO orders (id, customer_id, requested_delivery_date, status, created_at, updated_at) VALUES (1, 1, '2026-01-02', 'confirmed', '2026-01-01', '2026-01-01')`)
	must(`INSERT INTO delivery_routes (id, route_date, driver_user_id, status, created_at) VALUES (1, '2026-01-02', 1, 'planned', '2026-01-01')`)
	must(`INSERT INTO delivery_routes (id, route_date, driver_user_id, status, created_at) VALUES (2, '2026-01-02', 1, 'planned', '2026-01-01')`)
	must(`INSERT INTO delivery_stops (id, route_id, order_id, sequence, status) VALUES (1, 1, 1, 1, 'pending')`)

	// (a) a second pending stop for the same order on a different route violates delivery_stops_active_order.
	_, err = d.Exec(`INSERT INTO delivery_stops (id, route_id, order_id, sequence, status) VALUES (2, 2, 1, 1, 'pending')`)
	if err == nil || !strings.Contains(err.Error(), "UNIQUE") {
		t.Fatalf("expected UNIQUE constraint error, got %v", err)
	}

	// (b) skipping the first stop frees the order for a new pending stop.
	must(`UPDATE delivery_stops SET status = 'skipped' WHERE id = 1`)
	if _, err := d.Exec(`INSERT INTO delivery_stops (id, route_id, order_id, sequence, status) VALUES (2, 2, 1, 1, 'pending')`); err != nil {
		t.Fatalf("expected insert to succeed after skip, got %v", err)
	}

	// (c) an order referencing a non-existent customer violates the foreign key.
	_, err = d.Exec(`INSERT INTO orders (id, customer_id, requested_delivery_date, status, created_at, updated_at) VALUES (2, 999, '2026-01-02', 'confirmed', '2026-01-01', '2026-01-01')`)
	if err == nil || !strings.Contains(err.Error(), "FOREIGN KEY") {
		t.Fatalf("expected FOREIGN KEY constraint error, got %v", err)
	}

	// (d) a user with an unrecognized realm violates the CHECK constraint.
	_, err = d.Exec(`INSERT INTO users (id, realm, display_name, created_at) VALUES (2, 'cook', 'Chef', '2026-01-01')`)
	if err == nil || !strings.Contains(err.Error(), "CHECK") {
		t.Fatalf("expected CHECK constraint error, got %v", err)
	}
}
