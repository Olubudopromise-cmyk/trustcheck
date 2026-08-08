package research

import (
	"context"
	"fmt"
	"strings"
)

// ResearchEngine orchestrates searches using a SearchProvider and classifies
// the results as supporting, contradicting, or neutral evidence.
type ResearchEngine struct {
	provider SearchProvider
	config   SearchConfig
}

// NewResearchEngine creates a new research engine with the given provider.
func NewResearchEngine(provider SearchProvider, config SearchConfig) *ResearchEngine {
	return &ResearchEngine{
		provider: provider,
		config:   config,
	}
}

// Evidence represents classified evidence from a search.
type Evidence struct {
	// Supporting contains results that support the claim.
	Supporting []SearchResult

	// Contradicting contains results that contradict the claim.
	Contradicting []SearchResult

	// Neutral contains results that neither support nor contradict.
	Neutral []SearchResult

	// TotalSources is the total number of unique sources found.
	TotalSources int

	// IndependentCount is the number of independent sources.
	IndependentCount int

	// SearchErrors contains any errors encountered during search.
	SearchErrors []error

	// SearchStatus indicates the overall status of the research.
	SearchStatus SearchStatus
}

// SearchStatus indicates the status of the research.
type SearchStatus string

const (
	SearchStatusComplete    SearchStatus = "complete"
	SearchStatusPartial     SearchStatus = "partial"     // Some searches failed
	SearchStatusNoResults   SearchStatus = "no_results"  // No results found
	SearchStatusFailed      SearchStatus = "failed"      // All searches failed
	SearchStatusDisabled    SearchStatus = "disabled"    // Research disabled for this input type
)

// Research performs a full research cycle for a claim.
// It generates search queries, performs searches, and classifies the results.
func (e *ResearchEngine) Research(ctx context.Context, claim string, keywords []string) *Evidence {
	evidence := &Evidence{
		SearchStatus: SearchStatusComplete,
	}

	// Generate search queries based on the claim and mode
	queries := e.generateQueries(claim, keywords)

	// Perform searches
	var allResults []SearchResult
	for _, query := range queries {
		select {
		case <-ctx.Done():
			evidence.SearchErrors = append(evidence.SearchErrors, ctx.Err())
			evidence.SearchStatus = SearchStatusPartial
			return evidence
		default:
		}

		results, err := e.provider.Search(ctx, query, e.config.MaxResults)
		if err != nil {
			evidence.SearchErrors = append(evidence.SearchErrors, fmt.Errorf("search %q: %w", query, err))
			evidence.SearchStatus = SearchStatusPartial
			continue
		}
		allResults = append(allResults, results...)
	}

	// Deduplicate results
	allResults = deduplicateResults(allResults)

	if len(allResults) == 0 {
		// If we had errors, it's a failure, not just no results
		if len(evidence.SearchErrors) > 0 {
			evidence.SearchStatus = SearchStatusFailed
		} else {
			evidence.SearchStatus = SearchStatusNoResults
		}
		return evidence
	}

	// Classify evidence
	evidence.Supporting, evidence.Contradicting, evidence.Neutral = classifyEvidence(allResults, claim)

	// Calculate counts
	evidence.TotalSources = len(allResults)
	evidence.IndependentCount = countIndependentSources(allResults)

	return evidence
}

// generateQueries generates search queries based on the claim and mode.
func (e *ResearchEngine) generateQueries(claim string, keywords []string) []string {
	var queries []string

	switch e.config.Mode {
	case "deep_research":
		// Deep mode: multiple query formulations
		queries = append(queries, claim)
		queries = append(queries, fmt.Sprintf("fact check %s", claim))
		queries = append(queries, fmt.Sprintf("evidence for %s", claim))
		queries = append(queries, fmt.Sprintf("evidence against %s", claim))
		if len(keywords) > 0 {
			queries = append(queries, strings.Join(keywords, " "))
		}

	case "government_official":
		// Government mode: prioritize official sources
		queries = append(queries, claim)
		queries = append(queries, fmt.Sprintf("official %s", claim))
		queries = append(queries, fmt.Sprintf("site:.gov %s", claim))
		if len(keywords) > 0 {
			queries = append(queries, strings.Join(keywords, " "))
		}

	default: // quick
		// Quick mode: fewer searches
		queries = append(queries, claim)
		if len(keywords) > 0 {
			queries = append(queries, strings.Join(keywords, " "))
		}
	}

	// Limit queries
	if len(queries) > e.config.MaxQueries {
		queries = queries[:e.config.MaxQueries]
	}

	return queries
}

// deduplicateResults removes duplicate results based on URL.
func deduplicateResults(results []SearchResult) []SearchResult {
	seen := make(map[string]bool)
	var deduped []SearchResult

	for _, r := range results {
		if !seen[r.URL] {
			seen[r.URL] = true
			deduped = append(deduped, r)
		}
	}

	return deduped
}

// classifyEvidence classifies search results as supporting, contradicting, or neutral.
// This uses keyword-based classification; in production, use NLP or an LLM.
func classifyEvidence(results []SearchResult, claim string) (supporting, contradicting, neutral []SearchResult) {
	claimLower := strings.ToLower(claim)

	for _, result := range results {
		titleLower := strings.ToLower(result.Title)
		snippetLower := strings.ToLower(result.Snippet)
		domainLower := strings.ToLower(result.Domain)

		// Calculate support and contradiction scores
		supportScore := 0
		contradictScore := 0

		// Check title and snippet for indicators
		supportIndicators := []string{
			"confirms", "supports", "proves", "evidence shows",
			"research shows", "study finds", "according to",
			"official", "verified", "true", "accurate",
		}
		contradictIndicators := []string{
			"false", "fake", "debunked", "myth", "conspiracy",
			"no evidence", "unsupported", "untrue", "misleading",
			"fact check", "incorrect", "wrong", "disputed",
		}

		for _, indicator := range supportIndicators {
			if strings.Contains(titleLower, indicator) || strings.Contains(snippetLower, indicator) {
				supportScore++
			}
		}

		for _, indicator := range contradictIndicators {
			if strings.Contains(titleLower, indicator) || strings.Contains(snippetLower, indicator) {
				contradictScore++
			}
		}

		// Fact-checking domains typically debunk claims
		if strings.Contains(domainLower, "fact") || strings.Contains(domainLower, "snopes") ||
			strings.Contains(domainLower, "politifact") || strings.Contains(domainLower, "fullfact") {
			contradictScore += 2
		}

		// Official sources are more likely to support official claims
		if result.IsOfficial && strings.Contains(claimLower, "official") {
			supportScore += 2
		}

		// Classify based on scores
		if supportScore > contradictScore {
			supporting = append(supporting, result)
		} else if contradictScore > supportScore {
			contradicting = append(contradicting, result)
		} else {
			neutral = append(neutral, result)
		}
	}

	return supporting, contradicting, neutral
}

// countIndependentSources counts the number of independent sources.
func countIndependentSources(results []SearchResult) int {
	count := 0
	for _, r := range results {
		if !r.IsOfficial {
			count++
		}
	}
	return count
}
