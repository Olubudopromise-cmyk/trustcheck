package verifier

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/pamierin/trustcheck/apps/api/internal/scoring"
)

// nonAlnumRegex strips everything but lowercase letters and digits when
// deriving a website candidate from a company name.
var nonAlnumRegex = regexp.MustCompile(`[^a-z0-9]`)

// legalSuffixes are common legal/structural tokens whose presence raises the
// confidence that the input is a real company. They are never kept when
// deriving the official-domain candidate.
var legalSuffixes = map[string]bool{
	"inc": true, "ltd": true, "llc": true, "plc": true, "corp": true,
	"corporation": true, "limited": true, "company": true, "group": true,
	"technologies": true, "tech": true, "systems": true, "international": true,
	"holdings": true,
}

// companyVerifier implements the Verifier interface for company inputs.
type companyVerifier struct{}

// Verify validates and scores a company name.
//
// Steps (Go standard library only):
//  1. Normalize: trim whitespace and collapse duplicate spaces; empty => invalid / 0.
//  2. Reject obvious garbage (no letters at all, e.g. "???", "@@@@", "123456")
//     => invalid / 0.
//  3. Detect a legal suffix; presence adds +10 confidence.
//  4. Infer the official domain: lowercase, strip punctuation and legal
//     suffixes, append ".com".
//  5. Reuse VerifyDomain on the inferred domain (no DNS logic duplicated).
//  6. Score: 40 base + 10 suffix + 20 domain resolves + 20 verified HTTPS,
//     capped at 100. >=80 verified, 50-79 warning, below invalid.
func (companyVerifier) Verify(input string) Result {
	// 1. Normalize.
	name := strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
	if name == "" {
		b := scoring.New()
		b.Fail("Valid Company Name", 0)
		return Result{Status: "invalid", TrustScore: 0, Summary: "Invalid company name.", Evidence: b.Evidence()}
	}

	// 2. Obvious garbage.
	if isObviousGarbage(name) {
		b := scoring.New()
		b.Fail("Valid Company Name", 0)
		return Result{Status: "invalid", TrustScore: 0, Summary: "Invalid company name.", Evidence: b.Evidence()}
	}

	// 3. Legal suffix confidence bonus.
	score := scoring.New()
	score.Pass("Company Name", scoring.CompanyBaseScore)
	if hasLegalSuffix(name) {
		score.Pass("Legal Suffix", scoring.CompanySuffixBonus)
	}

	// 4-5. Infer and verify the official website candidate.
	if domain := inferCompanyDomain(name); domain != "" {
		score.Info("Website Inferred")
		r := VerifyDomain(domain)
		if r.Status != "unreachable" {
			score.Pass("Website Resolves", scoring.ResolveBonus) // domain resolves
		}
		if r.Status == "verified" {
			score.Pass("Website Verified", scoring.HTTPSBonus) // verified HTTPS
		}
	}

	// 6. Interpret (the Builder already clamped to [0, 100]).
	status, summary := interpretCompany(score.Score())
	return Result{Status: status, TrustScore: score.Score(), Summary: summary, Evidence: score.Evidence()}
}

// isObviousGarbage reports whether a name contains no letters at all, i.e.
// punctuation, digit or underscore sequences ("???", "@@@@", "123456", "___").
func isObviousGarbage(name string) bool {
	for _, r := range name {
		if unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// hasLegalSuffix reports whether any word in the name is a known legal suffix.
func hasLegalSuffix(name string) bool {
	for _, w := range strings.Fields(strings.ToLower(name)) {
		if legalSuffixes[w] {
			return true
		}
	}
	return false
}

// inferCompanyDomain derives the official website candidate: lowercased, all
// punctuation removed, legal suffixes dropped, ".com" appended. It returns ""
// when no usable word remains.
func inferCompanyDomain(name string) string {
	var kept []string
	for _, w := range strings.Fields(strings.ToLower(name)) {
		w = nonAlnumRegex.ReplaceAllString(w, "")
		if w != "" && !legalSuffixes[w] {
			kept = append(kept, w)
		}
	}
	if len(kept) == 0 {
		return ""
	}
	return strings.Join(kept, "") + ".com"
}

// interpretCompany maps the final score to a status + human summary.
func interpretCompany(score int) (status, summary string) {
	switch {
	case score >= scoring.HighConfidenceScore:
		return "verified", "Recognized company with an active website."
	case score >= scoring.WarningThreshold:
		return "warning", "Company name appears valid but could not be fully verified."
	default:
		return "invalid", "Unable to verify company."
	}
}
