// Package interpretations generates 2-3 plausible readings of an input so a
// single meaning is never assumed. Every interpretation carries an explanation,
// a confidence (0-100), and the reasoning behind it.
//
// Free-form text gets a literal reading, a satire/exaggeration reading, and an
// alternative-context reading. Structured identifiers get a legitimate reading,
// a lookalike/impersonation reading, and an origin-unknown reading.
package interpretations

import (
	"strings"

	"github.com/pamierin/trustcheck/apps/api/internal/claims"
	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

// Context carries the facts the generator needs to reason about plausibility.
// It is decoupled from the verifier packages so future evidence sources can be
// added without changing the generator contract.
type Context struct {
	Input           string
	InputType       classifier.InputType
	Claim           claims.Claim
	Status          string
	Warnings        []model.WarningSignal
	EvidenceFor     []model.EvidenceItem
	EvidenceAgainst []model.EvidenceItem
}

// supported indicates whether the verification produced net-positive evidence.
func supported(c Context) bool {
	return len(c.EvidenceFor) > len(c.EvidenceAgainst)
}

// undermined indicates whether there is material contradicting evidence.
func undermined(c Context) bool {
	return len(c.EvidenceAgainst) > len(c.EvidenceFor)
}

// highSeverityWarnings reports whether any high-severity misinformation signal
// was detected (e.g. no citations, manipulated statistics).
func highSeverityWarnings(c Context) bool {
	for _, w := range c.Warnings {
		if w.Severity == "high" {
			return true
		}
	}
	return false
}

// Generate returns 2-3 interpretations of the input, most plausible first.
func Generate(c Context) []model.Interpretation {
	if c.InputType == classifier.TypeUnknown {
		return forText(c)
	}
	return forIdentifier(c)
}

// forText builds the literal, satire, and alternative-context readings of a
// sentence. The literal reading is most plausible when verification succeeded;
// otherwise the sceptical readings move up.
func forText(c Context) []model.Interpretation {
	claim := firstSentence(c.Claim.MainClaim, c.Input)

	literal := model.Interpretation{
		Title:                   "Literal reading",
		Explanation:             "The text means exactly what it says: " + claim,
		Confidence:              70,
		Reasoning:               "Read at face value, the sentence is grammatically direct and makes a specific assertion.",
		SupportingEvidenceCount: len(c.EvidenceFor),
	}
	satire := model.Interpretation{
		Title:                   "Satire or exaggeration",
		Explanation:             "The claim may be satire, parody, or hyperbole rather than a factual report.",
		Confidence:              40,
		Reasoning:               "Sensational or surprising claims often originate from satirical outlets or are exaggerated for engagement.",
		SupportingEvidenceCount: len(c.EvidenceAgainst),
	}
	alternative := model.Interpretation{
		Title:                   "Alternative context",
		Explanation:             "The headline may be about something else entirely — an unrelated event, an analogy, or an incomplete context.",
		Confidence:              35,
		Reasoning:               "Headlines routinely condense or reframe events; without the full source the intended referent is unclear.",
		SupportingEvidenceCount: 0,
	}

	if highSeverityWarnings(c) {
		literal.Confidence = 30
		literal.Reasoning = "The wording carries strong misinformation signals (e.g. no citations), so the literal reading is less plausible."
		satire.Confidence = 55
		satire.Reasoning = "Strong misinformation signals suggest the claim may be exaggerated or satirical."
		alternative.Confidence = 45
		alternative.Reasoning = "The wording may be an exaggerated retelling of an unrelated, more mundane event."
		return []model.Interpretation{satire, alternative, literal}
	}
	return []model.Interpretation{literal, satire, alternative}
}

// forIdentifier builds the legitimate, lookalike, and origin-unknown readings
// for structured inputs (domains, emails, phones, companies, URLs, IPs).
func forIdentifier(c Context) []model.Interpretation {
	legit := model.Interpretation{
		Title:                   "Genuine identifier",
		Explanation:             "The " + string(c.InputType) + " belongs to the organization or person it claims to represent.",
		Confidence:              80,
		Reasoning:               "The identifier passes the expected format and infrastructure checks.",
		SupportingEvidenceCount: len(c.EvidenceFor),
	}
	lookalike := model.Interpretation{
		Title:                   "Lookalike or impersonation",
		Explanation:             "The " + string(c.InputType) + " may imitate a well-known name to deceive (typosquatting, spoofing, or a similar-sounding entity).",
		Confidence:              45,
		Reasoning:               "Identifiers that resemble famous names without belonging to them are a common phishing vector.",
		SupportingEvidenceCount: len(c.EvidenceAgainst),
	}
	unknown := model.Interpretation{
		Title:                   "Origin unknown",
		Explanation:             "Not enough is known about the " + string(c.InputType) + " to judge who controls it.",
		Confidence:              50,
		Reasoning:               "Ownership and provenance of the identifier were not confirmed by the available checks.",
		SupportingEvidenceCount: 0,
	}

	if undermined(c) {
		legit.Confidence = 25
		legit.Reasoning = "Contradicting evidence makes the genuine reading unlikely."
		lookalike.Confidence = 65
		lookalike.Reasoning = "Failed infrastructure checks are consistent with impersonation or misuse."
		unknown.Confidence = 55
		return []model.Interpretation{lookalike, unknown, legit}
	}
	if !supported(c) {
		legit.Confidence = 50
		legit.Reasoning = "The identifier is plausible but its legitimacy was not confirmed by the checks."
	}
	return []model.Interpretation{legit, lookalike, unknown}
}

// firstSentence returns the first sentence of the claim text, falling back to
// the raw input for single-line claims.
func firstSentence(claim, input string) string {
	if claim != "" {
		return claim
	}
	first := strings.TrimSpace(strings.SplitN(input, ".", 2)[0])
	if first != "" {
		return first + "."
	}
	return "an unknown claim"
}
