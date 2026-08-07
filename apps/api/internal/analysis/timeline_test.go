package analysis

import (
	"strings"
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

// TestBuildTimeline_HasSixSteps guarantees the full reasoning flow is exposed.
func TestBuildTimeline_HasSixSteps(t *testing.T) {
	steps := buildTimeline(minimalResult(), 0)

	wantTitles := []string{
		"Claim Detected",
		"Evidence Gathered",
		"Conflicts Identified",
		"Risk Signals Detected",
		"AI Reasoning",
		"Final Assessment",
	}
	if len(steps) != len(wantTitles) {
		t.Fatalf("expected %d timeline steps, got %d", len(wantTitles), len(steps))
	}
	for i, want := range wantTitles {
		if steps[i].Title != want {
			t.Errorf("step %d title = %q, want %q", i, steps[i].Title, want)
		}
		if steps[i].Summary == "" {
			t.Errorf("step %q has an empty summary", want)
		}
		if len(steps[i].Details) == 0 {
			t.Errorf("step %q has no detail lines", want)
		}
	}
}

func TestBuildTimeline_ClaimStep(t *testing.T) {
	result := minimalResult()
	result.KeyClaim = "NASA confirms aliens landed in Lagos yesterday."
	result.Entities = []model.Entity{
		{Name: "NASA", Kind: model.EntityOrganization},
		{Name: "Lagos", Kind: model.EntityLocation},
		{Name: "yesterday", Kind: model.EntityDate},
	}
	result.Keywords = []string{"alien", "land"}

	step := claimStep(result)
	joined := strings.Join(step.Details, "\n")
	for _, want := range []string{"Claim: NASA", "Organizations: NASA", "Locations: Lagos", "Dates: yesterday", "Keywords: alien, land"} {
		if !strings.Contains(joined, want) {
			t.Errorf("claim step missing %q, got:\n%s", want, joined)
		}
	}
}

func TestBuildTimeline_EvidenceStepStatesNoneFound(t *testing.T) {
	result := minimalResult() // no evidence at all
	step := evidenceStep(result, 0)

	if !strings.Contains(strings.Join(step.Details, "\n"), "None were invented") {
		t.Errorf("evidence step should state no sources were invented, got %v", step.Details)
	}
}

func TestBuildTimeline_ConflictsStep(t *testing.T) {
	noConflict := conflictsStep(minimalResult())
	if noConflict.Summary != "No conflicting evidence identified." {
		t.Errorf("expected no-conflict summary, got %q", noConflict.Summary)
	}

	result := minimalResult()
	result.EvidenceAgainst = []model.EvidenceItem{{Label: "Expired Certificate", Result: "fail", Points: -30}}
	conflict := conflictsStep(result)
	if !strings.Contains(conflict.Summary, "1 conflicting") {
		t.Errorf("expected 1 conflict, got %q", conflict.Summary)
	}
	if conflict.Details[0] != "• Expired Certificate" {
		t.Errorf("conflict detail wrong: %v", conflict.Details)
	}
}

// TestBuildTimeline_EngineFallbackIsNotAConflict ensures generic engine
// messages like "No Suggestion" are never presented as conflicting evidence.
func TestBuildTimeline_EngineFallbackIsNotAConflict(t *testing.T) {
	result := minimalResult()
	result.EvidenceAgainst = []model.EvidenceItem{{Label: "No Suggestion", Result: "warning", Points: 10}}
	step := conflictsStep(result)
	if step.Summary != "No conflicting evidence identified." {
		t.Errorf("engine fallback should not be a conflict, got %q", step.Summary)
	}
}

func TestBuildTimeline_RiskSignalsImpact(t *testing.T) {
	result := minimalResult()
	result.WarningSignals = []model.WarningSignal{
		{Label: "No citations", Severity: "high", Description: "No source."},
		{Label: "Clickbait wording", Severity: "medium", Description: "Clicks."},
	}

	step := riskSignalsStep(result)
	joined := strings.Join(step.Details, "\n")
	if !strings.Contains(joined, "-15 points") {
		t.Errorf("high severity should estimate -15 points, got %v", step.Details)
	}
	if !strings.Contains(joined, "-8 points") {
		t.Errorf("medium severity should estimate -8 points, got %v", step.Details)
	}

	if none := riskSignalsStep(minimalResult()); none.Summary != "No risk signals detected." {
		t.Errorf("expected no-risk summary, got %q", none.Summary)
	}
}

func TestBuildTimeline_FinalAssessment(t *testing.T) {
	result := minimalResult()
	result.TrustScore = 68
	result.Verdict = model.VerdictMedium
	result.Confidence = 72
	result.EvidenceFor = []model.EvidenceItem{{Label: "DNS Resolves", Result: "pass", Points: 20}}
	result.Recommendations = []model.Recommendation{{Title: "Check WHOIS", Description: "Look it up."}}

	step := finalAssessmentStep(result)
	joined := strings.Join(step.Details, "\n")
	for _, want := range []string{"68 / 100", "Medium", "72%", "Key strengths: DNS Resolves", "Top recommendation: Check WHOIS."} {
		if !strings.Contains(joined, want) {
			t.Errorf("final assessment missing %q, got:\n%s", want, joined)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate short string should be unchanged, got %q", got)
	}
	if got := truncate("a longer string", 6); !strings.HasSuffix(got, "…") {
		t.Errorf("truncated string should end with ellipsis, got %q", got)
	}
}

// minimalResult returns an analysis result with every field set to a safe
// zero value so timeline builders can be exercised in isolation.
func minimalResult() model.Result {
	return model.Result{
		KeyClaim:        "test claim",
		Verdict:         model.VerdictMedium,
		TrustScore:      50,
		Confidence:      ConfidenceMedium,
		Reasoning:       []string{"Reasoning bullet."},
		Entities:        []model.Entity{{Name: "Test", Kind: model.EntityOrganization}},
		Keywords:        []string{"test"},
		Recommendations: []model.Recommendation{{Title: "Verify", Description: "Verify."}},
	}
}
