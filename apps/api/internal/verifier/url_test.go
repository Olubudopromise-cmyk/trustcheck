package verifier

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestURLVerifier_Invalid covers malformed URLs deterministically: every input
// must be rejected with the invalid result.
func TestURLVerifier_Invalid(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"no scheme", "example.com"},
		{"no host", "https://"},
		{"empty host with fragment", "https://#fragment"},
		{"garbage scheme delimiter", "ht!tp://example.com"},
		{"bare path", "/some/path"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := urlVerifier{}.Verify(tc.in)
			if v.Status != "invalid" || v.TrustScore != 0 || v.Summary != "Invalid URL." {
				t.Errorf("Verify(%q) = %+v, want invalid/0/Invalid URL.", tc.in, v)
			}
			assertEvidenceLabels(t, v, []string{"Valid URL"})
		})
	}
}

// TestURLVerifier_UnsupportedScheme ensures only http and https are accepted.
func TestURLVerifier_UnsupportedScheme(t *testing.T) {
	for _, in := range []string{
		"ftp://example.com",
		"file:///etc/passwd",
		"mailto:user@example.com",
		"javascript:alert(1)",
		"ws://example.com",
	} {
		v := urlVerifier{}.Verify(in)
		if v.Status != "invalid" || v.TrustScore != 0 || v.Summary != "Invalid URL." {
			t.Errorf("Verify(%q) = %+v, want invalid/0/Invalid URL.", in, v)
		}
		assertEvidenceLabels(t, v, []string{"Valid URL"})
	}
}

// TestURLVerifier_ServerCases exercises the reachable-URL behaviour against
// local httptest servers, so it is deterministic and needs no outbound
// network. A TLS server covers the HTTPS scoring path, a plain server the
// HTTP-only path.
func TestURLVerifier_ServerCases(t *testing.T) {
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer plain.Close()

	tlsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/", http.StatusFound)
		case "/client-error":
			w.WriteHeader(http.StatusNotFound)
		case "/server-error":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer tlsSrv.Close()

	cases := []struct {
		name       string
		in         string
		wantStatus string
		wantScore  int
		wantSum    string
		wantLabels []string
	}{
		{"https valid", tlsSrv.URL, "verified", 85, "HTTPS available with a valid certificate.",
			[]string{"Valid URL", "HTTP Status OK", "HTTPS Available", "Valid TLS Certificate"}},
		{"https redirect", tlsSrv.URL + "/redirect", "verified", 75, "HTTPS available with a valid certificate.",
			[]string{"Valid URL", "HTTP Status OK", "HTTPS Available", "Valid TLS Certificate", "Redirect Detected"}},
		{"https client error", tlsSrv.URL + "/client-error", "warning", 75, "Site responded with a client error.",
			[]string{"Valid URL", "HTTP Client Error", "HTTPS Available", "Valid TLS Certificate"}},
		{"https server error", tlsSrv.URL + "/server-error", "warning", 65, "Site responded with a server error.",
			[]string{"Valid URL", "HTTP Server Error", "HTTPS Available", "Valid TLS Certificate"}},
		{"http only", plain.URL, "warning", 60, "Site is reachable over HTTP only.",
			[]string{"Valid URL", "HTTP Status OK"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := urlVerifier{}.Verify(tc.in)
			if v.Status != tc.wantStatus || v.TrustScore != tc.wantScore || v.Summary != tc.wantSum {
				t.Errorf("Verify(%q) = %+v, want %s/%d/%q",
					tc.in, v, tc.wantStatus, tc.wantScore, tc.wantSum)
			}
			assertEvidenceLabels(t, v, tc.wantLabels)
		})
	}
}

// TestURLVerifier_UnreachableHost uses the RFC 2606 ".invalid" TLD, which
// never resolves, so the engine must deterministically report unreachable.
func TestURLVerifier_UnreachableHost(t *testing.T) {
	v := urlVerifier{}.Verify("https://this-host-does-not-exist-xyz123.invalid")
	if v.Status != "unreachable" || v.TrustScore != 15 ||
		v.Summary != "URL host does not resolve." {
		t.Errorf("got %+v, want unreachable/15/URL host does not resolve.", v)
	}
	assertEvidenceLabels(t, v, []string{"DNS Lookup"})
}

// TestURLVerifier_Localhost: localhost resolves (loopback) but serves no web
// stack here, so it must never be reported as invalid. The exact warning
// result depends on whether anything listens on port 80, so only shape is
// asserted, mirroring TestVerifyDomain_Localhost.
func TestURLVerifier_Localhost(t *testing.T) {
	v := urlVerifier{}.Verify("http://localhost")
	if v.Status == "invalid" {
		t.Fatal("localhost must not be reported invalid")
	}
	if v.TrustScore < 0 || v.TrustScore > 100 {
		t.Errorf("trustScore %d out of [0,100]", v.TrustScore)
	}
	if len(v.Evidence) == 0 {
		t.Errorf("localhost result carries no evidence: %+v", v)
	}
}

// TestURLVerifier_RealSite mirrors the repo's network-tolerant style: online
// https://example.com verifies at 85, offline it degrades to unreachable or a
// reachable-but-unreachable warning. Only well-formedness is asserted in the
// degraded cases.
func TestURLVerifier_RealSite(t *testing.T) {
	v := urlVerifier{}.Verify("https://example.com")
	t.Logf("https://example.com -> %+v", v)

	switch v.Status {
	case "unreachable":
		if v.TrustScore != 15 {
			t.Errorf("offline expected score 15, got %d", v.TrustScore)
		}
	case "verified":
		if v.TrustScore != 85 || v.Summary != "HTTPS available with a valid certificate." {
			t.Errorf("got %+v, want verified/85/HTTPS available with a valid certificate.", v)
		}
	case "warning":
		if v.TrustScore < 0 || v.TrustScore > 100 {
			t.Errorf("trustScore %d out of [0,100]", v.TrustScore)
		}
	default:
		t.Errorf("unexpected status %q (%+v)", v.Status, v)
	}
	if len(v.Evidence) == 0 {
		t.Errorf("result carries no evidence: %+v", v)
	}
}
