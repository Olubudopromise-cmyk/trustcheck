package perspectives

import (
	"strings"
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

func TestSupportingEvidence_GroupsBySourceCategory(t *testing.T) {
	groups := SupportingEvidence([]model.EvidenceItem{
		{Label: "DNS Lookup", Result: "pass", Points: 20},
		{Label: "HTTPS Available", Result: "pass", Points: 20},
		{Label: "MX Record", Result: "pass", Points: 20},
	})

	if len(groups) < 2 {
		t.Fatalf("expected multiple source categories, got %d", len(groups))
	}
	joined := ""
	for _, g := range groups {
		joined += g.Category + "|"
		for _, item := range g.Items {
			if item.Credibility != "high" {
				t.Errorf("passing check should have high credibility, got %q", item.Credibility)
			}
			if item.Summary == "" {
				t.Error("source evidence should have a summary")
			}
		}
	}
	if !strings.Contains(joined, "Network infrastructure") {
		t.Errorf("DNS/MX checks should be grouped under network infrastructure, got %q", joined)
	}
	if !strings.Contains(joined, "Transport security") {
		t.Errorf("HTTPS check should be grouped under transport security, got %q", joined)
	}
}

func TestSupportingEvidence_Empty(t *testing.T) {
	if groups := SupportingEvidence(nil); len(groups) != 0 {
		t.Errorf("no evidence should produce no source groups, got %d", len(groups))
	}
}

func TestContradictingEvidence_BuildsDisagreement(t *testing.T) {
	contradictions := ContradictingEvidence("The site is safe.", []model.EvidenceItem{
		{Label: "Expired Certificate", Result: "fail", Points: -30},
	})

	if len(contradictions) != 1 {
		t.Fatalf("expected 1 contradiction, got %d", len(contradictions))
	}
	c := contradictions[0]
	if c.SourceA != "Submitted claim" || c.ClaimA != "The site is safe." {
		t.Errorf("contradiction should frame claim vs source, got %+v", c)
	}
	if c.ConfidenceInContradiction != 80 {
		t.Errorf("fail should give 80 confidence in contradiction, got %d", c.ConfidenceInContradiction)
	}
	if !strings.Contains(c.WhyTheyDisagree, "failed") {
		t.Errorf("fail contradiction should say the check failed, got %q", c.WhyTheyDisagree)
	}
}

func TestContradictingEvidence_EmptyWhenNoOpposingEvidence(t *testing.T) {
	if got := ContradictingEvidence("claim", nil); got != nil {
		t.Errorf("no opposing evidence should produce no contradictions, got %+v", got)
	}
}

// TestContradictingEvidence_EngineFallbackIsNotAConflict ensures generic
// engine messages like "No Suggestion" are never presented as contradictions.
func TestContradictingEvidence_EngineFallbackIsNotAConflict(t *testing.T) {
	got := ContradictingEvidence("claim", []model.EvidenceItem{
		{Label: "No Suggestion", Result: "warning", Points: 10},
	})
	if got != nil {
		t.Errorf("engine fallback should not be a contradiction, got %+v", got)
	}
}

func TestAISummary_IgnoresEngineFallbackCount(t *testing.T) {
	result := minimalResult()
	result.EvidenceAgainst = []model.EvidenceItem{{Label: "No Suggestion", Result: "warning", Points: 10}}
	summary := AISummary(result)
	if strings.Contains(summary, "contradicting check") {
		t.Errorf("engine fallback should not count as contradicting evidence, got %q", summary)
	}
}

func TestMissingInformation_GroundsItems(t *testing.T) {
	items := MissingInformation(
		[]string{"Not verified: WHOIS registration data."},
		[]string{"The author of this content is unknown."},
		[]model.WarningSignal{{Label: "No citations or sources", Severity: "high"}},
	)

	if len(items) < 3 {
		t.Fatalf("expected at least 3 missing items, got %d", len(items))
	}
	for _, item := range items {
		if item.Item == "" || item.WhyItMatters == "" {
			t.Errorf("missing item must have text and reason, got %+v", item)
		}
	}
}

func TestMissingInformation_NoDuplicates(t *testing.T) {
	items := MissingInformation(
		[]string{"Not verified: WHOIS registration data.", "Not verified: WHOIS registration data."},
		nil, nil,
	)
	if len(items) != 1 {
		t.Errorf("duplicate missing items should be collapsed, got %d", len(items))
	}
}

func TestConfidenceBreakdown_HasSixUserFacingMetrics(t *testing.T) {
	b := ConfidenceBreakdown(minimalResult())

	if len(b.Metrics) != 6 {
		t.Fatalf("expected 6 confidence metrics, got %d", len(b.Metrics))
	}
	if b.Overall != 72 {
		t.Errorf("overall should mirror analysis confidence, got %d", b.Overall)
	}
	for _, m := range b.Metrics {
		if m.Name == "" || m.Note == "" {
			t.Errorf("metric must have name and note, got %+v", m)
		}
		if m.Score < 0 || m.Score > 100 {
			t.Errorf("metric score out of range: %d", m.Score)
		}
	}
}

func TestConfidenceBreakdown_NoScoredEvidence(t *testing.T) {
	result := minimalResult()
	result.EvidenceFor = nil
	result.EvidenceAgainst = nil

	b := ConfidenceBreakdown(result)
	for _, m := range b.Metrics {
		if m.Score < 0 || m.Score > 100 {
			t.Errorf("metric score out of range for empty evidence: %d", m.Score)
		}
	}
}

func TestConfidenceBreakdown_ConsistencyAllOneDirection(t *testing.T) {
	result := minimalResult()
	result.EvidenceFor = []model.EvidenceItem{{Label: "A", Result: "pass", Points: 10}}
	result.EvidenceAgainst = nil

	score := 0
	for _, m := range ConfidenceBreakdown(result).Metrics {
		if m.Name == "Evidence consistency" {
			score = m.Score
		}
	}
	if score != 100 {
		t.Errorf("all checks agreeing should give 100 consistency, got %d", score)
	}
}

func TestAISummary_UnderWordLimitAndHonest(t *testing.T) {
	result := minimalResult()
	summary := AISummary(result)

	words := len(strings.Fields(summary))
	if words > 120 {
		t.Errorf("AI summary must be ≤120 words, got %d", words)
	}
	if summary == "" {
		t.Error("AI summary should not be empty")
	}
}

func TestAISummary_EmptyEvidenceIsExplicit(t *testing.T) {
	result := minimalResult()
	result.EvidenceFor = nil
	result.EvidenceAgainst = nil

	summary := AISummary(result)
	if !strings.Contains(summary, "No scored evidence") {
		t.Errorf("empty evidence should be stated explicitly, got %q", summary)
	}
}

func TestSuggestedReading_NeverFabricatesLinks(t *testing.T) {
	items, note := SuggestedReading(classifier.TypeUnknown, []model.Entity{
		{Name: "NASA", Kind: model.EntityOrganization},
	})

	if len(items) == 0 {
		t.Fatal("expected generic reading guidance")
	}
	if note == "" {
		t.Error("suggested reading should carry an honest note")
	}
	for _, item := range items {
		if item.Title == "" || item.Publisher == "" || item.WhyItHelps == "" {
			t.Errorf("reading item incomplete: %+v", item)
		}
	}
}

func TestWhatChanged_NeverFabricatesTimeline(t *testing.T) {
	events, note := WhatChanged(minimalResult())

	if events != nil {
		t.Error("no dated history should be observed, got fabricated events")
	}
	if note == "" {
		t.Error("what-changed should carry an honest statement")
	}
}

// minimalResult returns a result with enough populated fields for the
// perspectives builders to be exercised in isolation.
func minimalResult() model.Result {
	return model.Result{
		KeyClaim:           "Test claim.",
		Verdict:            model.VerdictMedium,
		TrustScore:         68,
		Status:             "good",
		Confidence:         72,
		EvidenceFor:        []model.EvidenceItem{{Label: "DNS Lookup", Result: "pass", Points: 20}},
		EvidenceAgainst:    []model.EvidenceItem{{Label: "Expired Certificate", Result: "fail", Points: -30}},
		MissingEvidence:    []string{"Not verified: WHOIS registration data."},
		WarningSignals:     []model.WarningSignal{{Label: "No citations or sources", Severity: "high"}},
		UnknownInformation: []string{"The author of this content is unknown."},
	}
}
