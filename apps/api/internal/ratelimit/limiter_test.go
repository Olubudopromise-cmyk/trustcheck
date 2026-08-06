package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func newTestLimiter(rate, burst float64, now func() time.Time) *Limiter {
	return New(Options{Rate: rate, Burst: burst, Now: now})
}

func TestAllowWithinBurst(t *testing.T) {
	now := time.Unix(0, 0)
	l := newTestLimiter(1, 3, func() time.Time { return now })

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("1.2.3.4"); !ok {
			t.Fatalf("request %d should be allowed within burst", i+1)
		}
	}
	if ok, _ := l.Allow("1.2.3.4"); ok {
		t.Fatal("request beyond burst should be denied")
	}
}

func TestRefillRestoresTokens(t *testing.T) {
	now := time.Unix(0, 0)
	l := newTestLimiter(1, 3, func() time.Time { return now })

	for i := 0; i < 3; i++ {
		l.Allow("1.2.3.4")
	}

	now = now.Add(time.Second)
	if ok, _ := l.Allow("1.2.3.4"); !ok {
		t.Fatal("expected a refilled token to be available after 1 second")
	}

	now = now.Add(4 * time.Second)
	if ok, _ := l.Allow("1.2.3.4"); !ok {
		t.Fatal("expected the bucket to refill back toward its burst capacity")
	}
}

func TestRetryAfter(t *testing.T) {
	now := time.Unix(0, 0)
	l := newTestLimiter(2, 1, func() time.Time { return now })

	l.Allow("1.2.3.4")
	_, retryAfter := l.Allow("1.2.3.4")

	if retryAfter <= 0 || retryAfter > 500*time.Millisecond {
		t.Fatalf("expected retryAfter ~500ms for rate 2 tokens/s, got %v", retryAfter)
	}
}

func TestClientsAreIndependent(t *testing.T) {
	now := time.Unix(0, 0)
	l := newTestLimiter(1, 1, func() time.Time { return now })

	if ok, _ := l.Allow("client-a"); !ok {
		t.Fatal("client-a first request should be allowed")
	}
	if ok, _ := l.Allow("client-a"); ok {
		t.Fatal("client-a should be exhausted after its burst")
	}
	if ok, _ := l.Allow("client-b"); !ok {
		t.Fatal("client-b should have its own bucket")
	}
}

func TestCleanupRemovesIdleClients(t *testing.T) {
	now := time.Unix(0, 0)
	l := newTestLimiter(1, 3, func() time.Time { return now })

	l.Allow("1.2.3.4")
	l.Allow("5.6.7.8")
	if len(l.clients) != 2 {
		t.Fatalf("expected 2 client buckets, got %d", len(l.clients))
	}

	now = now.Add(10 * time.Minute)
	l.Cleanup(time.Minute)
	if len(l.clients) != 0 {
		t.Fatalf("expected idle clients to be removed, got %d", len(l.clients))
	}
}

func TestCleanupKeepsActiveClients(t *testing.T) {
	now := time.Unix(0, 0)
	l := newTestLimiter(1, 3, func() time.Time { return now })

	l.Allow("active")
	now = now.Add(30 * time.Second)
	l.Allow("active")

	now = now.Add(30 * time.Second)
	l.Cleanup(time.Minute)
	if len(l.clients) != 1 {
		t.Fatalf("expected the active client to be kept, got %d", len(l.clients))
	}
}

func TestStartCleanupStopsAndCleans(t *testing.T) {
	l := New(Options{Rate: 1, Burst: 3})
	l.Allow("1.2.3.4")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l.StartCleanup(ctx, time.Millisecond, time.Millisecond)

	deadline := time.Now().Add(2 * time.Second)
	for {
		l.mu.Lock()
		n := len(l.clients)
		l.mu.Unlock()
		if n == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("cleanup goroutine did not remove the idle client in time")
		}
		time.Sleep(time.Millisecond)
	}

	cancel()
}

func TestConcurrentAllow(t *testing.T) {
	now := time.Unix(0, 0)
	l := newTestLimiter(0, 100, func() time.Time { return now })

	const (
		workers = 10
		calls   = 20
	)
	var wg sync.WaitGroup
	granted := make(chan bool, workers*calls)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < calls; i++ {
				ok, _ := l.Allow("1.2.3.4")
				granted <- ok
			}
		}()
	}
	wg.Wait()
	close(granted)

	var total int
	for ok := range granted {
		if ok {
			total++
		}
	}
	if total != 100 {
		t.Fatalf("expected exactly the burst capacity of 100 granted under concurrency, got %d", total)
	}
}
