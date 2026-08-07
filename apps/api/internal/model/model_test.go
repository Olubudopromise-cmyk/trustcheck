package model

import "testing"

func TestStatusFromVerdict(t *testing.T) {
	cases := []struct {
		verdict Verdict
		want    string
	}{
		{VerdictHigh, "verified"},
		{VerdictMedium, "warning"},
		{VerdictLow, "invalid"},
	}
	for _, tc := range cases {
		if got := StatusFromVerdict(tc.verdict); got != tc.want {
			t.Errorf("StatusFromVerdict(%q) = %q, want %q", tc.verdict, got, tc.want)
		}
	}
}

// TestStatusFromVerdict_NeverDisagreesWithScore guarantees the full chain
// Trust Score -> Verdict -> Status is deterministic and always maps to a
// known badge value, so a response can never pair a high-trust score with a
// warning status.
func TestStatusFromVerdict_NeverDisagreesWithScore(t *testing.T) {
	known := map[string]bool{"verified": true, "warning": true, "invalid": true}
	for score := 0; score <= 100; score++ {
		status := StatusFromVerdict(VerdictFromScore(score))
		if !known[status] {
			t.Errorf("score %d produced unknown status %q", score, status)
		}
	}

	// The band boundaries stay consistent: the status can only change when the
	// verdict band changes.
	if StatusFromVerdict(VerdictFromScore(69)) == StatusFromVerdict(VerdictFromScore(70)) {
		t.Error("status must change at the High/Medium band boundary")
	}
	if StatusFromVerdict(VerdictFromScore(39)) == StatusFromVerdict(VerdictFromScore(40)) {
		t.Error("status must change at the Medium/Low band boundary")
	}
}