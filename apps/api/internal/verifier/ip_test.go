package verifier

import "testing"

func TestIPVerifier_IPv4(t *testing.T) {
	cases := []struct {
		in   string
		want Result
	}{
		{"8.8.8.8", Result{"verified", 70, "Globally routable IP address."}},
		{"127.0.0.1", Result{"local", 100, "Loopback address."}},
		{"192.168.1.10", Result{"private", 90, "Private network address."}},
		{"10.0.0.1", Result{"private", 90, "Private network address."}},
		{"172.16.0.5", Result{"private", 90, "Private network address."}},
		{"169.254.1.1", Result{"local", 80, "Link-local address."}},
		{"224.0.0.1", Result{"warning", 50, "Multicast address."}},
		{"0.0.0.0", Result{"invalid", 0, "Unspecified address."}},
	}
	for _, tc := range cases {
		got := ipVerifier{}.Verify(tc.in)
		if got != tc.want {
			t.Errorf("Verify(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestIPVerifier_IPv6(t *testing.T) {
	cases := []struct {
		in   string
		want Result
	}{
		{"::1", Result{"local", 100, "Loopback address."}},
		{"::", Result{"invalid", 0, "Unspecified address."}},
		{"fe80::1", Result{"local", 80, "Link-local address."}},
		{"fc00::1", Result{"private", 90, "Private network address."}},
		{"2606:4700:4700::1111", Result{"verified", 70, "Globally routable IP address."}},
	}
	for _, tc := range cases {
		got := ipVerifier{}.Verify(tc.in)
		if got != tc.want {
			t.Errorf("Verify(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestIPVerifier_Malformed(t *testing.T) {
	for _, in := range []string{"abc", "999.999.999.999", "not-an-ip", "", "256.256.256.256"} {
		got := ipVerifier{}.Verify(in)
		if got.Status != "invalid" || got.TrustScore != 0 || got.Summary != "Invalid IP address." {
			t.Errorf("Verify(%q) = %+v, want invalid/0/Invalid IP address.", in, got)
		}
	}
}
