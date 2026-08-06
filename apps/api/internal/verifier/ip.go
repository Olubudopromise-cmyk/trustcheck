package verifier

import (
	"net"
	"strings"
)

// ipVerifier implements the Verifier interface for IPv4 and IPv6 addresses using
// only the standard library's net.IP classification methods.
type ipVerifier struct{}

// Verify parses and classifies an IP address, scoring it per the spec.
//
// Detection order follows the spec so the most specific class wins:
// loopback -> private -> link-local -> multicast -> unspecified -> global unicast.
func (ipVerifier) Verify(input string) Result {
	ip := net.ParseIP(strings.TrimSpace(input))
	if ip == nil {
		return Result{
			Status:     "invalid",
			TrustScore: 0,
			Summary:    "Invalid IP address.",
		}
	}

	switch {
	case ip.IsLoopback():
		// 127.0.0.0/8, ::1
		return Result{Status: "local", TrustScore: 100, Summary: "Loopback address."}
	case ip.IsPrivate():
		// 10/8, 172.16/12, 192.168/16, 100.64/10, fc00::/7
		return Result{Status: "private", TrustScore: 90, Summary: "Private network address."}
	case ip.IsLinkLocalUnicast():
		// 169.254/16, fe80::/10
		return Result{Status: "local", TrustScore: 80, Summary: "Link-local address."}
	case ip.IsMulticast():
		// 224.0.0.0/4, ff00::/8
		return Result{Status: "warning", TrustScore: 50, Summary: "Multicast address."}
	case ip.IsUnspecified():
		// 0.0.0.0, ::
		return Result{Status: "invalid", TrustScore: 0, Summary: "Unspecified address."}
	case ip.IsGlobalUnicast():
		// Globally routable unicast
		return Result{Status: "verified", TrustScore: 70, Summary: "Globally routable IP address."}
	default:
		// Reserved/unrecognized ranges (e.g. 240.0.0.0/4).
		return Result{Status: "warning", TrustScore: 50, Summary: "Reserved or unrecognized IP address."}
	}
}
