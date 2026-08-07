package analysis

import (
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

func TestVerdictFromScore(t *testing.T) {
	cases := []struct {
		score int
		want  model.Verdict
	}{
		{0, model.VerdictLow},
		{39, model.VerdictLow},
		{40, model.VerdictMedium},
		{50, model.VerdictMedium},
		{69, model.VerdictMedium},
		{70, model.VerdictHigh},
		{90, model.VerdictHigh},
		{100, model.VerdictHigh},
	}
	for _, c := range cases {
		if got := VerdictFromScore(c.score); got != c.want {
			t.Errorf("VerdictFromScore(%d) = %q, want %q", c.score, got, c.want)
		}
	}
}

func TestConfidenceOf(t *testing.T) {
	mk := func(n int) []model.EvidenceItem {
		items := make([]model.EvidenceItem, n)
		for i := range items {
			items[i] = model.EvidenceItem{Label: "check", Result: "pass", Points: 10}
		}
		return items
	}

	cases := []struct {
		name      string
		forItems  []model.EvidenceItem
		against   []model.EvidenceItem
		inputType classifier.InputType
		want      int
	}{
		{"rich evidence", mk(3), mk(1), classifier.TypeDomain, ConfidenceHigh},
		{"moderate evidence", mk(2), nil, classifier.TypeEmail, ConfidenceMedium},
		{"single check", mk(1), nil, classifier.TypePhone, ConfidenceLow},
		{"no evidence", nil, nil, classifier.TypeCompany, ConfidenceLow},
		{"unknown type with evidence is discounted", mk(4), nil, classifier.TypeUnknown, ConfidenceHigh - 15},
		{"unknown type without evidence", nil, nil, classifier.TypeUnknown, ConfidenceLow - 15},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := confidenceOf(c.forItems, c.against, c.inputType); got != c.want {
				t.Errorf("confidenceOf() = %d, want %d", got, c.want)
			}
		})
	}
}
