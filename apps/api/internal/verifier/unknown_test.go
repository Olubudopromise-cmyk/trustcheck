package verifier

import "testing"

func TestUnknownVerifier_Suggestions(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Result
	}{
		{"domain missing TLD", "google",
			Result{Status: "suggestion", TrustScore: 20, Summary: "Did you mean the domain google.com?"}},
		{"www prefix", "www.google",
			Result{Status: "suggestion", TrustScore: 20, Summary: "Did you mean www.google.com?"}},
		{"malformed http", "http:/google.com",
			Result{Status: "suggestion", TrustScore: 20, Summary: "Did you mean http://google.com?"}},
		{"malformed https", "https:/openai.com",
			Result{Status: "suggestion", TrustScore: 20, Summary: "Did you mean https://openai.com?"}},
		{"digits only", "2348012345678",
			Result{Status: "suggestion", TrustScore: 20, Summary: "Did you mean +2348012345678?"}},
		{"email missing TLD", "user@gmail",
			Result{Status: "suggestion", TrustScore: 20, Summary: "Did you mean user@gmail.com?"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := unknownVerifier{}.Verify(tc.in)
			assertResult(t, got, tc.want)
			assertEvidenceLabels(t, got, []string{"Suggestion Generated"})
		})
	}
}

func TestUnknownVerifier_Whitespace(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Result
	}{
		{"padded domain", "   google   ",
			Result{Status: "suggestion", TrustScore: 20, Summary: "Did you mean the domain google.com?"}},
		{"padded www", " www.google ",
			Result{Status: "suggestion", TrustScore: 20, Summary: "Did you mean www.google.com?"}},
		{"padded digits", "\t2348012345678\n",
			Result{Status: "suggestion", TrustScore: 20, Summary: "Did you mean +2348012345678?"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := unknownVerifier{}.Verify(tc.in)
			assertResult(t, got, tc.want)
			assertEvidenceLabels(t, got, []string{"Suggestion Generated"})
		})
	}
}

func TestUnknownVerifier_NoSuggestion(t *testing.T) {
	for _, in := range []string{"???", "!!!!", "__&__", "Hello!! World", "user@a@b"} {
		got := unknownVerifier{}.Verify(in)
		if got.Status != "unknown" || got.TrustScore != 10 ||
			got.Summary != "Unable to classify the input." {
			t.Errorf("Verify(%q) = %+v, want unknown/10/Unable to classify the input.", in, got)
		}
		assertEvidenceLabels(t, got, []string{"No Suggestion"})
	}
}

func TestUnknownVerifier_Empty(t *testing.T) {
	for _, in := range []string{"", "   ", "\t\n "} {
		got := unknownVerifier{}.Verify(in)
		if got.Status != "invalid" || got.TrustScore != 0 ||
			got.Summary != "No input provided." {
			t.Errorf("Verify(%q) = %+v, want invalid/0/No input provided.", in, got)
		}
		assertEvidenceLabels(t, got, []string{"Input Provided"})
	}
}
