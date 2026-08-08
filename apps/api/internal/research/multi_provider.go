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

// Search tries each provider in order until one succeeds.
func (p *MultiProvider) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	var lastErr error

	for i, provider := range p.providers {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Try up to 2 times per provider (for rate limiting)
		for attempt := 0; attempt < 2; attempt++ {
			results, err := provider.Search(ctx, query, maxResults)
			if err == nil && len(results) > 0 {
				log.Printf("[research] provider %s returned %d results for query: %s",
					provider.Name(), len(results), truncateStr(query, 50))
				return results, nil
			}

			if err != nil {
				lastErr = err
				log.Printf("[research] provider %s attempt %d failed: %v", provider.Name(), attempt+1, err)
				if attempt < 1 {
					time.Sleep(1 * time.Second) // Wait before retry
				}
				continue
			}

			// Provider returned no results
			if attempt < 1 {
				time.Sleep(500 * time.Millisecond)
			}
		}

		log.Printf("[research] provider %s exhausted, trying next", provider.Name())
		if i < len(p.providers)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed, last error: %w", lastErr)
	}
	return nil, nil
}

// truncateStr truncates a string to maxLen characters.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
