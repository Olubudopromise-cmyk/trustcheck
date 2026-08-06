package verifier

import (
	"net"
	"net/mail"
	"strings"
)

// disposableDomains is a small, internal blocklist of throwaway email
// providers. Matching is exact (case-insensitive) on the email's domain.
var disposableDomains = map[string]bool{
	"mailinator.com":     true,
	"10minutemail.com":   true,
	"guerrillamail.com":  true,
	"tempmail.com":       true,
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
		invalid        = "invalid"
		unreachable    = "unreachable"
		verified       = "verified"
		warning        = "warning"
	)

	addr, err := mail.ParseAddress(input)
	if err != nil || strings.TrimSpace(input) != addr.Address || addr.Name != "" {
		return Result{Status: invalid, TrustScore: 0, Summary: "Invalid email format."}
	}

	parts := strings.SplitN(addr.Address, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Result{Status: invalid, TrustScore: 0, Summary: "Invalid email format."}
	}
	domain := strings.ToLower(parts[1])

	// 3. DNS MX (and A/AAAA) lookup.
	emailScore, reachable := scoreDomainMail(domain)
	if !reachable {
		return Result{
			Status:     unreachable,
			TrustScore: 15,
			Summary:    "Email domain cannot receive mail.",
		}
	}

	// 4. Disposable provider penalty.
	if disposableDomains[domain] {
		emailScore -= 40
	}

	// 5. Reuse the domain engine for the web-presence score.
	domainScore := VerifyDomain(domain).TrustScore

	score := (domainScore + emailScore) / 2
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	status, summary := interpretEmail(score)
	if disposableDomains[domain] {
		summary = "Disposable email provider detected."
	}
	return Result{Status: status, TrustScore: score, Summary: summary}
}

// scoreDomainMail returns the mail-delivery score and whether the domain is
// mail-reachable (has MX or A/AAAA records).
func scoreDomainMail(domain string) (score int, reachable bool) {
	if mxs, err := net.LookupMX(domain); err == nil && len(mxs) > 0 {
		return 40, true
	}
	if ips, err := net.LookupHost(domain); err == nil && len(ips) > 0 {
		return 20, true
	}
	return 0, false
}

// interpretEmail maps a final score to a status + human summary.
func interpretEmail(score int) (status, summary string) {
	switch {
	case score >= 80:
		return "verified", "Email domain receives mail and has a valid web presence."
	case score >= 50:
		return "warning", "Email domain receives mail; web presence partially verified."
	default:
		return "invalid", "Email domain receives mail but failed verification."
	}
}
