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
	return e.count <= l.max
}

func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}
