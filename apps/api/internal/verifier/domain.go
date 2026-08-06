package verifier

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"time"
)

// probeTimeout is the per-attempt timeout for the HTTPS / HTTP checks.
const probeTimeout = 5 * time.Second

// statusBand scores the HTTP response status code per the spec:
// 200-399 -> +20, 400-499 -> +10, 500+ (and anything else) -> +0.
func statusBand(code int) int {
	switch {
	case code >= 200 && code <= 399:
		return 20
	case code >= 400 && code <= 499:
		return 10
	default:
		return 0
	}
}

// headOrGet issues a HEAD request; if the server rejects HEAD (405/501) it
// retries with GET, so verification survives servers that do not implement
// HEAD. The returned response body must be closed by the caller.
func headOrGet(client *http.Client, target string) (*http.Response, error) {
	resp, err := client.Head(target)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusMethodNotAllowed ||
		resp.StatusCode == http.StatusNotImplemented {
		resp.Body.Close()
		return client.Get(target)
	}
	return resp, nil
}

// isCertExpired reports whether a certificate is (or will be) expired.
func isCertExpired(c *x509.Certificate) bool {
	if c == nil || c.NotAfter.IsZero() {
		return false
	}
	return c.NotAfter.Before(time.Now())
}

// VerifyDomain runs the domain verification engine for a single domain.
//
// Scoring (Go standard library only):
//  1. DNS lookup with net.LookupHost; failure => unreachable / 15.
//  2. HEAD https://<domain> (5s); reachable => +20 (HTTPS available).
//  3. HEAD http://<domain> only if HTTPS failed; reachable => +10 (HTTP fallback).
//  4. Inspect resp.TLS; certificate present => +20, expired => -30.
//  5. Response status band: 200-399 => +20, 400-499 => +10, 500+ => +0.
//  6. Final score is clamped to [0, 100].
func VerifyDomain(domain string) Result {
	const (
		statusUnreachable = "unreachable"
		statusVerified    = "verified"
		statusWarning     = "warning"
	)

	// 1. DNS lookup.
	if ips, err := net.LookupHost(domain); err != nil || len(ips) == 0 {
		return Result{
			Status:     statusUnreachable,
			TrustScore: 15,
			Summary:    "Domain does not resolve.",
		}
	}

	httpsClient := &http.Client{
		Timeout:   probeTimeout,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	httpClient := &http.Client{Timeout: probeTimeout}

	var (
		httpsOK     bool
		httpOK      bool
		statusCode  int
		tlsState    *tls.ConnectionState
		certValid   bool
		certExpired bool
	)

	// 2. HTTPS check (TLS is inspected from the same response).
	resp, err := headOrGet(httpsClient, "https://"+domain)
	if err == nil {
		httpsOK = true
		statusCode = resp.StatusCode
		tlsState = resp.TLS
		if tlsState != nil && len(tlsState.PeerCertificates) > 0 {
				if isCertExpired(tlsState.PeerCertificates[0]) {
				certExpired = true
			} else {
				certValid = true
			}
		}
		resp.Body.Close()
	}

	// 3. HTTP fallback only when HTTPS failed.
	if !httpsOK {
		if resp, err = headOrGet(httpClient, "http://"+domain); err == nil {
			httpOK = true
			statusCode = resp.StatusCode
			resp.Body.Close()
		}
	}

	// 2-5. Accumulate the score.
	score := 0
	if httpsOK {
		score += 20 // HTTPS available
		if tlsState != nil && len(tlsState.PeerCertificates) > 0 {
			score += 20 // certificate present
			if certExpired {
				score -= 30
			}
		}
		score += statusBand(statusCode)
	} else if httpOK {
		score += 10 // HTTP fallback only
		score += statusBand(statusCode)
	}

	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	// Status + summary follow the spec's interpretation.
	var status, summary string
	switch {
	case httpsOK && certValid && statusBand(statusCode) == 20:
		status = statusVerified
		summary = "Domain resolves, HTTPS available, certificate valid."
	case !httpsOK && httpOK:
		status = statusWarning
		summary = "HTTPS unavailable."
	case httpsOK && certExpired:
		status = statusWarning
		summary = "TLS certificate expired."
	default:
		status = statusWarning
		summary = "Domain resolves, but verification is inconclusive."
	}

	return Result{Status: status, TrustScore: score, Summary: summary}
}
