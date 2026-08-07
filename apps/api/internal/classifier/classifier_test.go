package classifier

import "testing"

func TestDetect(t *testing.T) {
	cases := []struct {
		in  string
		out InputType
	}{
		// URL
		{"https://openai.com", TypeURL},
		{"http://google.com", TypeURL},
		{"https://github.com/openai", TypeURL},
		// Domain (never a URL when no scheme)
		{"google.com", TypeDomain},
		{"openai.com", TypeDomain},
		{"bbc.co.uk", TypeDomain},
		// Email
		{"support@stripe.com", TypeEmail},
		{"admin@example.org", TypeEmail},
		// Phone (international)
		{"+2348012345678", TypePhone},
		{"+15551234567", TypePhone},
		{"+442071838750", TypePhone},
		{"+1 555-123-4567", TypePhone},
		// IPv4 / IPv6
		{"8.8.8.8", TypeIPv4},
		{"1.1.1.1", TypeIPv4},
		{"2606:4700:4700::1111", TypeIPv6},
		// Company
		{"OpenAI", TypeCompany},
		{"Stripe Inc", TypeCompany},
		{"Acme Corporation", TypeCompany},
		{"Microsoft", TypeCompany},
		{"Amazon", TypeCompany},
		// Unknown
		{"???", TypeUnknown},
		{"", TypeUnknown},
		{"   ", TypeUnknown},
		{"1.2.3", TypeUnknown},
		// Free-form sentences are unknown (not company names), so the
		// explainable text analysis can run on them.
		{"NASA confirms aliens landed in Lagos yesterday.", TypeUnknown},
		{"Aliens landed in Lagos yesterday", TypeUnknown},
		{"The company announced a major security breach today", TypeUnknown},
	}
	for _, c := range cases {
		got := Detect(c.in)
		if got != c.out {
			t.Errorf("Detect(%q) = %q, want %q", c.in, got, c.out)
		}
	}
}
