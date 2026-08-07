package interpretations

import (
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/claims"
	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

func TestGenerate_TextInput(t *testing.T) {
	got := Generate(Context{
		Input:     "NASA confirms aliens landed in Lagos yesterday.",
		InputType: classifier.TypeUnknown,
		Claim:     claims.Claim{MainClaim: "NASA confirms aliens landed in Lagos yesterday."},
		Status:    "verified",
		EvidenceFor: []model.EvidenceItem{
			{Label: "Credible language", Result: "pass", Points: 10},
			{Label: "Multiple references", Result: "pass", Points: 10},
		},
	})

	if len(got) < 2 || len(got) > 3 {
		t.Fatalf("expected 2-3 interpretations, got %d", len(got))
	}
	if got[0].Title != "Literal reading" {
		t.Errorf("most plausible interpretation for a supported claim should be literal, got %q", got[0].Title)
	}
	for _, it := range got {
		if it.Explanation == "" || it.Reasoning == "" {
			t.Errorf("interpretation %q is missing explanation or reasoning", it.Title)
		}
		if it.Confidence < 0 || it.Confidence > 100 {
			t.Errorf("interpretation %q confidence %d out of range", it.Title, it.Confidence)
		}
	}
}

func TestGenerate_TextUndermined(t *testing.T) {
	got := Generate(Context{
		Input:     "Breaking: scientists prove 5G causes the flu!",
		InputType: classifier.TypeUnknown,
		Claim:     claims.Claim{MainClaim: "Breaking: scientists prove 5G causes the flu."},
		Status:    "poor",
		Warnings: []model.WarningSignal{
			{Label: "No citations or sources", Severity: "high", Description: "No source."},
		},
	})

	if len(got) < 2 {
		t.Fatalf("expected multiple interpretations, got %d", len(got))
	}
	// A strong warning signal should lead with the sceptical reading.
	if got[0].Title == "Literal reading" {
		t.Errorf("contradicted claim should not lead with the literal reading: %+v", got)
	}
}

func TestGenerate_Identifier(t *testing.T) {
	got := Generate(Context{
		Input:     "google.com",
		InputType: classifier.TypeDomain,
		Claim:     claims.Claim{MainClaim: "google.com is legitimate."},
		Status:    "verified",
		EvidenceFor: []model.EvidenceItem{
			{Label: "DNS Resolves", Result: "pass", Points: 20},
			{Label: "HTTPS Available", Result: "pass", Points: 20},
			{Label: "Valid Certificate", Result: "pass", Points: 20},
		},
	})

	if len(got) < 2 {
		t.Fatalf("expected multiple interpretations, got %d", len(got))
	}
	if got[0].Title != "Genuine identifier" {
		t.Errorf("verified domain should lead with genuine reading, got %q", got[0].Title)
	}
}

func TestGenerate_IdentifierUndermined(t *testing.T) {
	got := Generate(Context{
		Input:     "free-crypto-win.xyz",
		InputType: classifier.TypeDomain,
		Claim:     claims.Claim{MainClaim: "free-crypto-win.xyz is legitimate."},
		Status:    "poor",
		EvidenceAgainst: []model.EvidenceItem{
			{Label: "No HTTPS", Result: "fail", Points: -20},
		},
	})

	if got[0].Title != "Lookalike or impersonation" {
		t.Errorf("contradicted domain should lead with lookalike reading, got %q", got[0].Title)
	}
}
