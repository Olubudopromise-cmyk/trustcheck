package reasoning

import (
	"strings"
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

func TestExplain_WithEvidence(t *testing.T) {
	got := Explain(68, classifier.TypeDomain,
		[]model.EvidenceItem{{Label: "DNS Resolves", Result: "pass", Points: 20}},
		[]model.EvidenceItem{{Label: "Expired Certificate", Result: "fail", Points: -30}},
	)

	if len(got) != 3 {
		t.Fatalf("expected 3 bullets, got %d: %v", len(got), got)
	}
	if !strings.Contains(got[0], "68") {
		t.Errorf("first bullet should mention the score, got %q", got[0])
	}
	if got[1] != "+ DNS Resolves" {
		t.Errorf("supporting evidence should be a + bullet, got %q", got[1])
	}
	if got[2] != "- Expired Certificate" {
		t.Errorf("contradicting evidence should be a - bullet, got %q", got[2])
	}
}

func TestExplain_NoEvidence(t *testing.T) {
	got := Explain(50, classifier.TypeCompany, nil, nil)
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "provisional") {
		t.Errorf("no-evidence case should say the score is provisional, got %q", joined)
	}
}

func TestExplain_NoContradicting(t *testing.T) {
	got := Explain(90, classifier.TypeEmail,
		[]model.EvidenceItem{{Label: "MX Record", Result: "pass", Points: 40}},
		nil,
	)
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "No contradicting evidence") {
		t.Errorf("expected a no-contradicting note, got %q", joined)
	}
}

func TestExplain_UnknownType(t *testing.T) {
	got := Explain(30, classifier.TypeUnknown,
		[]model.EvidenceItem{{Label: "Wording", Result: "info", Points: 0}}, nil)
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "unstructured text") {
		t.Errorf("unknown type should mention unstructured text, got %q", joined)
	}
}
