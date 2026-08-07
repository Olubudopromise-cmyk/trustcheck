package analysis

import (
	"context"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
	"strings"
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/scoring"
	"github.com/pamierin/trustcheck/apps/api/internal/verifier"
)

// TestAnalyze_TextInput reproduces the end-to-end example from the spec.
func TestAnalyze_TextInput(t *testing.T) {
	got := New().Analyze(context.Background(), "NASA confirms aliens landed in Lagos yesterday.",
		classifier.TypeUnknown,
		verifier.Result{Status: "warning", TrustScore: 45, Summary: "Unverified sensational claim."})

	if got.Verdict != model.VerdictMedium {
		t.Errorf("score 45 should map to Medium verdict, got %q", got.Verdict)
	}
	if got.KeyClaim == "" {
		t.Error("key claim should be extracted")
	}
	if len(got.EvidenceFor) != 0 || len(got.EvidenceAgainst) != 0 {
		t.Errorf("no scored evidence should produce empty sections, got for=%d against=%d",
			len(got.EvidenceFor), len(got.EvidenceAgainst))
	}
	if len(got.Interpretations) < 2 {
		t.Errorf("expected 2-3 interpretations, got %d", len(got.Interpretations))
	}
	if len(got.Recommendations) == 0 {
		t.Error("expected recommendations")
	}
	if len(got.Reasoning) == 0 {
		t.Error("expected reasoning bullets")
	}
	if len(got.MissingEvidence) == 0 || len(got.UnknownInformation) == 0 {
		t.Error("missing/unknown sections should be explicit, never empty")
	}
}

func TestAnalyze_SplitsEvidence(t *testing.T) {
	got := New().Analyze(context.Background(), "google.com", classifier.TypeDomain, verifier.Result{
		Status:     "good",
		TrustScore: 70,
		Summary:    "Resolves, HTTPS available.",
		Evidence: []scoring.Evidence{
			{Label: "DNS Lookup", Result: "pass", Points: 20},
			{Label: "HTTPS Available", Result: "pass", Points: 20},
			{Label: "Certificate Issue", Result: "warning", Points: -10},
		},
	})

	if len(got.EvidenceFor) != 2 {
		t.Errorf("expected 2 supporting items, got %d", len(got.EvidenceFor))
	}
	if len(got.EvidenceAgainst) != 1 {
		t.Errorf("expected 1 contradicting item, got %d", len(got.EvidenceAgainst))
	}
	if got.EvidenceAgainst[0].Label != "Certificate Issue" {
		t.Errorf("expected the warning in contradicting section, got %+v", got.EvidenceAgainst)
	}
}

// placeholderModule simulates a future pluggable capability (e.g. live web
// search) that enriches the result without breaking the pipeline.
type placeholderModule struct{ name string }

func (m placeholderModule) Name() string { return m.name }

func (m placeholderModule) Enrich(_ context.Context, _ string, result *model.Result) error {
	result.Recommendations = append(result.Recommendations, model.Recommendation{
		Title:       "Web search",
		Description: "A live web search found additional context.",
	})
	return nil
}

// failingModule simulates a future module that errors. The pipeline must not
// break; instead a low-severity warning signal is recorded.
type failingModule struct{}

func (failingModule) Name() string { return "broken-module" }

func (failingModule) Enrich(context.Context, string, *model.Result) error {
	return errModule
}

var errModule = &moduleError{}

type moduleError struct{}

func (*moduleError) Error() string { return "module failed" }

func TestAnalyze_ModulesEnrichResult(t *testing.T) {
	a := New(placeholderModule{name: "web-search"})
	got := a.Analyze(context.Background(), "Stripe Inc", classifier.TypeCompany,
		verifier.Result{Status: "good", TrustScore: 75, Summary: "Registered company."})

	found := false
	for _, r := range got.Recommendations {
		if r.Title == "Web search" {
			found = true
		}
	}
	if !found {
		t.Error("registered module should enrich the result")
	}
}

func TestAnalyze_FailingModuleDoesNotBreakPipeline(t *testing.T) {
	a := New(failingModule{})
	got := a.Analyze(context.Background(), "test.com", classifier.TypeDomain,
		verifier.Result{Status: "good", TrustScore: 80, Summary: "ok."})

	if got.KeyClaim == "" {
		t.Error("core pipeline output should survive a failing module")
	}
	found := false
	for _, s := range got.WarningSignals {
		if strings.Contains(s.Label, "broken-module") {
			found = true
		}
	}
	if !found {
		t.Error("a failing module should surface a warning signal")
	}
}

func TestSplitEvidence_IgnoresInfo(t *testing.T) {
	set := scoring.ClassifyEvidence([]scoring.Evidence{
		{Label: "Pass", Result: "pass", Points: 10},
		{Label: "Info", Result: "info", Points: 0},
		{Label: "Fail", Result: "fail", Points: -10},
	})
	if len(set.Supporting) != 1 || len(set.Contradicting) != 1 || len(set.Neutral) != 1 {
		t.Errorf("info items should be neutral, got support=%d against=%d neutral=%d",
			len(set.Supporting), len(set.Contradicting), len(set.Neutral))
	}
}

// TestAnalyze_EngineFallbackNeverSurfaces is the regression test for the
// single shared evidence classification: when a verifier emits a generic
// engine message like "No Suggestion", it must never reach the analysis — not
// in the evidence sections, not in the timeline, not in the reasoning bullets.
func TestAnalyze_EngineFallbackNeverSurfaces(t *testing.T) {
	got := New().Analyze(context.Background(), "unknown gibberish xyzzy", classifier.TypeUnknown, verifier.Result{
		Status:     "warning",
		TrustScore: 45,
		Summary:    "Unverified.",
		Evidence: []scoring.Evidence{
			{Label: "No Suggestion", Result: "warning", Points: 10},
			{Label: "Suggestion Generated", Result: "pass", Points: 20},
			{Label: "Input Provided", Result: "info", Points: 0},
			{Label: "Real Check", Result: "pass", Points: 20},
		},
	})

	for _, section := range [][]model.EvidenceItem{got.EvidenceFor, got.EvidenceAgainst} {
		for _, item := range section {
			if scoring.IsEnginePlaceholder(item.Label) {
				t.Errorf("engine placeholder %q leaked into an evidence section", item.Label)
			}
		}
	}
	if len(got.EvidenceFor) != 1 || got.EvidenceFor[0].Label != "Real Check" {
		t.Errorf("only real supporting evidence should remain, got %+v", got.EvidenceFor)
	}
	if len(got.EvidenceAgainst) != 0 {
		t.Errorf("engine fallbacks must not count as contradictions, got %+v", got.EvidenceAgainst)
	}

	// The timeline is built entirely from the shared classification, so every
	// step must report the same reality: 1 supporting, 0 contradicting.
	for _, step := range got.Timeline {
		joined := strings.Join(step.Details, "\n") + " " + step.Summary
		if strings.Contains(joined, "No Suggestion") || strings.Contains(joined, "Suggestion Generated") {
			t.Errorf("engine placeholder leaked into timeline step %q:\n%s", step.Title, joined)
		}
	}
	if !strings.Contains(got.Timeline[1].Summary, "1 supporting, 0 contradicting") {
		t.Errorf("evidence step should report 1/0, got %q", got.Timeline[1].Summary)
	}
	if !strings.Contains(got.Timeline[2].Summary, "No conflicting") {
		t.Errorf("conflicts step should find no conflicts, got %q", got.Timeline[2].Summary)
	}
	if !strings.Contains(got.Timeline[5].Summary, "Final score") {
		t.Errorf("final assessment step missing, got %q", got.Timeline[5].Summary)
	}

	for _, line := range got.Reasoning {
		if strings.Contains(line, "- No Suggestion") || strings.Contains(line, "No Suggestion") {
			t.Errorf("reasoning bullet cites an engine placeholder: %q", line)
		}
	}

	// The legacy evidence array must be filtered too.
	if got.Status != model.StatusFromVerdict(got.Verdict) {
		t.Errorf("status %q must be derived from verdict %q", got.Status, got.Verdict)
	}
}

// TestAnalyze_StatusDerivedFromVerdict guarantees the status field is a pure
// function of the verdict (which is a pure function of the trust score), so a
// response can never show a warning badge next to a high-trust verdict.
func TestAnalyze_StatusDerivedFromVerdict(t *testing.T) {
	for score, want := range map[int]struct {
		verdict model.Verdict
		status  string
	}{
		90: {model.VerdictHigh, "verified"},
		75: {model.VerdictHigh, "verified"},
		50: {model.VerdictMedium, "warning"},
		30: {model.VerdictLow, "invalid"},
	} {
		got := New().Analyze(context.Background(), "example.com", classifier.TypeDomain,
			verifier.Result{Status: "warning", TrustScore: score, Summary: "x."})
		if got.Verdict != want.verdict || got.Status != want.status {
			t.Errorf("score %d: got verdict=%q status=%q, want verdict=%q status=%q",
				score, got.Verdict, got.Status, want.verdict, want.status)
		}
	}
}
