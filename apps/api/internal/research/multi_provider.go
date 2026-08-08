package research

import (
	"context"
	"fmt"
	"log"
	"time"
)

// MultiProvider tries multiple search providers in order, falling back to
// the next one if the previous fails. This ensures we always get some
// results even if one provider is rate-limited or unavailable.
//
// All waiting is context-aware: when the request context is cancelled (for
// example the serverless function deadline), in-flight provider calls are
// aborted and backoff timers return immediately, so a slow provider can
// never push the overall request past the platform timeout.
type MultiProvider struct {
	providers []SearchProvider
}

// NewMultiProvider creates a new multi-provider with the given providers.
func NewMultiProvider(providers ...SearchProvider) *MultiProvider {
	return &MultiProvider{
		providers: providers,
	}
}

// Name returns the provider name.
func (p *MultiProvider) Name() string {
	return "multi"
}

// SupportsMode indicates if this provider supports the given analysis mode.
func (p *MultiProvider) SupportsMode(mode string) bool {
	return true
}

// Search tries each provider in order until one succeeds. A provider that
// errors or returns no results is skipped — the failure is never escalated:
// callers decide how to report it. The context deadline is respected at
// every step, so this function returns at or before ctx expiry.
func (p *MultiProvider) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	var lastErr error

	for i, provider := range p.providers {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		results, err := provider.Search(ctx, query, maxResults)
		if err == nil && len(results) > 0 {
			log.Printf("[research] provider %s returned %d results for query: %s",
				provider.Name(), len(results), truncateStr(query, 50))
			return results, nil
		}

		if err != nil {
			lastErr = err
			log.Printf("[research] provider %s failed: %v", provider.Name(), err)
		} else {
			log.Printf("[research] provider %s returned no results for query: %s",
				provider.Name(), truncateStr(query, 50))
		}

		// Small backoff before the next provider, aborted if the context
		// expires while waiting.
		if i < len(p.providers)-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(300 * time.Millisecond):
			}
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed, last error: %w", lastErr)
	}
	return nil, nil
}

// truncateStr truncates a string to maxLen characters without splitting a
// multi-byte rune, preserving UTF-8 correctness.
func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
