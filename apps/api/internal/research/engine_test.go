package research

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// MockSearchProvider is a mock implementation of SearchProvider for testing.
type MockSearchProvider struct {
	Results    []SearchResult
	Error      error
	SearchCount int
	LastQuery  string
}

func (m *MockSearchProvider) Name() string {
	return "mock"
}

func (m *MockSearchProvider) SupportsMode(mode string) bool {
	return true
}

func (m *MockSearchProvider) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	m.SearchCount++
	m.LastQuery = query
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Results, nil
}

func TestResearchEngine_OrdinaryClaim(t *testing.T) {
	mock := &MockSearchProvider{
		Results: []SearchResult{
			{
				Title:      "NASA confirms water on Mars",
				URL:        "https://nasa.gov/mars-water",
				Snippet:    "NASA has confirmed the presence of water on Mars.",
				Domain:     "nasa.gov",
				SourceType: SourceTypeOfficial,
				IsOfficial: true,
				SupportsClaim: true,
				Confidence: 90,
			},
			{
				Title:      "Mars Water Discovery",
				URL:        "https://science.org/mars-water",
				Snippet:    "Scientists confirm water ice on Mars.",
				Domain:     "science.org",
				SourceType: SourceTypeAcademic,
				IsAcademic: true,
				SupportsClaim: true,
				Confidence: 85,
			},
		},
	}

	engine := NewResearchEngine(mock, DefaultSearchConfig())
	evidence := engine.Research(context.Background(), "NASA confirms water on Mars", []string{"nasa", "mars", "water"})

	if evidence.SearchStatus != SearchStatusComplete {
		t.Errorf("expected search status complete, got %s", evidence.SearchStatus)
	}

	if len(evidence.Supporting) == 0 {
		t.Error("expected supporting evidence")
	}

	if len(evidence.Contradicting) > 0 {
		t.Error("expected no contradicting evidence for this claim")
	}

	if evidence.TotalSources != 2 {
		t.Errorf("expected 2 total sources, got %d", evidence.TotalSources)
	}

	if mock.SearchCount == 0 {
		t.Error("expected at least one search to be performed")
	}
}

func TestResearchEngine_ControversialClaim(t *testing.T) {
	mock := &MockSearchProvider{
		Results: []SearchResult{
			{
				Title:      "Fact Check: Microchips in Vaccines",
				URL:        "https://snopes.com/fact-check/microchips",
				Snippet:    "This claim is false. No evidence supports microchips in vaccines.",
				Domain:     "snopes.com",
				SourceType: SourceTypeFactChecker,
				SupportsClaim: false,
				ContradictsClaim: true,
				Confidence: 80,
			},
			{
				Title:      "COVID-19 Vaccine Safety",
				URL:        "https://cdc.gov/vaccine-safety",
				Snippet:    "COVID-19 vaccines are safe and effective.",
				Domain:     "cdc.gov",
				SourceType: SourceTypeOfficial,
				IsOfficial: true,
				SupportsClaim: false,
				ContradictsClaim: true,
				Confidence: 95,
			},
		},
	}

	engine := NewResearchEngine(mock, DefaultSearchConfig())
	evidence := engine.Research(context.Background(), "COVID-19 vaccines contain microchips", []string{"covid", "vaccines", "microchips"})

	if len(evidence.Contradicting) == 0 {
		t.Error("expected contradicting evidence for this controversial claim")
	}

	if len(evidence.Supporting) > 0 {
		t.Error("expected no supporting evidence for this false claim")
	}
}

func TestResearchEngine_GovernmentMode(t *testing.T) {
	mock := &MockSearchProvider{
		Results: []SearchResult{
			{
				Title:      "EPA Climate Report",
				URL:        "https://epa.gov/climate-report",
				Snippet:    "Official EPA report on climate change.",
				Domain:     "epa.gov",
				SourceType: SourceTypeOfficial,
				IsOfficial: true,
				SupportsClaim: true,
				Confidence: 95,
			},
		},
	}

	config := ModeConfig("government_official")
	engine := NewResearchEngine(mock, config)
	evidence := engine.Research(context.Background(), "Climate change is real", []string{"climate", "change"})

	if evidence.SearchStatus != SearchStatusComplete {
		t.Errorf("expected search status complete, got %s", evidence.SearchStatus)
	}

	// Check that government mode was used (more queries)
	if mock.SearchCount < 2 {
		t.Errorf("expected at least 2 searches in government mode, got %d", mock.SearchCount)
	}
}

func TestResearchEngine_DeepMode(t *testing.T) {
	mock := &MockSearchProvider{
		Results: []SearchResult{
			{
				Title:      "Research Study Confirms",
				URL:        "https://nature.com/study",
				Snippet:    "Peer-reviewed study confirms the relationship.",
				Domain:     "nature.com",
				SourceType: SourceTypeAcademic,
				IsAcademic: true,
				SupportsClaim: true,
				Confidence: 90,
			},
		},
	}

	config := ModeConfig("deep_research")
	engine := NewResearchEngine(mock, config)
	evidence := engine.Research(context.Background(), "Vaccines cause autism", []string{"vaccines", "autism"})

	// Deep mode should perform more queries
	if mock.SearchCount < 3 {
		t.Errorf("expected at least 3 searches in deep mode, got %d", mock.SearchCount)
	}

	// Should have some evidence (supporting, contradicting, or neutral)
	totalEvidence := len(evidence.Supporting) + len(evidence.Contradicting) + len(evidence.Neutral)
	if totalEvidence == 0 {
		t.Error("expected some evidence in deep mode")
	}
}

func TestResearchEngine_SearchFailure(t *testing.T) {
	mock := &MockSearchProvider{
		Error: fmt.Errorf("search engine unavailable"),
	}

	engine := NewResearchEngine(mock, DefaultSearchConfig())
	evidence := engine.Research(context.Background(), "Test claim", []string{"test"})

	// When all searches fail, status should be failed or partial
	if evidence.SearchStatus != SearchStatusFailed && evidence.SearchStatus != SearchStatusPartial {
		t.Errorf("expected search status failed or partial, got %s", evidence.SearchStatus)
	}

	if len(evidence.SearchErrors) == 0 {
		t.Error("expected search errors")
	}
}

func TestResearchEngine_NoResults(t *testing.T) {
	mock := &MockSearchProvider{
		Results: []SearchResult{},
	}

	engine := NewResearchEngine(mock, DefaultSearchConfig())
	evidence := engine.Research(context.Background(), "Nonexistent topic xyz123", []string{"nonexistent"})

	if evidence.SearchStatus != SearchStatusNoResults {
		t.Errorf("expected search status no_results, got %s", evidence.SearchStatus)
	}

	if len(evidence.Supporting) > 0 || len(evidence.Contradicting) > 0 {
		t.Error("expected no evidence when no results found")
	}
}

func TestResearchEngine_Deduplication(t *testing.T) {
	mock := &MockSearchProvider{
		Results: []SearchResult{
			{
				Title:      "Same Article",
				URL:        "https://example.com/article1",
				Snippet:    "Test snippet",
				Domain:     "example.com",
				SupportsClaim: true,
			},
			{
				Title:      "Same Article Again",
				URL:        "https://example.com/article1", // Duplicate URL
				Snippet:    "Different snippet",
				Domain:     "example.com",
				SupportsClaim: true,
			},
			{
				Title:      "Different Article",
				URL:        "https://example.com/article2",
				Snippet:    "Another snippet",
				Domain:     "example.com",
				SupportsClaim: true,
			},
		},
	}

	engine := NewResearchEngine(mock, DefaultSearchConfig())
	evidence := engine.Research(context.Background(), "Test claim", []string{"test"})

	if evidence.TotalSources != 2 {
		t.Errorf("expected 2 unique sources after deduplication, got %d", evidence.TotalSources)
	}
}

func TestResearchEngine_EvidenceClassification(t *testing.T) {
	mock := &MockSearchProvider{
		Results: []SearchResult{
			{
				Title:      "Study Confirms Link",
				URL:        "https://example.com/confirm",
				Snippet:    "Research confirms the relationship",
				Domain:     "example.com",
				SupportsClaim: true,
			},
			{
				Title:      "Fact Check: Claim is False",
				URL:        "https://snopes.com/false",
				Snippet:    "This claim has been debunked",
				Domain:     "snopes.com",
				ContradictsClaim: true,
			},
			{
				Title:      "General Article",
				URL:        "https://news.com/general",
				Snippet:    "General information about the topic",
				Domain:     "news.com",
			},
		},
	}

	engine := NewResearchEngine(mock, DefaultSearchConfig())
	evidence := engine.Research(context.Background(), "Test claim", []string{"test"})

	if len(evidence.Supporting) != 1 {
		t.Errorf("expected 1 supporting evidence, got %d", len(evidence.Supporting))
	}

	if len(evidence.Contradicting) != 1 {
		t.Errorf("expected 1 contradicting evidence, got %d", len(evidence.Contradicting))
	}

	if len(evidence.Neutral) != 1 {
		t.Errorf("expected 1 neutral evidence, got %d", len(evidence.Neutral))
	}
}

// blockingProvider models a provider whose HTTP call hangs until the context
// fires, as happens when DuckDuckGo/Wikipedia are slow or blocked. The engine
// must return promptly (marked partial/failed) rather than serializing
// per-provider timeouts and blowing the serverless function deadline.
type hangingProvider struct{}

func (hangingProvider) Name() string            { return "hanging" }
func (hangingProvider) SupportsMode(mode string) bool { return true }
func (hangingProvider) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestResearchEngine_HangingProviderRespectsDeadline(t *testing.T) {
	engine := NewResearchEngine(hangingProvider{}, DefaultSearchConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	evidence := engine.Research(ctx, "Some claim", []string{"claim"})
	elapsed := time.Since(start)

	if evidence.SearchStatus != SearchStatusPartial && evidence.SearchStatus != SearchStatusFailed {
		t.Errorf("expected partial or failed status, got %s", evidence.SearchStatus)
	}
	if len(evidence.SearchErrors) == 0 {
		t.Error("expected recorded search errors")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("engine took %s; the provider deadline was not honored", elapsed)
	}
}

func TestResearchEngine_ContextCancellation(t *testing.T) {
	mock := &MockSearchProvider{
		Results: []SearchResult{
			{
				Title:  "Test",
				URL:    "https://example.com",
				Domain: "example.com",
			},
		},
	}

	engine := NewResearchEngine(mock, DefaultSearchConfig())
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	evidence := engine.Research(ctx, "Test claim", []string{"test"})

	// Should handle cancellation gracefully
	if evidence.SearchStatus != SearchStatusPartial && evidence.SearchStatus != SearchStatusFailed {
		t.Errorf("expected partial or failed status on cancellation, got %s", evidence.SearchStatus)
	}
}

func TestModeConfig(t *testing.T) {
	quickConfig := ModeConfig("quick")
	if quickConfig.MaxQueries != 3 {
		t.Errorf("expected quick mode to have 3 max queries, got %d", quickConfig.MaxQueries)
	}

	deepConfig := ModeConfig("deep_research")
	if deepConfig.MaxQueries != 5 {
		t.Errorf("expected deep mode to have 5 max queries, got %d", deepConfig.MaxQueries)
	}
	if !deepConfig.SearchContradictions {
		t.Error("expected deep mode to search for contradictions")
	}

	govConfig := ModeConfig("government_official")
	if !govConfig.RequireOfficialSources {
		t.Error("expected government mode to require official sources")
	}
}
