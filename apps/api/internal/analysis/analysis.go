// Package analysis defines the explainable-AI analysis pipeline for every
// TrustCheck verification.
//
// A verification is no longer just a score. The analysis pipeline turns the
// raw verifier output into a structured, explainable result: the main claim,
// the entities and keywords it mentions, evidence split into supporting and
// contradicting sections, alternative interpretations, warning signals,
// an explanation of the score, and next-step recommendations.
//
// The result model lives in internal/model so every module can share it
// without depending on one another. Extension modules plug into the Analyzer
// through the Module interface (see pipeline.go).
package analysis

import (
	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

// VerdictFromScore maps a 0-100 trust score to a High/Medium/Low verdict.
// It re-exports the model helper for callers that build results directly.
func VerdictFromScore(score int) model.Verdict { return model.VerdictFromScore(score) }

// ConfidenceThresholds drive how confident the analysis is in itself.
const (
	// ConfidenceHigh is used when a rich set of evidence was collected.
	ConfidenceHigh = 90
	// ConfidenceMedium is used when some evidence was collected.
	ConfidenceMedium = 72
	// ConfidenceLow is used when almost no evidence could be gathered.
	ConfidenceLow = 45
)

// confidenceOf derives a base confidence (0-100) from the amount of evidence
// collected. More evidence means a firmer assessment; an unclassifiable input
// lowers confidence because the system knows less about it.
func confidenceOf(evidenceFor, evidenceAgainst []model.EvidenceItem, inputType classifier.InputType) int {
	total := len(evidenceFor) + len(evidenceAgainst)
	confidence := ConfidenceLow
	switch {
	case total >= 4:
		confidence = ConfidenceHigh
	case total >= 2:
		confidence = ConfidenceMedium
	}

	if inputType == classifier.TypeUnknown {
		confidence -= 15
	}
	if confidence < 0 {
		return 0
	}
	return confidence
}
