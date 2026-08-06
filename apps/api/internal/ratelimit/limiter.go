// Package ratelimit provides a lightweight in-memory token-bucket rate
// limiter keyed by client. It uses only the Go standard library, is safe
// for concurrent use, and periodically removes idle clients to keep memory
// usage bounded.
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Limiter implements a token bucket per client key. Each bucket holds up to
// Burst tokens and is refilled at Rate tokens per second. Allow consumes one
// token when one is available.
type Limiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	now     func() time.Time
	clients map[string]*bucket
}

// bucket tracks the token balance for a single client.
type bucket struct {
	tokens   float64
	updated  time.Time
	lastSeen time.Time
}

// Options configures a Limiter.
type Options struct {
	// Rate is the sustained refill rate in tokens per second.
	Rate float64

	// Burst is the bucket capacity: the maximum number of tokens a client
	// can consume immediately, before refills accrue.
	Burst float64

	// Now returns the current time; defaults to time.Now. Injected in tests
	// for deterministic behavior.
	Now func() time.Time
}

// New returns an empty Limiter with the given rate and burst.
func New(opts Options) *Limiter {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Limiter{
		rate:    opts.Rate,
		burst:   opts.Burst,
		now:     opts.Now,
		clients: make(map[string]*bucket),
	}
}

// Allow reports whether the client identified by key may proceed now. When
// denied, the returned duration is the time the client must wait before the
// next request is allowed (usable for the Retry-After header).
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b, ok := l.clients[key]
	if !ok {
		b = &bucket{tokens: l.burst, updated: now}
		l.clients[key] = b
	}
	b.lastSeen = now
	b.tokens = min(l.burst, b.tokens+l.rate*now.Sub(b.updated).Seconds())
	b.updated = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	if l.rate <= 0 {
		return false, time.Hour
	}
	retryAfter := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	return false, retryAfter
}

// Cleanup removes buckets for clients that have not been seen for at least
// idle. It is safe to call concurrently with Allow.
func (l *Limiter) Cleanup(idle time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := l.now().Add(-idle)
	for key, b := range l.clients {
		if b.lastSeen.Before(cutoff) {
			delete(l.clients, key)
		}
	}
}

// StartCleanup launches a background goroutine that removes idle client
// buckets every interval, until ctx is cancelled. Any existing goroutine is
// left to exit via ctx.
func (l *Limiter) StartCleanup(ctx context.Context, interval, idle time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				l.Cleanup(idle)
			}
		}
	}()
}
