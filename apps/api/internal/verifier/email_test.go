package verifier

import "testing"

// onlineResult asserts that a reachable email returns a sane verified/warning
// result. When the sandbox has no outbound network, DNS fails and the engine
// correctly degrades to unreachable/15, which is also accepted.
func assertReachableOrUnreachable(t *testing.T, v Result) {
	t.Helper()
	switch v.Status {
	case "unreachable":
		if v.TrustScore != 15 || v.Summary != "Email domain cannot receive mail." {
			t.Errorf("unreachable result malformed: %+v", v)
		}
	case "verified", "warning", "invalid":
		// A reachable email can legitimately land in any of the three score
		// bands (verified/warning/invalid), e.g. outlook.com returns 4xx to
		// HEAD -> domainScore 50 -> email score 45 -> invalid per the spec.
		if v.TrustScore < 0 || v.TrustScore > 100 {
			t.Errorf("score %d out of [0,100]", v.TrustScore)
		}
		if v.Summary == "Invalid email format." {
			t.Errorf("reachable email got invalid-format summary")
		}
	default:
		t.Errorf("unexpected status %q (%+v)", v.Status, v)
	}
	if len(v.Evidence) == 0 {
		t.Errorf("result carries no evidence: %+v", v)
	}
}

func TestEmailVerifier_Malformed(t *testing.T) {
	for _, in := range []string{"not-an-email", "", "@domain", "user@", "no at sign"} {
		v := emailVerifier{}.Verify(in)
		if v.Status != "invalid" || v.TrustScore != 0 ||
			v.Summary != "Invalid email format." {
			t.Errorf("Verify(%q) = %+v, want invalid/0/Invalid email format.", in, v)
		}
		assertEvidenceLabels(t, v, []string{"Valid Syntax"})
	}
}

func TestEmailVerifier_NonexistentDomain(t *testing.T) {
	v := emailVerifier{}.Verify("user@nonexistent.invalid")
	if v.Status != "unreachable" || v.TrustScore != 15 ||
		v.Summary != "Email domain cannot receive mail." {
		t.Errorf("got %+v, want unreachable/15/Email domain cannot receive mail.", v)
	}
	assertEvidenceLabels(t, v, []string{"Valid Syntax", "Mail Domain"})
}

func TestEmailVerifier_ValidGmail(t *testing.T) {
	v := emailVerifier{}.Verify("test@gmail.com")
	t.Logf("gmail.com -> %+v", v)
	assertReachableOrUnreachable(t, v)
}

func TestEmailVerifier_ValidOutlook(t *testing.T) {
	v := emailVerifier{}.Verify("test@outlook.com")
	t.Logf("outlook.com -> %+v", v)
	assertReachableOrUnreachable(t, v)
}

func TestEmailVerifier_DisposableProvider(t *testing.T) {
	v := emailVerifier{}.Verify("user@mailinator.com")
	t.Logf("mailinator.com -> %+v", v)
	switch v.Status {
	case "unreachable":
		// offline: mailinator did not resolve in this environment
	case "invalid":
		// online + disposable -> -40 keeps the score below 50 -> invalid
		if v.Summary != "Disposable email provider detected." {
			t.Errorf("summary = %q, want %q", v.Summary, "Disposable email provider detected.")
		}
		if v.TrustScore < 0 || v.TrustScore > 100 {
			t.Errorf("score %d out of [0,100]", v.TrustScore)
		}
	default:
		t.Errorf("unexpected status %q for disposable provider (%+v)", v.Status, v)
	}
}
