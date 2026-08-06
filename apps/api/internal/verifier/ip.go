package verifier

import (
	"net"
	"strings"

	"github.com/pamierin/trustcheck/apps/api/internal/scoring"
)

// ipVerifier implements the Verifier interface for IPv4 and IPv6 addresses using
// only the standard library's net.IP classification methods.
type ipVerifier struct{}

// Verify parses and classifies an IP address, scoring it per the spec.
//
// Detection order follows the spec so the most specific class wins:
// loopback -> private -> link-local -> multicast -> unspecified -> global unicast.
// Each class contributes a single scored check through the shared Builder, so
// evidence and score stay in sync.
func (ipVerifier) Verify(input string) Result {
	ip := net.ParseIP(strings.TrimSpace(input))
	if ip == nil {
		b := scoring.New()
		b.Fail("Valid IP Address", 0)
		return Result{
			Status:     "invalid",
			TrustScore: 0,
			Summary:    "Invalid IP address.",
			Evidence:   b.Evidence(),
		}
	}

	switch {
	case ip.IsLoopback():
		// 127.0.0.0/8, ::1
		b := scoring.New()
		b.Pass("Loopback", scoring.LoopbackBonus)
		return Result{Status: "local", TrustScore: b.Score(), Summary: "Loopback address.", Evidence: b.Evidence()}
	case ip.IsPrivate():
		// 10/8, 172.16/12, 192.168/16, 100.64/10, fc00::/7
		b := scoring.New()
		b.Pass("Private Network", scoring.PrivateIPBonus)
		return Result{Status: "private", TrustScore: b.Score(), Summary: "Private network address.", Evidence: b.Evidence()}
	case ip.IsLinkLocalUnicast():
		// 169.254/16, fe80::/10
		b := scoring.New()
		b.Pass("Link-Local", scoring.LinkLocalBonus)
		return Result{Status: "local", TrustScore: b.Score(), Summary: "Link-local address.", Evidence: b.Evidence()}
	case ip.IsMulticast():
		// 224.0.0.0/4, ff00::/8
		b := scoring.New()
		b.Warning("Multicast", scoring.MulticastScore)
		return Result{Status: "warning", TrustScore: b.Score(), Summary: "Multicast address.", Evidence: b.Evidence()}
	case ip.IsUnspecified():
		// 0.0.0.0, ::
		b := scoring.New()
		b.Fail("Unspecified Address", 0)
		return Result{Status: "invalid", TrustScore: 0, Summary: "Unspecified address.", Evidence: b.Evidence()}
	case ip.IsGlobalUnicast():
		// Globally routable unicast
		b := scoring.New()
		b.Pass("Global Unicast", scoring.GlobalUnicastBonus)
		return Result{Status: "verified", TrustScore: b.Score(), Summary: "Globally routable IP address.", Evidence: b.Evidence()}
	default:
		// Reserved/unrecognized ranges (e.g. 240.0.0.0/4).
		b := scoring.New()
		b.Warning("Reserved Range", scoring.MulticastScore)
		return Result{Status: "warning", TrustScore: b.Score(), Summary: "Reserved or unrecognized IP address.", Evidence: b.Evidence()}
	}
}
