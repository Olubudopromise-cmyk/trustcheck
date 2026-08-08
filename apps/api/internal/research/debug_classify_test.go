package research

import (
	"context"
	"fmt"
	"testing"
)

func TestDebugClassify(t *testing.T) {
	provider := NewDuckDuckGoProvider()
	engine := NewResearchEngine(provider, DefaultSearchConfig())
	ctx := context.Background()

	claim := "COVID-19 vaccines contain microchips for tracking people"
	keywords := extractKeywords(claim)
	evidence := engine.Research(ctx, claim, keywords)

	fmt.Printf("Claim: %s\n", claim)
	fmt.Printf("Total sources: %d\n", evidence.TotalSources)
	fmt.Printf("Supporting: %d\n", len(evidence.Supporting))
	fmt.Printf("Contradicting: %d\n", len(evidence.Contradicting))
	fmt.Printf("Neutral: %d\n", len(evidence.Neutral))

	for i, r := range evidence.Neutral {
		if i < 3 {
			fmt.Printf("\nNeutral %d:\n", i+1)
			fmt.Printf("  Title: %s\n", r.Title)
			fmt.Printf("  Snippet: %s\n", r.Snippet[:min(100, len(r.Snippet))])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
