package analysis

import (
	"strings"
	"testing"
	"unicode/utf8"

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

// TestBuildTimeline_AllStepsAgree guarantees every timeline section that
// reports evidence counts reads the same pre-classified slices, so the
// Evidence Gathered, Conflicts Identified, and Final Assessment steps can
// never contradict each other. This is the integrity contract of the shared
// evidence classification.
func TestBuildTimeline_AllStepsAgree(t *testing.T) {
	result := minimalResult()
	result.EvidenceFor = []model.EvidenceItem{
		{Label: "DNS Resolves", Result: "pass", Points: 20},
		{Label: "HTTPS Available", Result: "pass", Points: 20},
	}
	result.EvidenceAgainst = []model.EvidenceItem{
		{Label: "Expired Certificate", Result: "fail", Points: -30},
	}

	evidence := evidenceStep(result, 0)
	conflicts := conflictsStep(result)
	final := finalAssessmentStep(result)

	if !strings.Contains(evidence.Summary, "2 supporting, 1 contradicting") {
		t.Errorf("evidence step summary = %q", evidence.Summary)
	}
	if !strings.Contains(conflicts.Summary, "1 conflicting") {
		t.Errorf("conflicts step summary = %q", conflicts.Summary)
	}
	if !strings.Contains(strings.Join(final.Details, "\n"), "1 contradicting check(s)") {
		t.Errorf("final assessment should report 1 contradicting check, got %v", final.Details)
	}

	// The AI Reasoning step derives its one-liner from the same counts.
	summary := reasoningSummary(result)
	for _, want := range []string{"2 supporting", "1 contradicting"} {
		if !strings.Contains(summary, want) {
			t.Errorf("reasoning summary should mention %q, got %q", want, summary)
		}
	}
}

// TestReasoningSummary_DataDriven covers the evidence-count based summary, so
// the reasoning step never contradicts the evidence sections above it.
func TestReasoningSummary_DataDriven(t *testing.T) {
	with := func(forCount, againstCount int) model.Result {
		r := minimalResult()
		for i := 0; i < forCount; i++ {
			r.EvidenceFor = append(r.EvidenceFor, model.EvidenceItem{Label: "S", Result: "pass", Points: 10})
		}
		for i := 0; i < againstCount; i++ {
			r.EvidenceAgainst = append(r.EvidenceAgainst, model.EvidenceItem{Label: "C", Result: "fail", Points: -10})
		}
		return r
	}

	cases := []struct {
		name string
		in   model.Result
		want string
	}{
		{"no evidence", with(0, 0), "No scored evidence"},
		{"leaning positive", with(2, 0), "leans positive"},
		{"leaning negative", with(0, 2), "leans negative"},
		{"balanced", with(1, 1), "mixed"},
	}
	for _, tc := range cases {
		got := reasoningSummary(tc.in)
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: summary %q should contain %q", tc.name, got, tc.want)
		}
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

// TestTruncate_UTF8 ensures multibyte characters are never split mid-rune, so
// no replacement characters or corrupted text can appear in the timeline.
func TestTruncate_UTF8(t *testing.T) {
	input := "🚀🚀🚀🚀 rocket launch"
	got := truncate(input, 4)
	if !strings.Contains(got, "🚀🚀🚀🚀") {
		t.Errorf("expected 4 intact rockets, got %q", got)
	}
	if !utf8.ValidString(got) {
		t.Errorf("truncated string is not valid UTF-8: %q", got)
	}

	long := "abcde"
	cut := truncate(long, 3)
	if cut != "abc…" {
		t.Errorf("ascii truncation wrong: %q", cut)
	}
	if !utf8.ValidString(cut) {
		t.Errorf("ascii truncation produced invalid UTF-8: %q", cut)
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
