package warnings

import (
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
)

func TestDetect_SensationalAndNoCitations(t *testing.T) {
	got := Detect("SHOCKING!!: Doctors hate this amazing one weird trick that cures everything!", classifier.TypeUnknown)
	if len(got) == 0 {
		t.Fatal("expected warning signals for sensational clickbait")
	}
	labels := map[string]bool{}
	for _, s := range got {
		labels[s.Label] = true
		if s.Severity != SeverityHigh && s.Severity != SeverityMedium && s.Severity != SeverityLow {
			t.Errorf("signal %q has invalid severity %q", s.Label, s.Severity)
		}
	}
	for _, want := range []string{"Sensational headline", "Clickbait wording", "No citations or sources", "Excessive emotional language"} {
		if !labels[want] {
			t.Errorf("expected signal %q, got %v", want, labels)
		}
	}
}

func TestDetect_NoWarningsForPlainText(t *testing.T) {
	got := Detect("According to the Ministry of Health, cases rose by 12 percent in March 2024.", classifier.TypeUnknown)
	if len(got) != 0 {
		t.Errorf("plain, cited text should produce no warnings, got %+v", got)
	}
}

func TestDetect_DateAndCitationSuppress(t *testing.T) {
	got := Detect("The study published yesterday by Reuters reports new findings.", classifier.TypeUnknown)
	for _, s := range got {
		if s.Label == "No publication date" || s.Label == "No citations or sources" {
			t.Errorf("date and citation present but signal %q fired", s.Label)
		}
	}
}

func TestDetect_AiPattern(t *testing.T) {
	got := Detect("As an AI language model, I cannot provide an opinion on this matter.", classifier.TypeUnknown)
	found := false
	for _, s := range got {
		if s.Label == "Possible AI-generated pattern" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected AI-generated pattern signal, got %+v", got)
	}
}

func TestDetect_ManipulatedStatistics(t *testing.T) {
	got := Detect("This supplement is 100% guaranteed to double your energy.", classifier.TypeUnknown)
	found := false
	for _, s := range got {
		if s.Label == "Manipulated statistics" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected manipulated statistics signal, got %+v", got)
	}
}

func TestDetect_AnonymousAuthor(t *testing.T) {
	got := Detect("Anonymous sources say the company is collapsing.", classifier.TypeUnknown)
	found := false
	for _, s := range got {
		if s.Label == "Anonymous author" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected anonymous author signal, got %+v", got)
	}
}

func TestDetect_UnusualTLD(t *testing.T) {
	got := Detect("free-crypto-win.xyz", classifier.TypeDomain)
	found := false
	for _, s := range got {
		if s.Label == "Unusual top-level domain" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unusual TLD signal, got %+v", got)
	}
}

func TestDetect_ValidDomainHasNoSignals(t *testing.T) {
	got := Detect("google.com", classifier.TypeDomain)
	if len(got) != 0 {
		t.Errorf("google.com should have no warning signals, got %+v", got)
	}
}

func TestDetect_EmptyInput(t *testing.T) {
	if got := Detect("   ", classifier.TypeUnknown); len(got) != 0 {
		t.Errorf("empty input should produce no signals, got %+v", got)
	}
}
