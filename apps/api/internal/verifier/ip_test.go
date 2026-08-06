package verifier

import "testing"

func TestIPVerifier_IPv4(t *testing.T) {
	cases := []struct {
		in         string
		want       Result
		wantLabels []string
	}{
		{"8.8.8.8", Result{Status: "verified", TrustScore: 70, Summary: "Globally routable IP address."}, []string{"Global Unicast"}},
		{"127.0.0.1", Result{Status: "local", TrustScore: 100, Summary: "Loopback address."}, []string{"Loopback"}},
		{"192.168.1.10", Result{Status: "private", TrustScore: 90, Summary: "Private network address."}, []string{"Private Network"}},
		{"10.0.0.1", Result{Status: "private", TrustScore: 90, Summary: "Private network address."}, []string{"Private Network"}},
		{"172.16.0.5", Result{Status: "private", TrustScore: 90, Summary: "Private network address."}, []string{"Private Network"}},
		{"169.254.1.1", Result{Status: "local", TrustScore: 80, Summary: "Link-local address."}, []string{"Link-Local"}},
		{"224.0.0.1", Result{Status: "warning", TrustScore: 50, Summary: "Multicast address."}, []string{"Multicast"}},
		{"0.0.0.0", Result{Status: "invalid", TrustScore: 0, Summary: "Unspecified address."}, []string{"Unspecified Address"}},
	}
	for _, tc := range cases {
		got := ipVerifier{}.Verify(tc.in)
		assertResult(t, got, tc.want)
		assertEvidenceLabels(t, got, tc.wantLabels)
	}
}

func TestIPVerifier_IPv6(t *testing.T) {
	cases := []struct {
		in         string
		want       Result
		wantLabels []string
	}{
		{"::1", Result{Status: "local", TrustScore: 100, Summary: "Loopback address."}, []string{"Loopback"}},
		{"::", Result{Status: "invalid", TrustScore: 0, Summary: "Unspecified address."}, []string{"Unspecified Address"}},
		{"fe80::1", Result{Status: "local", TrustScore: 80, Summary: "Link-local address."}, []string{"Link-Local"}},
		{"fc00::1", Result{Status: "private", TrustScore: 90, Summary: "Private network address."}, []string{"Private Network"}},
		{"2606:4700:4700::1111", Result{Status: "verified", TrustScore: 70, Summary: "Globally routable IP address."}, []string{"Global Unicast"}},
	}
	for _, tc := range cases {
		got := ipVerifier{}.Verify(tc.in)
		assertResult(t, got, tc.want)
		assertEvidenceLabels(t, got, tc.wantLabels)
	}
}

func TestIPVerifier_Malformed(t *testing.T) {
	for _, in := range []string{"abc", "999.999.999.999", "not-an-ip", "", "256.256.256.256"} {
		got := ipVerifier{}.Verify(in)
		if got.Status != "invalid" || got.TrustScore != 0 || got.Summary != "Invalid IP address." {
			t.Errorf("Verify(%q) = %+v, want invalid/0/Invalid IP address.", in, got)
		}
		assertEvidenceLabels(t, got, []string{"Valid IP Address"})
	}
}
