package scoring

import "testing"

func TestIsEnginePlaceholder(t *testing.T) {
	for _, label := range []string{"No Suggestion", "Suggestion Generated", "Input Provided"} {
		if !IsEnginePlaceholder(label) {
			t.Errorf("%q should be recognized as an engine placeholder", label)
		}
	}
	if IsEnginePlaceholder("Real Check") {
		t.Error("real checks must never be treated as placeholders")
	}
}

func TestClassifyEvidence_Buckets(t *testing.T) {
	set := ClassifyEvidence([]Evidence{
		{Label: "Pass", Result: "pass", Points: 20},
		{Label: "Warn", Result: "warning", Points: -10},
		{Label: "Fail", Result: "fail", Points: -30},
		{Label: "Info", Result: "info", Points: 0},
	})

	if len(set.Supporting) != 1 || set.Supporting[0].Label != "Pass" {
		t.Errorf("supporting bucket wrong: %+v", set.Supporting)
	}
	if len(set.Contradicting) != 2 {
		t.Errorf("contradicting bucket wrong: %+v", set.Contradicting)
	}
	if len(set.Neutral) != 1 || set.Neutral[0].Label != "Info" {
		t.Errorf("neutral bucket wrong: %+v", set.Neutral)
	}
}

func TestClassifyEvidence_FiltersPlaceholders(t *testing.T) {
	set := ClassifyEvidence([]Evidence{
		{Label: "No Suggestion", Result: "warning", Points: 10},
		{Label: "Suggestion Generated", Result: "pass", Points: 20},
		{Label: "Input Provided", Result: "info", Points: 0},
		{Label: "DNS Resolves", Result: "pass", Points: 20},
	})

	if len(set.Supporting) != 1 || set.Supporting[0].Label != "DNS Resolves" {
		t.Errorf("placeholders leaked into supporting: %+v", set.Supporting)
	}
	if len(set.Contradicting) != 0 {
		t.Errorf("placeholders must not be contradictions: %+v", set.Contradicting)
	}
	if len(set.Neutral) != 0 {
		t.Errorf("placeholders must not be neutral evidence: %+v", set.Neutral)
	}
}

func TestFilterEnginePlaceholders(t *testing.T) {
	in := []Evidence{
		{Label: "No Suggestion", Result: "warning", Points: 10},
		{Label: "Real", Result: "pass", Points: 20},
		{Label: "Input Provided", Result: "info", Points: 0},
	}
	got := FilterEnginePlaceholders(in)
	if len(got) != 1 || got[0].Label != "Real" {
		t.Errorf("expected only real evidence to remain, got %+v", got)
	}
}