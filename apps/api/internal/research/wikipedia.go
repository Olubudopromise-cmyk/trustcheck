package research

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WikipediaProvider implements SearchProvider using the Wikipedia API.
// It searches Wikipedia for factual claims and returns results as evidence.
type WikipediaProvider struct {
	client *http.Client
}

// NewWikipediaProvider creates a new Wikipedia search provider. The HTTP
// client timeout is kept well below the serverless function deadline so a
// slow or blocked Wikipedia API can never push the whole verification past
// the platform limit; the request context provides the final bound.
func NewWikipediaProvider() *WikipediaProvider {
	return &WikipediaProvider{
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// Name returns the provider name.
func (p *WikipediaProvider) Name() string {
	return "wikipedia"
}

// SupportsMode indicates if this provider supports the given analysis mode.
func (p *WikipediaProvider) SupportsMode(mode string) bool {
	return true // Wikipedia works for all modes
}

// wikiSearchResponse represents the Wikipedia API search response.
type wikiSearchResponse struct {
	Query struct {
		Search []struct {
			Title   string `json:"title"`
			PageID  int    `json:"pageid"`
			Snippet string `json:"snippet"`
		} `json:"search"`
	} `json:"query"`
}

// wikiPageResponse represents the Wikipedia API page response.
type wikiPageResponse struct {
	Parse struct {
		Title   string `json:"title"`
		PageID  int    `json:"pageid"`
		Text    struct {
			Text string `json:"*"`
		} `json:"text"`
	} `json:"parse"`
}

// Search performs a web search using the Wikipedia API.
func (p *WikipediaProvider) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 10
	}

	// Step 1: Search for relevant pages with one retry. The backoff between
	// attempts is context-aware so an expired request deadline aborts the
	// retry immediately instead of sleeping past it.
	var searchResults []struct {
		Title   string
		PageID  int
		Snippet string
	}
	var err error

	for attempt := 0; attempt < 2; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		searchResults, err = p.searchPages(ctx, query, maxResults)
		if err == nil {
			break
		}
		if attempt < 1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("wikipedia search failed: %w", err)
	}

	// Step 2: Get page summaries for each result
	var results []SearchResult
	for _, sr := range searchResults {
		if len(results) >= maxResults {
			break
		}

		summary, err := p.getPageSummary(ctx, sr.Title)
		if err != nil {
			// If we can't get the summary, use the search snippet
			summary = sr.Snippet
		}

		// Clean up HTML from summary, truncating on rune boundaries so
		// multi-byte characters are never split (UTF-8 safety).
		summary = stripHTML(summary)
		runes := []rune(summary)
		if len(runes) > 500 {
			summary = string(runes[:500]) + "..."
		}

		domain := "wikipedia.org"
		result := SearchResult{
			Title:      sr.Title,
			URL:        fmt.Sprintf("https://en.wikipedia.org/wiki/%s", url.PathEscape(sr.Title)),
			Snippet:    summary,
			Domain:     domain,
			SourceType: SourceTypeAcademic, // Wikipedia is educational
			IsOfficial: false,
			IsAcademic: true,
			IsNews:     false,
			Confidence: 70, // Wikipedia has moderate confidence
		}

		results = append(results, result)
	}

	return results, nil
}

// searchPages searches Wikipedia for pages matching the query.
func (p *WikipediaProvider) searchPages(ctx context.Context, query string, limit int) ([]struct {
	Title   string
	PageID  int
	Snippet string
}, error) {
	// Use Wikipedia's opensearch API
	apiURL := fmt.Sprintf(
		"https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&srlimit=%d&format=json",
		url.QueryEscape(query),
		limit,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Wikipedia requires a descriptive User-Agent with contact info
	req.Header.Set("User-Agent", "TrustCheck/1.0 (https://trustcheck.netlify.app; fact-checking tool for research)")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == 429 {
		resp.Body.Close()
		return nil, fmt.Errorf("wikipedia rate limited")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var wikiResp wikiSearchResponse
	if err := json.Unmarshal(body, &wikiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var results []struct {
		Title   string
		PageID  int
		Snippet string
	}
	for _, r := range wikiResp.Query.Search {
		results = append(results, struct {
			Title   string
			PageID  int
			Snippet string
		}{
			Title:   r.Title,
			PageID:  r.PageID,
			Snippet: r.Snippet,
		})
	}

	return results, nil
}

// getPageSummary gets the summary/extract of a Wikipedia page.
func (p *WikipediaProvider) getPageSummary(ctx context.Context, title string) (string, error) {
	// Use the parse API to get the page content
	apiURL := fmt.Sprintf(
		"https://en.wikipedia.org/w/api.php?action=parse&page=%s&prop=text&format=json",
		url.PathEscape(title),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "TrustCheck/1.0 (fact-checking tool)")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var wikiResp wikiPageResponse
	if err := json.Unmarshal(body, &wikiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract the first paragraph from the HTML
	text := wikiResp.Parse.Text.Text
	return extractFirstParagraph(text), nil
}

// extractFirstParagraph extracts the first meaningful paragraph from HTML.
func extractFirstParagraph(html string) string {
	// Find the first <p> tag
	startIdx := strings.Index(html, "<p>")
	if startIdx == -1 {
		return ""
	}

	endIdx := strings.Index(html[startIdx:], "</p>")
	if endIdx == -1 {
		return ""
	}

	paragraph := html[startIdx+3 : startIdx+endIdx]
	return stripHTML(paragraph)
}
