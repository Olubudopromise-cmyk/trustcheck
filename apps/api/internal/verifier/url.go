package verifier

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"
)

const (
	urlStatusInvalid     = "invalid"
	urlStatusUnreachable = "unreachable"
	urlStatusVerified    = "verified"
	urlStatusWarning     = "warning"
)

// maxRedirectsPenalty caps the redirect penalty so that a pathological chain
// of hops cannot zero out an otherwise healthy site.
const maxRedirectsPenalty = 20

// urlVerifier implements the Verifier interface for URL inputs.
type urlVerifier struct{}

// Verify runs the URL verification engine for a single URL.
//
// Steps (Go standard library only):
//  1. Parse with net/url; malformed => invalid / 0 / "Invalid URL.".
//  2. Scheme must be http or https; anything else => invalid / 0.
//  3. Host existence is delegated to VerifyDomain (which already does DNS +
//     reachability checks) instead of duplicating that logic here.
//  4. HEAD request with GET fallback (shared headOrGet) and a 5s timeout.
//  5. For HTTPS the TLS certificate is inspected with the shared
//     isCertExpired helper (presence + expiration).
//  6. Redirects are followed and counted via http.Client.CheckRedirect; only
//     the count influences the score (chains are not exposed).
//  7. The response status is scored with the shared statusBand helper.
//
// Scoring: valid URL +10, host resolves +10, request OK +20, status band
// (2xx/3xx +20, 4xx +10, 5xx +0), HTTPS +15, valid certificate +10, expired
// certificate -30, redirect -10 each (capped at 20). The final score is
// clamped to [0, 100].
func (urlVerifier) Verify(input string) Result {
	// 1. Parse.
	u, err := url.Parse(strings.TrimSpace(input))
	if err != nil || u.Scheme == "" || u.Host == "" || u.Hostname() == "" {
		return Result{Status: urlStatusInvalid, TrustScore: 0, Summary: "Invalid URL."}
	}

	// 2. Scheme.
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return Result{Status: urlStatusInvalid, TrustScore: 0, Summary: "Invalid URL."}
	}

	// 3. Host existence, reusing the domain engine.
	if VerifyDomain(u.Hostname()).Status == urlStatusUnreachable {
		return Result{
			Status:     urlStatusUnreachable,
			TrustScore: 15,
			Summary:    "URL host does not resolve.",
		}
	}

	// 4-6. Request with redirect tracking. The redirect chain is only recorded
	// for scoring (count + final destination); it is never part of the Result.
	var redirects int
	client := &http.Client{
		Timeout: probeTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirects++
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	resp, err := headOrGet(client, u.String())
	if err != nil {
		// Host resolved but no HTTP response (refused / timeout / reset).
		return Result{
			Status:     urlStatusWarning,
			TrustScore: 25,
			Summary:    "URL host could not be reached.",
		}
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	// resp.Request.URL is the final destination after any redirects. A chain
	// that lands outside HTTPS is a trust signal: a downgrade from the
	// requested https URL is penalized below, an upgrade (http -> https) is not.
	finalURL := resp.Request.URL

	// 5. TLS inspection.
	var certValid, certExpired bool
	if scheme == "https" && resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		if isCertExpired(resp.TLS.PeerCertificates[0]) {
			certExpired = true
		} else {
			certValid = true
		}
	}

	// 7. Score.
	score := 40 // valid URL +10, host resolves +10, request OK +20
	score += statusBand(statusCode)
	if scheme == "https" {
		score += 15
		switch {
		case certValid:
			score += 10
		case certExpired:
			score -= 30
		}
		if finalURL != nil && strings.ToLower(finalURL.Scheme) != "https" {
			score -= 10 // redirected off HTTPS (TLS downgrade)
		}
	}
	penalty := redirects * 10
	if penalty > maxRedirectsPenalty {
		penalty = maxRedirectsPenalty
	}
	score -= penalty
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	status, summary := summarizeURL(scheme, certValid, certExpired, statusCode)
	return Result{Status: status, TrustScore: score, Summary: summary}
}

// summarizeURL maps the observed verification results to a status + summary.
func summarizeURL(scheme string, certValid, certExpired bool, statusCode int) (status, summary string) {
	switch {
	case scheme == "https" && certValid && statusBand(statusCode) == 20:
		return urlStatusVerified, "HTTPS available with a valid certificate."
	case scheme == "https" && certExpired:
		return urlStatusWarning, "TLS certificate expired."
	case statusBand(statusCode) == 10:
		return urlStatusWarning, "Site responded with a client error."
	case statusBand(statusCode) == 0:
		return urlStatusWarning, "Site responded with a server error."
	case scheme == "http":
		return urlStatusWarning, "Site is reachable over HTTP only."
	default:
		return urlStatusWarning, "Site is reachable but verification is inconclusive."
	}
}
