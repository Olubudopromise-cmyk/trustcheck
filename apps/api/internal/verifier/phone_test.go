package verifier

import "testing"

// TestPhoneVerifier_Valid covers every supported country calling code. All of
// these are deterministic: no network involved.
func TestPhoneVerifier_Valid(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"nigeria", "+2348012345678", "Phone number format is valid (Nigeria)."},
		{"usa", "+14155552671", "Phone number format is valid (USA/Canada)."},
		{"uk", "+447700900123", "Phone number format is valid (United Kingdom)."},
		{"india", "+919876543210", "Phone number format is valid (India)."},
		{"japan", "+819012345678", "Phone number format is valid (Japan)."},
		{"australia", "+61412345678", "Phone number format is valid (Australia)."},
		{"germany", "+4915123456789", "Phone number format is valid (Germany)."},
		{"france", "+33123456789", "Phone number format is valid (France)."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := phoneVerifier{}.Verify(tc.in)
			if v.Status != "verified" || v.TrustScore != 80 || v.Summary != tc.want {
				t.Errorf("Verify(%q) = %+v, want verified/80/%q", tc.in, v, tc.want)
			}
		})
	}
}

// TestPhoneVerifier_Normalization proves the formatting characters are
// stripped before validation, exactly as the classifier does.
func TestPhoneVerifier_Normalization(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"spaces and dashes", "+234 801-234-5678", "Phone number format is valid (Nigeria)."},
		{"parentheses", "+1 (415) 555-2671", "Phone number format is valid (USA/Canada)."},
		{"dots", "+44.7700.900123", "Phone number format is valid (United Kingdom)."},
		{"mixed formatting", "(+91) 98765-43210", "Phone number format is valid (India)."},
		{"surrounding whitespace", "  +49 151 2345678  ", "Phone number format is valid (Germany)."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := phoneVerifier{}.Verify(tc.in)
			if v.Status != "verified" || v.TrustScore != 80 || v.Summary != tc.want {
				t.Errorf("Verify(%q) = %+v, want verified/80/%q", tc.in, v, tc.want)
			}
		})
	}
}

// TestPhoneVerifier_Invalid covers inputs that fail the E.164 format rule.
func TestPhoneVerifier_Invalid(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"missing plus", "14155552671"},
		{"too short", "+1234"},
		{"too long", "+12345678901234567890"},
		{"letters", "+234abc1234567"},
		{"empty", ""},
		{"plus only", "+"},
		{"spaces leave it too short", "+12 34"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := phoneVerifier{}.Verify(tc.in)
			if v.Status != "invalid" || v.TrustScore != 0 || v.Summary != "Invalid phone number." {
				t.Errorf("Verify(%q) = %+v, want invalid/0/Invalid phone number.", tc.in, v)
			}
		})
	}
}

// TestPhoneVerifier_UnknownCountry covers well-formed numbers whose calling
// code is not in the supported table: they stay valid but score lower.
func TestPhoneVerifier_UnknownCountry(t *testing.T) {
	for _, in := range []string{"+999123456789", "+297123456", "+71234567890"} {
		v := phoneVerifier{}.Verify(in)
		if v.Status != "warning" || v.TrustScore != 60 ||
			v.Summary != "Phone number format is valid but country is unknown." {
			t.Errorf("Verify(%q) = %+v, want warning/60/unknown summary", in, v)
		}
	}
}
