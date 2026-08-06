// Package classifier implements a deterministic input-type classifier.
//
// It is the single entry point for every future verifier: given an arbitrary
// user string it returns the best-matching InputType. Detection uses only the
// Go standard library (net, net/mail, net/url, regexp, strings) and is
// intentionally heuristic and stateless so it can be reused anywhere.
package classifier

import (
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
)

// InputType describes the category a user input was classified as.
type InputType string

const (
	TypeURL     InputType = "url"
	TypeDomain  InputType = "domain"
	TypeEmail   InputType = "email"
	TypePhone   InputType = "phone"
	TypeIPv4    InputType = "ipv4"
	TypeIPv6    InputType = "ipv6"
	TypeCompany InputType = "company"
	TypeUnknown InputType = "unknown"
)

var (
	// domainRegex matches dotted DNS names whose final label is a >=2 char TLD
	// (e.g. google.com, bbc.co.uk). It deliberately excludes pure IPs and
	// any input containing a scheme or "@".
	domainRegex = regexp.MustCompile(
		`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`,
	)

	// companyRegex matches a short, free-form proper name made of letters,
	// digits, spaces and a few punctuation characters used in company names
	// (e.g. "OpenAI", "Stripe Inc.", "Acme-Corp"). It must start with a letter
	// so numbers/IP-like tokens are rejected.
	companyRegex = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9\s.'-]{0,63}$`)

	// phoneDigitsRegex strips everything that is not part of an E.164 number
	// (whitespace, dashes, dots, parentheses) prior to matching.
	phoneDigitsRegex = regexp.MustCompile(`[\s.\-()]`)

	// phoneRegex matches E.164-style international phone numbers after
	// surrounding whitespace, dashes, dots and parentheses are removed: a
	// leading "+" followed by 6-15 digits.
	phoneRegex = regexp.MustCompile(`^\+\d{6,15}$`)
)

// Detect classifies a single user input string into an InputType.
//
// The checks run in an order chosen so that more specific formats are matched
// before less specific ones, and so that a value can never be misclassified as
// a less precise type (for example a URL is never reported as a domain).
//
// Order of precedence:
//  1. URL      -> has an http/https scheme and a host
//  2. Email    -> parses as a single RFC 5322 address containing "@"
//  3. IPv4/IPv6-> parses as an IP address (":" => IPv6, else IPv4)
//  4. Phone    -> international number after normalization
//  5. Domain   -> dotted name with a real TLD
//  6. Company  -> short alphabetic proper name
//  7. Unknown  -> everything else
func Detect(input string) InputType {
	in := strings.TrimSpace(input)
	if in == "" {
		return TypeUnknown
	}

	// 1. URL: a real scheme + host means this is a URL, never a bare domain.
	if u, err := url.Parse(in); err == nil && u.Scheme != "" && u.Host != "" {
		if u.Scheme == "http" || u.Scheme == "https" {
			return TypeURL
		}
	}

	// 2. Email: net/mail is stricter than a naive regex here.
	if !strings.ContainsAny(in, " \t") {
		if addr, err := mail.ParseAddress(in); err == nil && strings.Contains(addr.Address, "@") {
			return TypeEmail
		}
	}

	// 3. IPv4 / IPv6 via the stdlib IP parser.
	if ip := net.ParseIP(in); ip != nil {
		if strings.Contains(in, ":") {
			return TypeIPv6
		}
		return TypeIPv4
	}

	// 4. Phone: normalize then match E.164.
	normalized := phoneDigitsRegex.ReplaceAllString(in, "")
	if phoneRegex.MatchString(normalized) {
		return TypePhone
	}

	// 5. Domain: dotted name with a real TLD (IPs/emails/phones already handled).
	if domainRegex.MatchString(in) {
		return TypeDomain
	}

	// 6. Company: a short, letter-leading proper name.
	if companyRegex.MatchString(in) {
		return TypeCompany
	}

	// 7. Fallback.
	return TypeUnknown
}
