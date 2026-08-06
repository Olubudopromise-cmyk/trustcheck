package verifier

import (
	"strings"

	"github.com/pamierin/trustcheck/apps/api/internal/scoring"
)

// unknownVerifier implements the Verifier interface for unclassified inputs.
// Instead of a dead-end "not implemented" result it suggests a likely valid
// input the user may have meant, keeping the result deterministic and local.
type unknownVerifier struct{}

// Verify analyzes unclassified text and suggests a likely valid input.
//
// Rules (Go standard library only, no AI, no network):
//  1. Trim and collapse whitespace; empty => invalid / 0 / "No input provided.".
//  2. Bare domain token (letters/digits/hyphens, no dot) => suggest "<name>.com".
//  3. Input starting with "www" => suggest "<input>.com".
//  4. Malformed http(s) URL => repair the scheme delimiter ("http:/..." ->
//     "http://...").
//  5. Digits only => suggest an E.164 number ("+" + input).
//  6. Single "@" with no TLD after it => suggest "<input>.com".
//  7. Nothing matched => unknown / 10 / "Unable to classify the input.".
//
// The suggestion is embedded in the summary; the Result shape is unchanged.
func (unknownVerifier) Verify(input string) Result {
	// 1. Normalize.
	in := strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
	if in == "" {
		b := scoring.New()
		b.Fail("Input Provided", 0)
		return Result{Status: "invalid", TrustScore: 0, Summary: "No input provided.", Evidence: b.Evidence()}
	}
	low := strings.ToLower(in)

	// 2. Bare domain missing a TLD.
	if isBareDomainToken(low) {
		b := scoring.New()
		b.Pass("Suggestion Generated", scoring.UnknownSuggestionScore)
		return Result{
			Status:     "suggestion",
			TrustScore: b.Score(),
			Summary:    "Did you mean the domain " + in + ".com?",
			Evidence:   b.Evidence(),
		}
	}

	// 3. www-prefixed name missing a TLD.
	if strings.HasPrefix(low, "www") {
		return suggestion(in + ".com")
	}

	// 4. Malformed http(s) URL.
	if fixed, ok := repairURL(in); ok {
		return suggestion(fixed)
	}

	// 5. Digits only.
	if isAllDigits(in) {
		return suggestion("+" + in)
	}

	// 6. Single "@" with no TLD after it.
	if isAddrWithoutTLD(low) {
		return suggestion(in + ".com")
	}

	// 7. Nothing matched.
	b := scoring.New()
	b.Warning("No Suggestion", scoring.NoSuggestionScore)
	return Result{
		Status:     "unknown",
		TrustScore: b.Score(),
		Summary:    scoring.StatusSummary("unknown"),
		Evidence:   b.Evidence(),
	}
}

// isBareDomainToken reports whether the input is a single word made of
// letters, digits and hyphens (with at least one letter) and no dot, i.e. a
// domain missing its TLD.
func isBareDomainToken(low string) bool {
	if low == "" {
		return false
	}
	hasLetter := false
	for _, r := range low {
		switch {
		case r >= 'a' && r <= 'z':
			hasLetter = true
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return hasLetter
}

// isAllDigits reports whether every character is a digit.
func isAllDigits(in string) bool {
	if in == "" {
		return false
	}
	for _, r := range in {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isAddrWithoutTLD reports whether the input has exactly one "@" and the part
// after it contains no dot (i.e. an email missing its TLD).
func isAddrWithoutTLD(low string) bool {
	if strings.Count(low, "@") != 1 {
		return false
	}
	domain := strings.SplitN(low, "@", 2)[1]
	return domain != "" && !strings.Contains(domain, ".")
}

// repairURL repairs a malformed http/https URL by normalizing the scheme
// delimiter ("http:/..." or "http:..." become "http://..."). It returns
// whether a repair was made.
func repairURL(in string) (string, bool) {
	low := strings.ToLower(in)
	var scheme, rest string
	switch {
	case strings.HasPrefix(low, "https"):
		scheme, rest = "https", in[len("https"):]
	case strings.HasPrefix(low, "http"):
		scheme, rest = "http", in[len("http"):]
	default:
		return "", false
	}
	rest = strings.TrimLeft(rest, ":/")
	if rest == "" {
		return "", false
	}
	return scheme + "://" + rest, true
}

// suggestion builds a suggestion result with the suggestion embedded in the
// summary.
func suggestion(text string) Result {
	b := scoring.New()
	b.Pass("Suggestion Generated", scoring.UnknownSuggestionScore)
	return Result{
		Status:     "suggestion",
		TrustScore: b.Score(),
		Summary:    "Did you mean " + text + "?",
		Evidence:   b.Evidence(),
	}
}
