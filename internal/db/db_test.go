package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.sqlite")
	d, err := OpenAndMigrate(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	var mode string
	if err := d.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("journal_mode=%q err=%v", mode, err)
	}
	var fk int
	if err := d.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil || fk != 1 {
		t.Fatalf("foreign_keys=%d err=%v", fk, err)
	}
	var busyTimeout int
	if err := d.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil || busyTimeout != 5000 {
		t.Fatalf("busy_timeout=%d err=%v", busyTimeout, err)
	}
	var n int
	if err := d.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name IN ('customers','users','sessions','products','customer_prices','orders','order_lines','delivery_routes','delivery_stops','driver_actions')").Scan(&n); err != nil || n != 10 {
		t.Fatalf("tables=%d err=%v", n, err)
	}
	// migrate again is a no-op
	if err := Migrate(context.Background(), d); err != nil {
		t.Fatal(err)
	}
}
