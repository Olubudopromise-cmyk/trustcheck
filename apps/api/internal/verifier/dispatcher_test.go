package verifier

import (
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
)

// TestVerify_RoutesToCorrectVerifier confirms every InputType is dispatched to
// the right engine.
//
// domain uses a ".invalid" host (RFC 2606, never resolves) so it
// deterministically returns the domain engine's unreachable result -- proving
// the domain route is taken and is NOT the placeholder. All other types
// (except email, covered in TestVerify_EmailRoutesToEngine) hit the
// placeholder engine and are asserted exactly.
func TestVerify_RoutesToCorrectVerifier(t *testing.T) {
	const placeholderSummary = "Verification engine not implemented yet."

	cases := []struct {
		name    string
		typ     classifier.InputType
		in      string
		want    Result
	}{
		{"domain -> domainVerifier", classifier.TypeDomain, "nonexistent.invalid",
			Result{Status: "unreachable", TrustScore: 15, Summary: "Domain does not resolve."}},
		{"url -> placeholder", classifier.TypeURL, "https://google.com",
			Result{Status: "not_implemented", TrustScore: 0, Summary: placeholderSummary}},
		{"ipv4 -> placeholder", classifier.TypeIPv4, "8.8.8.8",
			Result{Status: "not_implemented", TrustScore: 0, Summary: placeholderSummary}},
		{"ipv6 -> placeholder", classifier.TypeIPv6, "2606:4700:4700::1111",
			Result{Status: "not_implemented", TrustScore: 0, Summary: placeholderSummary}},
		{"phone -> placeholder", classifier.TypePhone, "+15551234567",
			Result{Status: "not_implemented", TrustScore: 0, Summary: placeholderSummary}},
		{"company -> placeholder", classifier.TypeCompany, "OpenAI",
			Result{Status: "not_implemented", TrustScore: 0, Summary: placeholderSummary}},
		{"unknown -> placeholder", classifier.TypeUnknown, "???",
			Result{Status: "not_implemented", TrustScore: 0, Summary: placeholderSummary}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Verify(tc.typ, tc.in)
			if got != tc.want {
				t.Errorf("Verify(%s, %q) = %+v, want %+v", tc.typ, tc.in, got, tc.want)
			}
		})
	}
}

// TestVerify_UnknownTypeFallsBack ensures an unregistered InputType cannot
// panic and is treated as a placeholder.
func TestVerify_UnknownTypeFallsBack(t *testing.T) {
	got := Verify(classifier.InputType("nonsense"), "whatever")
	if got.Status != "not_implemented" || got.TrustScore != 0 {
		t.Errorf("unregistered type should fall back to placeholder, got %+v", got)
	}
}

// TestVerify_EmailRoutesToEngine proves the email type now routes to the real
// emailVerifier (result is NOT the placeholder). The outcome is network
// tolerant: online the engine returns a reachable status, offline DNS fails.
func TestVerify_EmailRoutesToEngine(t *testing.T) {
	got := Verify(classifier.TypeEmail, "support@stripe.com")
	t.Logf("email -> %+v", got)

	if got.Status == "not_implemented" || got.Summary == "Verification engine not implemented yet." {
		t.Fatalf("email type should route to emailVerifier, got placeholder: %+v", got)
	}
	if got.TrustScore < 0 || got.TrustScore > 100 {
		t.Errorf("trustScore %d out of [0,100]", got.TrustScore)
	}
}
