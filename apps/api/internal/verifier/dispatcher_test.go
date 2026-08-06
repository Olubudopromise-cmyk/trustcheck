package verifier

import (
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
)

// TestVerify_RoutesToCorrectVerifier confirms every InputType is dispatched to
// the right engine. The domain input uses a ".invalid" host (RFC 2606, never
// resolves) so it deterministically returns the domain engine's unreachable
// result — proving the domain route is taken and is NOT the placeholder.
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
		{"email -> placeholder", classifier.TypeEmail, "support@stripe.com",
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
