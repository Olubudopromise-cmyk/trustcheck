package scoring

import "testing"

func TestBuilderEvidence(t *testing.T) {
	b := New()
	b.Pass("Base", 40)
	b.Warning("Penalty", -10)
	b.Fail("Bad", -30)
	b.Info("Note")

	if got := b.Score(); got != 0 {
		t.Errorf("score = %d, want 0", got)
	}

	want := []Evidence{
		{"Base", "pass", 40},
		{"Penalty", "warning", -10},
		{"Bad", "fail", -30},
		{"Note", "info", 0},
	}
	ev := b.Evidence()
	if len(ev) != len(want) {
		t.Fatalf("evidence len = %d, want %d (%+v)", len(ev), len(want), ev)
	}
	for i, w := range want {
		if ev[i] != w {
			t.Errorf("evidence[%d] = %+v, want %+v (ordering not preserved)", i, ev[i], w)
		}
	}
}

func TestBuilderEvidenceClamps(t *testing.T) {
	b := New()
	b.Pass("Large", 120)
	if got := b.Score(); got != 100 {
		t.Errorf("overflow score = %d, want 100", got)
	}
	b.Fail("Huge", -200)
	if got := b.Score(); got != 0 {
		t.Errorf("underflow score = %d, want 0", got)
	}
}

func TestBuilderEvidenceNegativePoints(t *testing.T) {
	b := New()
	b.Warning("Redirect", -10)
	b.Fail("Expired Cert", -30)
	ev := b.Evidence()
	if len(ev) != 2 {
		t.Fatalf("evidence len = %d, want 2", len(ev))
	}
	if ev[0].Points != -10 || ev[1].Points != -30 {
		t.Errorf("negative points not preserved: %+v", ev)
	}
}

func TestBuilderEvidenceInfoNoPoints(t *testing.T) {
	b := New()
	b.Info("Normalized")
	if got := b.Score(); got != 0 {
		t.Errorf("info score = %d, want 0", got)
	}
	if ev := b.Evidence(); len(ev) != 1 || ev[0].Result != "info" || ev[0].Points != 0 {
		t.Errorf("info evidence = %+v", ev)
	}
}
