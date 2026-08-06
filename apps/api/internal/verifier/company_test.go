package verifier

import "testing"

// assertCompanyResult checks a company result is well-formed and internally
// consistent with the scoring bands (>=80 verified, 50-79 warning, <50
// invalid). It is used for cases whose outcome depends on live DNS.
func assertCompanyResult(t *testing.T, v Result) {
	t.Helper()
	if v.TrustScore < 0 || v.TrustScore > 100 {
		t.Errorf("trustScore %d out of [0,100]", v.TrustScore)
	}
	switch v.Status {
	case "verified":
		if v.TrustScore < 80 {
			t.Errorf("verified status with score %d (<80): %+v", v.TrustScore, v)
		}
	case "warning":
		if v.TrustScore < 50 || v.TrustScore > 79 {
			t.Errorf("warning status with score %d (want 50-79): %+v", v.TrustScore, v)
		}
	case "invalid":
		if v.TrustScore >= 50 {
			t.Errorf("invalid status with score %d (>=50): %+v", v.TrustScore, v)
		}
	default:
		t.Errorf("unexpected status %q (%+v)", v.Status, v)
	}
}

func TestCompanyVerifier_KnownCompanies(t *testing.T) {
	for _, in := range []string{"Google", "OpenAI", "Microsoft", "Amazon", "Flutterwave", "Paystack"} {
		v := companyVerifier{}.Verify(in)
		t.Logf("company(%q) -> %+v", in, v)
		assertCompanyResult(t, v)
	}
}

func TestCompanyVerifier_LegalSuffixes(t *testing.T) {
	for _, in := range []string{"Google LLC", "Microsoft Corporation", "OpenAI Inc"} {
		v := companyVerifier{}.Verify(in)
		t.Logf("company(%q) -> %+v", in, v)
		assertCompanyResult(t, v)
	}
}

func TestCompanyVerifier_Whitespace(t *testing.T) {
	v := companyVerifier{}.Verify("   Google   ")
	t.Logf("company(%q) -> %+v", "   Google   ", v)
	assertCompanyResult(t, v)
}

func TestCompanyVerifier_Invalid(t *testing.T) {
	for _, in := range []string{"???", "@@@@", "123456", "!!"} {
		v := companyVerifier{}.Verify(in)
		if v.Status != "invalid" || v.TrustScore != 0 ||
			v.Summary != "Invalid company name." {
			t.Errorf("Verify(%q) = %+v, want invalid/0/Invalid company name.", in, v)
		}
	}
}

func TestCompanyVerifier_Empty(t *testing.T) {
	for _, in := range []string{"", "   ", "\t\n"} {
		v := companyVerifier{}.Verify(in)
		if v.Status != "invalid" || v.TrustScore != 0 ||
			v.Summary != "Invalid company name." {
			t.Errorf("Verify(%q) = %+v, want invalid/0/Invalid company name.", in, v)
		}
	}
}
