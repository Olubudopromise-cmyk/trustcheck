package claims

import (
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
