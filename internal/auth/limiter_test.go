package auth

import (
	"fmt"
	"testing"
	"time"
)

func TestLimiterSweepsAndCaps(t *testing.T) {
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := NewLimiter(10, time.Minute, func() time.Time { return clock })

	for i := 0; i < maxEntries+5; i++ {
		l.Allow(fmt.Sprintf("k%d", i))
	}
	before := len(l.hits)
	if before > maxEntries {
		t.Fatalf("map should be capped at %d, got %d", maxEntries, before)
	}

	clock = clock.Add(2 * time.Minute)
	l.Allow("after-window")
	after := len(l.hits)
	if after >= before {
		t.Fatalf("map should shrink once entries expire: before=%d after=%d", before, after)
	}
}
