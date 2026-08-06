package verifier

import "testing"

// assertResult verifies status, score and summary equality while requiring the
// result to carry at least one evidence entry (Sprint 14 contract: every
// verifier result is backed by a score breakdown).
func assertResult(t *testing.T, got, want Result) {
	t.Helper()
	if got.Status != want.Status || got.TrustScore != want.TrustScore || got.Summary != want.Summary {
		t.Errorf("result = %+v, want status=%q score=%d summary=%q",
			got, want.Status, want.TrustScore, want.Summary)
	}
	if len(got.Evidence) == 0 {
		t.Errorf("result carries no evidence: %+v", got)
	}
}

// assertEvidenceLabels checks the evidence labels (and their order) for
// deterministic engines.
func assertEvidenceLabels(t *testing.T, got Result, labels []string) {
	t.Helper()
	if len(got.Evidence) != len(labels) {
		t.Fatalf("evidence length = %d, want %d (%+v)", len(got.Evidence), len(labels), got.Evidence)
	}
	for i, l := range labels {
		if got.Evidence[i].Label != l {
			t.Errorf("evidence[%d].label = %q, want %q", i, got.Evidence[i].Label, l)
		}
	}
}

func TestVerifyDomain_NotResolvable(t *testing.T) {
	// ".invalid" is reserved by RFC 2606 and never resolves: deterministic.
	r := VerifyDomain("this-domain-does-not-exist-xyz123.invalid")
	if r.Status != "unreachable" {
		t.Errorf("status = %q, want %q", r.Status, "unreachable")
	}
	if r.TrustScore != 15 {
		t.Errorf("trustScore = %d, want 15", r.TrustScore)
	}
	if r.Summary != "Domain does not resolve." {
		t.Errorf("summary = %q, want %q", r.Summary, "Domain does not resolve.")
	}
	assertEvidenceLabels(t, r, []string{"DNS Lookup"})
}

func TestVerifyDomain_Malformed(t *testing.T) {
	for _, in := range []string{"....", "not a domain", "-leading", " ", ""} {
		r := VerifyDomain(in)
		if r.Status != "unreachable" || r.TrustScore != 15 {
			t.Errorf("VerifyDomain(%q) = %+v, want unreachable/15", in, r)
		}
		assertEvidenceLabels(t, r, []string{"DNS Lookup"})
	}
}

func TestVerifyDomain_Valid(t *testing.T) {
	// example.com is reachable from CI with outbound network. When offline,
	// DNS fails and the engine degrades to unreachable/15. Either outcome is
	// exercised; only assert well-formedness in the offline case.
	r := VerifyDomain("example.com")
	switch r.Status {
	case "unreachable":
		// No outbound network in this environment: the DNS check fails.
		if r.TrustScore != 15 {
			t.Errorf("offline expected score 15, got %d", r.TrustScore)
		}
		assertEvidenceLabels(t, r, []string{"DNS Lookup"})
	case "verified":
		if r.TrustScore != 60 {
			t.Errorf("trustScore = %d, want 60", r.TrustScore)
		}
		if r.Summary != "Domain resolves, HTTPS available, certificate valid." {
			t.Errorf("summary = %q", r.Summary)
		}
		assertEvidenceLabels(t, r, []string{"DNS Resolves", "HTTPS Available", "TLS Certificate Present", "HTTP Status OK"})
	default:
		t.Errorf("unexpected status %q (%+v)", r.Status, r)
	}
}

func TestVerifyDomain_Localhost(t *testing.T) {
	// localhost resolves (loopback) but serves no web stack here, so it must
	// not be reported as "unreachable".
	r := VerifyDomain("localhost")
	if r.Status == "unreachable" {
		t.Fatal("localhost must resolve, expected non-unreachable status")
	}
	if r.TrustScore < 0 || r.TrustScore > 100 {
		t.Errorf("trustScore %d out of [0,100]", r.TrustScore)
	}
	if len(r.Evidence) == 0 {
		t.Errorf("localhost result carries no evidence: %+v", r)
	}
}
