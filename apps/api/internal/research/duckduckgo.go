package research

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// DuckDuckGoProvider implements SearchProvider using DuckDuckGo's HTML search.
// This is a free, no-API-key-required provider suitable for development and
// low-volume production use. For higher volume, consider using an official
// search API.
type DuckDuckGoProvider struct {
	client *http.Client
}

// NewDuckDuckGoProvider creates a new DuckDuckGo search provider.
func NewDuckDuckGoProvider() *DuckDuckGoProvider {
	return &DuckDuckGoProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the provider name.
func (p *DuckDuckGoProvider) Name() string {
	return "duckduckgo"
}

// SupportsMode indicates if this provider supports the given analysis mode.
// DuckDuckGo supports all modes but doesn't have special government search.
func (p *DuckDuckGoProvider) SupportsMode(mode string) bool {
	return true
}

// Search performs a web search using DuckDuckGo's HTML version.
func (p *DuckDuckGoProvider) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 10
	}

	// DuckDuckGo HTML version
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set a standard User-Agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 202 means rate limited / CAPTCHA
		if resp.StatusCode == 202 {
			return nil, fmt.Errorf("search rate limited (CAPTCHA)")
		}
		return nil, fmt.Errorf("search returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return parseDuckDuckGoResults(string(body), maxResults), nil
}

// parseDuckDuckGoResults extracts search results from DuckDuckGo HTML.
func parseDuckDuckGoResults(html string, maxResults int) []SearchResult {
	var results []SearchResult
	seen := make(map[string]bool)

	// Extract result blocks - DuckDuckGo uses div with class="result results_links results_links_deep web-result"
	resultBlockRegex := regexp.MustCompile(`(?s)<div[^>]*class="result results_links[^"]*">(.*?)</div>\s*</div>\s*</div>`)
	blocks := resultBlockRegex.FindAllString(html, -1)

	for _, block := range blocks {
		if len(results) >= maxResults {
			break
		}

		// Extract title and URL from <a class="result__a" href="...">
		titleRegex := regexp.MustCompile(`(?s)<a class="result__a"[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
		titleMatch := titleRegex.FindStringSubmatch(block)
		if titleMatch == nil {
			continue
		}
		resultURL := cleanDuckDuckGoURL(titleMatch[1])
		title := stripHTML(titleMatch[2])

		if resultURL == "" || title == "" {
			continue
		}

		// Skip duplicates
		if seen[resultURL] {
			continue
		}
		seen[resultURL] = true

		// Extract snippet
		snippetRegex := regexp.MustCompile(`(?s)<(?:a|div) class="result__snippet"[^>]*>(.*?)</(?:a|div)>`)
		snippetMatch := snippetRegex.FindStringSubmatch(block)
		snippet := ""
		if snippetMatch != nil {
			snippet = stripHTML(snippetMatch[1])
		}

		domain := extractDomain(resultURL)

		result := SearchResult{
			Title:      title,
			URL:        resultURL,
			Snippet:    snippet,
			Domain:     domain,
			SourceType: classifyDomainType(domain),
			IsOfficial: isOfficialDomain(domain),
			IsAcademic: isAcademicDomain(domain),
			IsNews:     isNewsDomain(domain),
			Confidence: calculateResultConfidence(domain),
		}

		results = append(results, result)
	}

	// Fallback: if no results found with the primary regex, try a simpler pattern
	if len(results) == 0 {
		linkRegex := regexp.MustCompile(`href="(//duckduckgo\.com/l/\?uddg=[^"]+)"`)
		links := linkRegex.FindAllStringSubmatch(html, -1)
		for _, link := range links {
			if len(results) >= maxResults {
				break
			}
			resultURL := cleanDuckDuckGoURL(link[1])
			if resultURL != "" && !seen[resultURL] {
				seen[resultURL] = true
				domain := extractDomain(resultURL)
				results = append(results, SearchResult{
					Title:      extractTitleFromDomain(domain),
					URL:        resultURL,
					Snippet:    "",
					Domain:     domain,
					SourceType: classifyDomainType(domain),
					IsOfficial: isOfficialDomain(domain),
					IsAcademic: isAcademicDomain(domain),
					IsNews:     isNewsDomain(domain),
					Confidence: calculateResultConfidence(domain),
				})
			}
		}
	}

	return results
}

// cleanDuckDuckGoURL extracts the actual URL from DuckDuckGo's redirect URL.
func cleanDuckDuckGoURL(rawURL string) string {
	if strings.Contains(rawURL, "uddg=") {
		parts := strings.Split(rawURL, "uddg=")
		if len(parts) > 1 {
			decoded, err := url.QueryUnescape(strings.Split(parts[1], "&")[0])
			if err == nil {
				return decoded
			}
		}
	}
	return rawURL
}

// extractDomain extracts the domain name from a URL.
func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := parsed.Hostname()
	host = strings.TrimPrefix(host, "www.")
	return host
}

// extractTitleFromDomain extracts a simple title from a domain.
func extractTitleFromDomain(domain string) string {
	return domain
}

// stripHTML removes HTML tags from a string.
func stripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	return s
}

// classifyDomainType determines the type of a domain.
func classifyDomainType(domain string) SourceType {
	lower := strings.ToLower(domain)

	// Official government domains
	if strings.HasSuffix(lower, ".gov") || strings.Contains(lower, ".gov.") {
		return SourceTypeOfficial
	}

	// Academic domains
	if strings.HasSuffix(lower, ".edu") || strings.Contains(lower, "arxiv.org") ||
		strings.Contains(lower, "pubmed") || strings.Contains(lower, "scholar") {
		return SourceTypeAcademic
	}

	// Fact-checking domains
	if strings.Contains(lower, "fact") || strings.Contains(lower, "snopes") ||
		strings.Contains(lower, "politifact") || strings.Contains(lower, "fullfact") {
		return SourceTypeFactChecker
	}

	// News domains
	newsDomains := []string{
		"reuters.com", "apnews.com", "bbc.com", "cnn.com",
		"nytimes.com", "washingtonpost.com", "theguardian.com",
		"aljazeera.com", "bloomberg.com", "wsj.com",
	}
	for _, news := range newsDomains {
		if strings.Contains(lower, news) {
			return SourceTypeJournalism
		}
	}

	return SourceTypeUnknown
}

// isOfficialDomain checks if a domain is from an official government source.
func isOfficialDomain(domain string) bool {
	lower := strings.ToLower(domain)
	return strings.HasSuffix(lower, ".gov") || strings.Contains(lower, ".gov.") ||
		strings.Contains(lower, "who.int") || strings.Contains(lower, "cdc.gov") ||
		strings.Contains(lower, "nih.gov") || strings.Contains(lower, "nasa.gov")
}

// isAcademicDomain checks if a domain is from an academic source.
func isAcademicDomain(domain string) bool {
	lower := strings.ToLower(domain)
	return strings.HasSuffix(lower, ".edu") || strings.Contains(lower, "arxiv.org") ||
		strings.Contains(lower, "pubmed") || strings.Contains(lower, "nature.com") ||
		strings.Contains(lower, "science.org") || strings.Contains(lower, "springer.com")
}

// isNewsDomain checks if a domain is from a news source.
func isNewsDomain(domain string) bool {
	lower := strings.ToLower(domain)
	newsDomains := []string{
		"reuters.com", "apnews.com", "bbc.com", "cnn.com",
		"nytimes.com", "washingtonpost.com", "theguardian.com",
	}
	for _, news := range newsDomains {
		if strings.Contains(lower, news) {
			return true
		}
	}
	return false
}

// calculateResultConfidence estimates confidence in a search result.
func calculateResultConfidence(domain string) int {
	base := 50
	if isOfficialDomain(domain) {
		base += 25
	}
	if isAcademicDomain(domain) {
		base += 20
	}
	if isNewsDomain(domain) {
		base += 10
	}
	if base > 100 {
		base = 100
	}
	return base
}
