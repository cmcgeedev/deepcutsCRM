package auth

import (
	"sync"
	"time"
)

// Limiter is a fixed-window counter: at most max attempts per key per window.
type Limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	now    func() time.Time
	hits   map[string]entry
}

type entry struct {
	count int
	start time.Time
}

// maxEntries caps the limiter's memory: past this size, Allow sweeps expired
// entries and, if still over the cap, evicts the oldest ones.
const maxEntries = 10000

func NewLimiter(max int, window time.Duration, now func() time.Time) *Limiter {
	return &Limiter{max: max, window: window, now: now, hits: map[string]entry{}}
}

// Allow records an attempt and reports whether it is within the limit.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	t := l.now()
	e := l.hits[key]
	if e.start.IsZero() || t.Sub(e.start) > l.window {
		e = entry{start: t}
	}
	e.count++
	l.hits[key] = e
	if len(l.hits) > maxEntries {
		l.sweep(t)
	}
	return e.count <= l.max
}

// sweep removes entries whose window has elapsed; if the map is still over
// maxEntries afterward, it evicts the oldest entries (by start) until it isn't.
func (l *Limiter) sweep(t time.Time) {
	for k, e := range l.hits {
		if t.Sub(e.start) > l.window {
			delete(l.hits, k)
		}
	}
	for len(l.hits) > maxEntries {
		var oldestKey string
		var oldestStart time.Time
		first := true
		for k, e := range l.hits {
			if first || e.start.Before(oldestStart) {
				oldestKey, oldestStart, first = k, e.start, false
			}
		}
		delete(l.hits, oldestKey)
	}
}

func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}
