package recommendations

import (
	"strings"
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

func TestGenerate_HighVerdictNoCaution(t *testing.T) {
	got := Generate(classifier.TypeDomain, model.VerdictHigh)
	if len(got) == 0 {
		t.Fatal("expected recommendations")
	}
	for _, r := range got {
		if r.Title == "Treat with caution" {
			t.Errorf("high verdict should not include the caution reminder")
		}
	}
}

func TestGenerate_LowVerdictPreprendsCaution(t *testing.T) {
	got := Generate(classifier.TypeDomain, model.VerdictLow)
	if got[0].Title != "Treat with caution" {
		t.Errorf("low verdict should start with the caution reminder, got %q", got[0].Title)
	}
}

func TestGenerate_TextHasGenericRecommendations(t *testing.T) {
	got := Generate(classifier.TypeUnknown, model.VerdictMedium)
	joined := strings.Join(titles(got), " ")
	if !strings.Contains(joined, "independent news") {
		t.Errorf("free-text recommendations should mention independent news, got %v", titles(got))
	}
	if !strings.Contains(joined, "publication date") {
		t.Errorf("free-text recommendations should mention publication date, got %v", titles(got))
	}
}

func TestGenerate_EveryTypeReturnsSomething(t *testing.T) {
	for _, typ := range []classifier.InputType{
		classifier.TypeDomain, classifier.TypeURL, classifier.TypeEmail,
		classifier.TypePhone, classifier.TypeCompany, classifier.TypeIPv4,
		classifier.TypeIPv6, classifier.TypeUnknown,
	} {
		if got := Generate(typ, model.VerdictHigh); len(got) == 0 {
			t.Errorf("type %q should return recommendations", typ)
		}
	}
}

func titles(recs []model.Recommendation) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.Title
	}
	return out
}
