package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/db"
)

// newTestService opens a fresh migrated SQLite file in a temp dir with a fixed clock.
func newTestService(t *testing.T) *Service {
	t.Helper()
	d, err := db.OpenAndMigrate(context.Background(), filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	loc, _ := time.LoadLocation("America/New_York")
	s := New(d, loc)
	s.Now = func() time.Time { return time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC) } // 11:00 New York
	return s
}

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	e, ok := AsError(err)
	if !ok || e.Code != code {
		t.Fatalf("want error code %q, got %v", code, err)
	}
}
