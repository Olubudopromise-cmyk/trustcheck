// Package analysis (see analysis.go) — this file implements the analysis
// pipeline: it runs the modular stages (claims, warnings, interpretations,
// recommendations, reasoning), splits the scored evidence into supporting and
// contradicting sections, and assembles the final explainable model.Result.
package analysis

import (
	"context"
	"fmt"

	"github.com/pamierin/trustcheck/apps/api/internal/claims"
	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/interpretations"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
	"github.com/pamierin/trustcheck/apps/api/internal/perspectives"
	"github.com/pamierin/trustcheck/apps/api/internal/reasoning"
	"github.com/pamierin/trustcheck/apps/api/internal/recommendations"
	"github.com/pamierin/trustcheck/apps/api/internal/scoring"
	"github.com/pamierin/trustcheck/apps/api/internal/verifier"
	"github.com/pamierin/trustcheck/apps/api/internal/warnings"
)

// Module is the pluggable extension point of the analysis pipeline. Future
// capabilities — live web search, fact-check integrations, reverse image
// search, citation verification, source reputation, bias detection, user
// feedback learning — implement this interface and register themselves with the
// Analyzer without changing the pipeline or the response model.
//
// Enrich receives the input and the partially-complete result and may amend it
// in place. Implementations must be idempotent and never fabricate evidence:
// when a module has nothing to add it should return without modifying Result.
type Module interface {
	// Name identifies the module for logging and debugging.
	Name() string
	// Enrich updates result with the module's findings for input.
	Enrich(ctx context.Context, input string, result *model.Result) error
}

// Analyzer runs the core analysis stages and then any registered modules.
type Analyzer struct {
	modules []Module
}

// New returns an Analyzer with the given optional extension modules. Modules
// run after the core pipeline, in registration order.
func New(modules ...Module) *Analyzer {
	return &Analyzer{modules: modules}
}

// Analyze produces the full explainable analysis for an input.
//
// The core pipeline is deterministic and stdlib-only: classify the claim,
// detect warning signals, generate interpretations, recommend next steps, and
// explain the score. Extension modules registered on the Analyzer then run in
// order and may enrich the result.
func (a *Analyzer) Analyze(ctx context.Context, input string, inputType classifier.InputType, vr verifier.Result) model.Result {
	claim := claims.Extract(input, inputType)
	signals := warnings.Detect(input, inputType)
	verdict := VerdictFromScore(vr.TrustScore)

	// The shared evidence classification is the single source of the
	// supporting / contradicting / neutral buckets. Every timeline section and
	// perspective consumes these slices, so counts can never disagree.
	set := scoring.ClassifyEvidence(vr.Evidence)
	evidenceFor := set.Supporting
	evidenceAgainst := set.Contradicting
	neutral := len(set.Neutral)

	interpretations := interpretations.Generate(interpretations.Context{
		Input:           input,
		InputType:       inputType,
		Claim:           claim,
		Status:          model.StatusFromVerdict(verdict),
		Warnings:        signals,
		EvidenceFor:     evidenceFor,
		EvidenceAgainst: evidenceAgainst,
	})

	result := model.Result{
		Input:              input,
		Type:               string(inputType),
		Status:             model.StatusFromVerdict(verdict),
		TrustScore:         vr.TrustScore,
		Summary:            vr.Summary,
		Verdict:            verdict,
		KeyClaim:           claim.MainClaim,
		Entities:           toEntities(claim.Entities),
		Keywords:           claim.Keywords,
		EvidenceFor:        evidenceFor,
		EvidenceAgainst:    evidenceAgainst,
		MissingEvidence:    missingEvidence(inputType, vr.Evidence),
		UnknownInformation: unknownInformation(inputType),
		Interpretations:    interpretations,
		WarningSignals:     signals,
		Confidence:         confidenceOf(evidenceFor, evidenceAgainst, inputType),
		Reasoning:          reasoning.Explain(vr.TrustScore, inputType, evidenceFor, evidenceAgainst),
		Recommendations:    recommendations.Generate(inputType, verdict),
	}

	// Phase 12: multi-perspective fact analysis.
	result.SupportingEvidence = perspectives.SupportingEvidence(result.EvidenceFor)
	result.ContradictingEvidence = perspectives.ContradictingEvidence(result.KeyClaim, result.EvidenceAgainst)
	result.MissingInformation = perspectives.MissingInformation(result.MissingEvidence, result.UnknownInformation, result.WarningSignals)
	result.ConfidenceBreakdown = perspectives.ConfidenceBreakdown(result)
	result.AISummary = perspectives.AISummary(result)
	result.SuggestedReading, result.SuggestedReadingNote = perspectives.SuggestedReading(inputType, result.Entities)
	result.WhatChanged, result.WhatChangedNote = perspectives.WhatChanged(result)

	result.Timeline = buildTimeline(result, neutral)

	for _, m := range a.modules {
		if err := m.Enrich(ctx, input, &result); err != nil {
			// A failing module must not break the whole analysis. Log-free by
			// design: the module name is preserved so callers can surface it.
			result.WarningSignals = append(result.WarningSignals, model.WarningSignal{
				Label:       "Module error: " + m.Name(),
				Severity:    "low",
				Description: fmt.Sprintf("The %s analysis module could not run.", m.Name()),
			})
		}
	}

	return result
}

// toEntities converts the claims package's entity model into the result model.
func toEntities(entities []claims.Entity) []model.Entity {
	out := make([]model.Entity, 0, len(entities))
	for _, e := range entities {
		out = append(out, model.Entity{Name: e.Name, Kind: e.Kind})
	}
	return out
}

// standardChecks lists, per input type, the checks that would normally
// contribute to a full verification. missingEvidence reports the ones that were
// not recorded so the user sees what could not be verified — nothing is
// fabricated or assumed to have passed.
var standardChecks = map[classifier.InputType][]string{
	classifier.TypeDomain:  {"WHOIS registration data", "TLS certificate chain", "Domain reputation history"},
	classifier.TypeURL:     {"Destination content scan", "Redirect chain audit", "TLS certificate chain"},
	classifier.TypeEmail:   {"Sender authentication (SPF/DKIM/DMARC)", "Breach database lookup"},
	classifier.TypePhone:   {"Number owner verification", "Spam-report database lookup"},
	classifier.TypeCompany: {"Official business registry lookup", "Physical address verification"},
	classifier.TypeIPv4:    {"Hosting provider identification", "Abuse report lookup"},
	classifier.TypeIPv6:    {"Hosting provider identification", "Abuse report lookup"},
	classifier.TypeUnknown: {"Verifiable source", "Original publication", "Independent corroboration"},
}

// missingEvidence returns explicit statements about checks that could not be
// performed for the input type.
func missingEvidence(inputType classifier.InputType, evidence []scoring.Evidence) []string {
	present := map[string]bool{}
	for _, e := range evidence {
		present[e.Label] = true
	}

	var missing []string
	for _, check := range standardChecks[inputType] {
		if !present[check] {
			missing = append(missing, "Not verified: "+check+".")
		}
	}
	if len(missing) == 0 && inputType != classifier.TypeUnknown {
		missing = append(missing, "No additional checks were identified for this input type.")
	}
	return missing
}

// unknownInformation returns explicit statements about facts that are genuinely
// unknown for the input type.
func unknownInformation(inputType classifier.InputType) []string {
	if inputType == classifier.TypeUnknown {
		return []string{
			"The author of this content is unknown.",
			"The publication date and provenance of this content are unknown.",
			"Whether this is original reporting, an opinion, or a repost is unknown.",
		}
	}
	return []string{
		"The real-world ownership of this identifier is unknown.",
		"How this identifier is used in practice (legitimate or malicious) is unknown.",
	}
}
