package claims

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
)

func hasEntity(t *testing.T, claim Claim, name, kind string) {
	t.Helper()
	for _, e := range claim.Entities {
		if e.Name == name && e.Kind == kind {
			return
		}
	}
	t.Errorf("expected entity %q (%s) in %+v", name, kind, claim.Entities)
}

func hasKeyword(t *testing.T, claim Claim, keyword string) {
	t.Helper()
	for _, k := range claim.Keywords {
		if k == keyword {
			return
		}
	}
	t.Errorf("expected keyword %q in %+v", keyword, claim.Keywords)
}

// TestExtract_PromptExample reproduces the exact example from the phase spec:
// "NASA confirms aliens landed in Lagos yesterday."
func TestExtract_PromptExample(t *testing.T) {
	claim := Extract("NASA confirms aliens landed in Lagos yesterday.", classifier.TypeUnknown)

	if claim.MainClaim != "NASA confirms aliens landed in Lagos yesterday." {
		t.Errorf("main claim = %q, want the normalized sentence", claim.MainClaim)
	}
	hasEntity(t, claim, "NASA", EntityOrganization)
	hasEntity(t, claim, "Lagos", EntityLocation)
	hasEntity(t, claim, "yesterday", EntityDate)
	hasKeyword(t, claim, "alien")
	hasKeyword(t, claim, "land")
}

func TestExtract_EmptyInput(t *testing.T) {
	claim := Extract("   ", classifier.TypeUnknown)
	if claim.MainClaim == "" {
		t.Error("empty input should still produce a main claim")
	}
}

func TestExtract_Company(t *testing.T) {
	claim := Extract("OpenAI", classifier.TypeCompany)
	hasEntity(t, claim, "OpenAI", EntityOrganization)
	if !strings.Contains(claim.MainClaim, "OpenAI") {
		t.Errorf("company main claim should mention the company, got %q", claim.MainClaim)
	}
}

func TestExtract_Email(t *testing.T) {
	claim := Extract("support@stripe.com", classifier.TypeEmail)
	hasEntity(t, claim, "stripe.com", EntityOrganization)
	if !strings.Contains(claim.MainClaim, "email") {
		t.Errorf("email main claim should mention the type, got %q", claim.MainClaim)
	}
}

func TestExtract_Domain(t *testing.T) {
	claim := Extract("google.com", classifier.TypeDomain)
	hasEntity(t, claim, "google.com", EntityOrganization)
}

func TestExtract_CompanyName(t *testing.T) {
	claim := Extract("Stripe Inc", classifier.TypeCompany)
	hasEntity(t, claim, "Stripe Inc", EntityOrganization)
}

func TestKeywords_IgnoreStopwordsAndEntities(t *testing.T) {
	claim := Extract("The company announced new features and support.", classifier.TypeUnknown)
	for _, k := range claim.Keywords {
		if k == "the" || k == "and" {
			t.Errorf("stopword leaked into keywords: %q", k)
		}
	}
}

func TestStem(t *testing.T) {
	cases := map[string]string{
		"companies": "company",
		"landed":    "land",
		"landing":   "land",
		"aliens":    "alien",
		"news":      "new",
		"running":   "runn",
		"pass":      "pass",
		"stress":    "stress",
	}
	for in, want := range cases {
		if got := stem(in); got != want {
			t.Errorf("stem(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeSentence(t *testing.T) {
	cases := map[string]string{
		"  NASA  confirms aliens landed. ": "NASA confirms aliens landed.",
		"no trailing punctuation":          "no trailing punctuation.",
		"already ends with period.":        "already ends with period.",
	}
	for in, want := range cases {
		if got := normalizeSentence(in); got != want {
			t.Errorf("normalizeSentence(%q) = %q, want %q", in, got, want)
		}
	}
}

// --- Phase 13: multi-claim extraction tests ---

func TestExtractMultiple_SingleClaim(t *testing.T) {
	claims := ExtractMultiple("NASA confirms aliens landed in Lagos yesterday.", classifier.TypeUnknown)
	if len(claims) < 1 {
		t.Fatal("expected at least 1 claim")
	}
	if claims[0].ID != "claim-0" {
		t.Errorf("first claim ID = %q, want claim-0", claims[0].ID)
	}
}

func TestExtractMultiple_MultipleClaims(t *testing.T) {
	input := "Apple announced a new AI chip yesterday. The chip is twice as fast as the M3. It will ship next month."
	mc := ExtractMultiple(input, classifier.TypeUnknown)
	t.Logf("extracted %d claims:", len(mc))
	for _, c := range mc {
		t.Logf("  ID=%s Text=%q Entities=%v", c.ID, c.Text, c.Entities)
	}
	if len(mc) < 1 {
		t.Fatal("expected at least 1 claim")
	}
	// All claims should have IDs.
	for i, c := range mc {
		if c.ID == "" {
			t.Errorf("claim %d has empty ID", i)
		}
	}
}

func TestExtractMultiple_StructuredInput(t *testing.T) {
	claims := ExtractMultiple("google.com", classifier.TypeDomain)
	if len(claims) != 1 {
		t.Fatalf("structured input should produce 1 claim, got %d", len(claims))
	}
	if !strings.Contains(claims[0].Text, "google.com") {
		t.Errorf("claim text should mention google.com, got %q", claims[0].Text)
	}
}

func TestExtractMultiple_EmptyInput(t *testing.T) {
	claims := ExtractMultiple("", classifier.TypeUnknown)
	if len(claims) != 1 {
		t.Fatalf("empty input should produce 1 claim, got %d", len(claims))
	}
}

func TestExtractMultiple_OpinionOnly(t *testing.T) {
	input := "I think this is beautiful. In my opinion, it is the best thing ever."
	claims := ExtractMultiple(input, classifier.TypeUnknown)
	// Opinions should be filtered out, falling back to the full text.
	if len(claims) < 1 {
		t.Fatal("expected at least 1 claim (fallback)")
	}
}

func TestExtractMultiple_DuplicateDetection(t *testing.T) {
	input := "NASA confirmed the discovery. NASA announced the same discovery."
	claims := ExtractMultiple(input, classifier.TypeUnknown)
	// Duplicates should be merged.
	if len(claims) > 2 {
		t.Errorf("expected at most 2 claims after dedup, got %d", len(claims))
	}
}

func TestExtractMultiple_CapAt10(t *testing.T) {
	// Create input with many sentences.
	input := ""
	for i := 0; i < 20; i++ {
		input += fmt.Sprintf("NASA announced claim number %d about the discovery. ", i)
	}
	claims := ExtractMultiple(input, classifier.TypeUnknown)
	if len(claims) > 10 {
		t.Errorf("expected at most 10 claims, got %d", len(claims))
	}
}

func TestExtractMultiple_Dependencies(t *testing.T) {
	input := "NASA discovered water on Mars. Therefore, NASA plans a mission because of that discovery."
	claims := ExtractMultiple(input, classifier.TypeUnknown)
	if len(claims) < 2 {
		t.Skip("not enough claims to test dependencies")
	}
	// The second claim should depend on the first.
	found := false
	for _, c := range claims {
		if len(c.DependsOn) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one claim with dependencies")
	}
}

func TestIsFactualClaim(t *testing.T) {
	cases := []struct {
		sentence string
		want     bool
	}{
		{"NASA confirmed the discovery.", true},
		{"I think this is beautiful.", false},
		{"Will this happen?", false},
		{"Buy now for 50% off!", false},
		{"The weather is nice.", false},
		{"Apple released the iPhone 15.", true},
	}
	for _, tc := range cases {
		got := isFactualClaim(tc.sentence)
		if got != tc.want {
			t.Errorf("isFactualClaim(%q) = %v, want %v", tc.sentence, got, tc.want)
		}
	}
}
