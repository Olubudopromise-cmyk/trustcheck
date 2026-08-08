// Package research provides a clean evidence-retrieval layer for the analysis
// pipeline. It defines a SearchProvider interface that can be implemented by
// different search backends (DuckDuckGo, Google, Bing, fact-checking APIs, etc.)
// without changing the analysis pipeline.
//
// The architecture is:
//
//	TrustCheck → SearchProvider interface → provider → web results → evidence pipeline
//
// This separation makes it easy to swap providers without rewriting the pipeline.
package research

import (
	"context"
	"time"
)

// SearchResult represents a single search result from any provider.
// Every field is grounded in observable data; nothing is fabricated.
type SearchResult struct {
	// Title is the title of the search result.
	Title string

	// URL is the original source URL that the user can click to verify.
	URL string

	// Snippet is a brief excerpt from the source, if available.
	// Empty string means no snippet was extracted (not fabricated).
	Snippet string

	// Domain is the domain name extracted from the URL.
	Domain string

	// PublicationDate is the publication date if available.
	// Empty string means the date is unknown (not fabricated).
	PublicationDate string

	// SourceType indicates the type of source.
	SourceType SourceType

	// IsOfficial indicates if this is from an official government or institutional source.
	IsOfficial bool

	// IsAcademic indicates if this is from an academic or research source.
	IsAcademic bool

	// IsNews indicates if this is from a news organization.
	IsNews bool

	// SupportsClaim indicates if this result supports the claim.
	SupportsClaim bool

	// ContradictsClaim indicates if this result contradicts the claim.
	ContradictsClaim bool

	// Confidence is a 0-100 score indicating confidence in this result's accuracy.
	Confidence int
}

// SourceType indicates the type of a source.
type SourceType string

const (
	SourceTypeOfficial     SourceType = "official"     // Government agencies, regulatory bodies
	SourceTypeAcademic     SourceType = "academic"     // Universities, research institutions
	SourceTypeJournalism   SourceType = "journalism"   // News organizations
	SourceTypeCommunity    SourceType = "community"    // Forums, social media
	SourceTypeCommercial   SourceType = "commercial"   // Company websites, blogs
	SourceTypeFactChecker  SourceType = "fact_checker" // Fact-checking organizations
	SourceTypeUnknown      SourceType = "unknown"      // Unclassified sources
)

// SearchProvider is the interface that all search backends must implement.
// This allows us to swap providers without changing the analysis pipeline.
type SearchProvider interface {
	// Name returns the provider name for logging and debugging.
	Name() string

	// Search performs a web search and returns results.
	// The query is a natural-language claim or search query.
	// maxResults limits the number of results returned.
	// Returns an error if the search fails (timeout, block, etc.)
	Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error)

	// SupportsMode indicates if this provider supports the given analysis mode.
	// For example, some providers may not support government-specific searches.
	SupportsMode(mode string) bool
}

// SearchConfig configures how a search is performed.
type SearchConfig struct {
	// MaxResults is the maximum number of results per query.
	MaxResults int

	// MaxQueries is the maximum number of search queries to perform.
	MaxQueries int

	// Timeout is the maximum time allowed for a single search.
	Timeout time.Duration

	// Mode is the analysis mode (quick, deep, government, etc.)
	Mode string

	// RequireOfficialSources prioritizes official sources.
	RequireOfficialSources bool

	// RequireAcademicSources prioritizes academic sources.
	RequireAcademicSources bool

	// SearchContradictions actively searches for contradicting evidence.
	SearchContradictions bool
}

// DefaultSearchConfig returns sensible defaults for search configuration.
func DefaultSearchConfig() SearchConfig {
	return SearchConfig{
		MaxResults:            10,
		MaxQueries:            3,
		Timeout:               10 * time.Second,
		Mode:                  "quick",
		RequireOfficialSources: false,
		RequireAcademicSources: false,
		SearchContradictions:  false,
	}
}

// ModeConfig returns search configuration for a specific analysis mode.
func ModeConfig(mode string) SearchConfig {
	switch mode {
	case "deep_research":
		return SearchConfig{
			MaxResults:            15,
			MaxQueries:            5,
			Timeout:               15 * time.Second,
			Mode:                  "deep_research",
			RequireOfficialSources: false,
			RequireAcademicSources: true,
			SearchContradictions:  true,
		}
	case "government_official":
		return SearchConfig{
			MaxResults:            12,
			MaxQueries:            4,
			Timeout:               12 * time.Second,
			Mode:                  "government_official",
			RequireOfficialSources: true,
			RequireAcademicSources: true,
			SearchContradictions:  true,
		}
	default: // quick
		return DefaultSearchConfig()
	}
}
