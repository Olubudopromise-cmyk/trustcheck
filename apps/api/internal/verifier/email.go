package verifier

import (
	"net"
	"net/mail"
	"strings"

	"github.com/pamierin/trustcheck/apps/api/internal/scoring"
)

// disposableDomains is a small, internal blocklist of throwaway email
// providers. Matching is exact (case-insensitive) on the email's domain.
var disposableDomains = map[string]bool{
	"mailinator.com":    true,
	"10minutemail.com":  true,
	"guerrillamail.com": true,
	"tempmail.com":      true,
}

// emailVerifier implements the Verifier interface for email inputs.
type emailVerifier struct{}

// Verify classifies and verifies an email address.
//
// Scoring (Go standard library only):
//  1. Parse with net/mail.ParseAddress; failure => invalid / 0.
//  2. Split into local + domain (domain lower-cased).
//  3. DNS MX lookup; +40 if MX exists, else A/AAAA +20, else unreachable / 15.
//  4. Disposable provider map => -40.
//  5. Reuse VerifyDomain on the email's domain and average with the email score.
//  6. Clamp to [0, 100]; map to verified(80-100) / warning(50-79) / invalid(0-49).
func (emailVerifier) Verify(input string) Result {
	const (
		invalid     = "invalid"
		unreachable = "unreachable"
		verified    = "verified"
		warning     = "warning"
	)

	addr, err := mail.ParseAddress(input)
	if err != nil || strings.TrimSpace(input) != addr.Address || addr.Name != "" {
		b := scoring.New()
		b.Fail("Valid Syntax", 0)
		return Result{Status: invalid, TrustScore: 0, Summary: "Invalid email format.", Evidence: b.Evidence()}
	}

	parts := strings.SplitN(addr.Address, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		b := scoring.New()
		b.Fail("Valid Syntax", 0)
		return Result{Status: invalid, TrustScore: 0, Summary: "Invalid email format.", Evidence: b.Evidence()}
	}
	domain := strings.ToLower(parts[1])

	// 3-5. Accumulate mail + disposable + web-presence scores with the shared
	// Builder, then average with the domain score. The Builder clamps the sum;
	// dividing afterwards yields the same result as the historical clamp-after
	// average because the sum never leaves [0, 100] for reachable emails.
	score := scoring.New()
	score.Pass("Valid Syntax", 0)

	// 3. DNS MX (and A/AAAA) lookup.
	if mxs, err := net.LookupMX(domain); err == nil && len(mxs) > 0 {
		score.Pass("MX Record", scoring.MXBonus)
	} else if ips, err := net.LookupHost(domain); err == nil && len(ips) > 0 {
		score.Pass("A/AAAA Record", scoring.ARecordBonus)
	} else {
		score.Fail("Mail Domain", 0)
		return Result{
			Status:     unreachable,
			TrustScore: scoring.UnreachableScore,
			Summary:    "Email domain cannot receive mail.",
			Evidence:   score.Evidence(),
		}
	}

	// 4. Disposable provider penalty.
	if disposableDomains[domain] {
		score.Fail("Disposable Provider", -scoring.DisposablePenalty)
	}

	// 5. Reuse the domain engine for the web-presence score.
	score.Pass("Website Verification", VerifyDomain(domain).TrustScore)
	total := score.Score() / 2

	status, summary := interpretEmail(total)
	if disposableDomains[domain] {
		summary = "Disposable email provider detected."
	}
	return Result{Status: status, TrustScore: total, Summary: summary, Evidence: score.Evidence()}
}

// interpretEmail maps a final score to a status + human summary.
func interpretEmail(score int) (status, summary string) {
	switch {
	case score >= scoring.HighConfidenceScore:
		return "verified", "Email domain receives mail and has a valid web presence."
	case score >= scoring.WarningThreshold:
		return "warning", "Email domain receives mail; web presence partially verified."
	default:
		return "invalid", "Email domain receives mail but failed verification."
	}
}
