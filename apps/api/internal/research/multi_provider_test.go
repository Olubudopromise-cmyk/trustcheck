package research

import (
	"context"
	"errors"
	"testing"
	"time"
)

// blockingProvider never returns until the context is cancelled. It models a
// provider whose HTTP call hangs, so tests prove MultiProvider respects the
// request deadline instead of sleeping past it.
type blockingProvider struct{}

func (blockingProvider) Name() string { return "blocking" }
func (blockingProvider) SupportsMode(mode string) bool {
	return true
}
func (blockingProvider) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// failProvider always returns an error immediately.
type failProvider struct{}

func (failProvider) Name() string { return "failing" }
func (failProvider) SupportsMode(mode string) bool {
	return true
}
func (failProvider) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	return nil, errors.New("provider down")
}

func TestMultiProvider_FallsBackToNextProvider(t *testing.T) {
	mp := NewMultiProvider(failProvider{}, &MockSearchProvider{
		Results: []SearchResult{{Title: "Result", URL: "https://example.com", Domain: "example.com"}},
	})

	results, err := mp.Search(context.Background(), "test", 5)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result from fallback provider, got %d", len(results))
	}
}

func TestMultiProvider_AllProvidersFail(t *testing.T) {
	mp := NewMultiProvider(failProvider{}, failProvider{})

	_, err := mp.Search(context.Background(), "test", 5)
	if err == nil {
		t.Fatal("expected an error when all providers fail")
	}
	if !errors.Is(err, errors.New("provider down")) && err.Error() == "" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestMultiProvider_ContextCancellationProvesNoSleepPastDeadline ensures a
// hanging provider is abandoned as soon as the context fires — the previous
// implementation could sleep for seconds after cancellation, which is what
// pushed /verify past the serverless function deadline.
func TestMultiProvider_ContextCancellation(t *testing.T) {
	mp := NewMultiProvider(blockingProvider{}, blockingProvider{})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := mp.Search(ctx, "test", 5)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context error, got %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("cancellation took %s; provider ignored the deadline", elapsed)
	}
}

// TestMultiProvider_DeadlineBoundsHangingProvider simulates the full /verify
// research shape: a provider that hangs until the deadline. The search must
// return well inside any generous bound (here 5s) rather than burning the
// provider's own multi-second timeouts serially.
func TestMultiProvider_DeadlineBoundsHangingProvider(t *testing.T) {
	mp := NewMultiProvider(blockingProvider{}, blockingProvider{})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := mp.Search(ctx, "test", 5)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from a hanging provider")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("search took %s; the deadline was not honored", elapsed)
	}
}
