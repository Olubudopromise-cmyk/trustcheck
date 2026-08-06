package verifier

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"time"

	"github.com/pamierin/trustcheck/apps/api/internal/scoring"
)

// probeTimeout is the per-attempt timeout for the HTTPS / HTTP checks.
const probeTimeout = 5 * time.Second

// statusBand scores the HTTP response status code per the spec:
// 200-399 -> +20, 400-499 -> +10, 500+ (and anything else) -> +0.
func statusBand(code int) int {
	switch {
	case code >= 200 && code <= 399:
		return scoring.StatusOKBonus
	case code >= 400 && code <= 499:
		return scoring.StatusClientBonus
	default:
		return scoring.StatusServerBonus
	}
}

// addStatusBand records the HTTP status code check as evidence, contributing
// the same points as statusBand.
func addStatusBand(b *scoring.Builder, code int) {
	switch {
	case code >= 200 && code <= 399:
		b.Pass("HTTP Status OK", scoring.StatusOKBonus)
	case code >= 400 && code <= 499:
		b.Warning("HTTP Client Error", scoring.StatusClientBonus)
	default:
		b.Info("HTTP Server Error")
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

	score := scoring.New()

	// 1. DNS lookup.
	if ips, err := net.LookupHost(domain); err != nil || len(ips) == 0 {
		score.Fail("DNS Lookup", 0)
		return Result{
			Status:     statusUnreachable,
			TrustScore: scoring.UnreachableScore,
			Summary:    "Domain does not resolve.",
			Evidence:   score.Evidence(),
		}
	}
	score.Pass("DNS Resolves", 0)

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

	// 2-5. Accumulate the score with the shared Builder (auto-clamps).
	if httpsOK {
		score.Pass("HTTPS Available", scoring.HTTPSBonus)
		if tlsState != nil && len(tlsState.PeerCertificates) > 0 {
			score.Pass("TLS Certificate Present", scoring.ValidCertBonus)
			if certExpired {
				score.Fail("TLS Certificate Expired", -scoring.ExpiredCertPenalty)
			}
		}
		addStatusBand(score, statusCode)
	} else if httpOK {
		score.Pass("HTTP Fallback", scoring.HTTPFallbackBonus)
		addStatusBand(score, statusCode)
	}

	// Status + summary follow the spec's interpretation.
	var status, summary string
	switch {
	case httpsOK && certValid && statusBand(statusCode) == scoring.StatusOKBonus:
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

	return Result{Status: status, TrustScore: score.Score(), Summary: summary, Evidence: score.Evidence()}
}
