// Package model defines the shared data types of the explainable-AI result.
//
// It is a leaf package imported by every analysis module (claims, warnings,
// interpretations, recommendations, reasoning) and by the analysis pipeline,
// so no module depends on another and new modules can be added without touching
// the existing ones.
package model

// Verdict is the coarse human-facing trust band derived from the numeric score.
type Verdict string

const (
	VerdictHigh   Verdict = "High"
	VerdictMedium Verdict = "Medium"
	VerdictLow    Verdict = "Low"
)

// VerdictFromScore maps a 0-100 trust score to a High/Medium/Low verdict.
//
//	70-100 -> High
//	40-69  -> Medium
//	0-39   -> Low
func VerdictFromScore(score int) Verdict {
	switch {
	case score >= 70:
		return VerdictHigh
	case score >= 40:
		return VerdictMedium
	default:
		return VerdictLow
	}
}

// Entity is a named thing (person, organization, location, ...) extracted from
// the input. Kind is one of the EntityKind constants.
type Entity struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// Entity kinds emitted by the claims extractor.
const (
	EntityOrganization = "organization"
	EntityLocation     = "location"
	EntityPerson       = "person"
	EntityDate         = "date"
)

// Interpretation is one plausible reading of an input. An analysis always
// returns 2-3 interpretations so a single meaning is never assumed. Confidence
// is 0-100 and reasoning explains why that reading is plausible.
type Interpretation struct {
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
	Confidence  int    `json:"confidence"`
	Reasoning   string `json:"reasoning"`
}

// WarningSignal is a structured misinformation indicator. Severity is one of
// "high", "medium", or "low".
type WarningSignal struct {
	Label       string `json:"label"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// Recommendation is an actionable next step for the user, with a short reason.
type Recommendation struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ReasoningStep is one stage of the analysis timeline. The timeline exposes
// how the assessment was reached — claim detected, evidence gathered, conflicts
// identified, risk signals, reasoning, final assessment — so the user reviews
// an investigation rather than an unexplained score. Details is a concise,
// user-facing list; it never contains chain-of-thought or internal reasoning.
type ReasoningStep struct {
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Details []string `json:"details"`
}

// EvidenceItem is one scored check, bucketed into a supporting or
// contradicting section. Note carries a plain-English explanation of the check.
type EvidenceItem struct {
	Label  string `json:"label"`
	Result string `json:"result"`
	Points int    `json:"points"`
	Note   string `json:"note,omitempty"`
}

// Result is the complete explainable analysis for one verified input.
//
// The Input/Type/Status/TrustScore/Summary fields mirror the legacy /verify
// response so older clients keep working; every field after them is the
// explainable-AI extension.
type Result struct {
	Input      string `json:"input"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	TrustScore int    `json:"trustScore"`
	Summary    string `json:"summary"`

	// Verdict is the coarse High/Medium/Low band of TrustScore.
	Verdict Verdict `json:"verdict"`

	// KeyClaim is the single sentence under review. For free-form text it is
	// the normalized input sentence; for structured identifiers it is the
	// trustworthiness claim being tested.
	KeyClaim string `json:"keyClaim"`

	// Entities and Keywords are extracted from the input by the claims module.
	Entities []Entity `json:"entities"`
	Keywords []string `json:"keywords"`

	// EvidenceFor / EvidenceAgainst split the scored checks by direction so
	// the user sees what supports and what contradicts the claim.
	EvidenceFor     []EvidenceItem `json:"evidenceFor"`
	EvidenceAgainst []EvidenceItem `json:"evidenceAgainst"`

	// MissingEvidence lists checks that could not be run or for which no
	// evidence was available. These are explicit statements, never fabricated.
	MissingEvidence []string `json:"missingEvidence"`

	// UnknownInformation lists facts that are genuinely unknown (author,
	// publication date, provenance, ...). Also explicit and never invented.
	UnknownInformation []string `json:"unknownInformation"`

	// Interpretations are 2-3 plausible readings of the input.
	Interpretations []Interpretation `json:"interpretations"`

	// WarningSignals are the structured misinformation indicators detected.
	WarningSignals []WarningSignal `json:"warningSignals"`

	// Confidence is 0-100 and reflects how confident the analysis is in its
	// own assessment (not the trust score itself).
	Confidence int `json:"confidence"`

	// Reasoning is the ordered bullet explanation of why the score is what it
	// is. Positive bullets are prefixed with "+", negative with "-".
	Reasoning []string `json:"reasoning"`

	// Timeline is the step-by-step reasoning timeline shown at the top of the
	// analysis. Each step has a one-line summary and expandable details.
	Timeline []ReasoningStep `json:"timeline"`

	// Recommendations are the next steps the user should take.
	Recommendations []Recommendation `json:"recommendations"`
}
