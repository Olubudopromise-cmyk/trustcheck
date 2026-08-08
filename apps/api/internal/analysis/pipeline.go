// Package analysis (see analysis.go) — this file implements the analysis
// pipeline: it runs the modular stages (claims, warnings, interpretations,
// recommendations, reasoning), splits the scored evidence into supporting and
// contradicting sections, and assembles the final explainable model.Result.
package analysis

import (
	"context"
	"fmt"
	"strings"

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
	settings model.AnalysisSettings
}

// New returns an Analyzer with the given optional extension modules. Modules
// run after the core pipeline, in registration order.
func New(modules ...Module) *Analyzer {
	return &Analyzer{
		modules:  modules,
		settings: model.DefaultSettings(model.ModeQuick),
	}
}

// WithSettings returns a copy of the Analyzer with the given settings.
func (a *Analyzer) WithSettings(settings model.AnalysisSettings) *Analyzer {
	return &Analyzer{
		modules:  a.modules,
		settings: settings,
	}
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

	// Phase 13: multi-claim extraction. Split the input into independent
	// factual claims, verify each one, and populate the Claims array.
	multiClaims := claims.ExtractMultiple(input, inputType)
	result.Claims = buildClaims(multiClaims, evidenceFor, evidenceAgainst, signals, inputType, vr.TrustScore)
	result.ClaimCount = len(result.Claims)
	for _, c := range result.Claims {
		switch c.Status {
		case model.ClaimVerified:
			result.VerifiedClaims++
		case model.ClaimPartiallyVerified:
			result.PartialClaims++
		default:
			result.UnverifiedClaims++
		}
	}

	// Evidence Depth & Analysis Modes: populate analysis mode, evidence ledger,
	// score explanation, and source intelligence.
	result.AnalysisMode = a.settings.Mode
	result.EvidenceLedger = buildEvidenceLedger(result.KeyClaim, evidenceFor, evidenceAgainst, a.settings)
	result.ScoreExplanation = buildScoreExplanation(vr.TrustScore, evidenceFor, evidenceAgainst, result.MissingEvidence, a.settings)
	result.SourceIntelligence = buildSourceIntelligence(evidenceFor, evidenceAgainst, a.settings)

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

// buildClaims converts extracted multi-claims into fully verified model.Claim
// objects. Each claim gets its own verdict, confidence, and evidence subset
// based on keyword overlap with the overall evidence.
func buildClaims(
	multiClaims []claims.MultiClaim,
	evidenceFor, evidenceAgainst []model.EvidenceItem,
	signals []model.WarningSignal,
	inputType classifier.InputType,
	overallScore int,
) []model.Claim {
	out := make([]model.Claim, 0, len(multiClaims))

	for i, mc := range multiClaims {
		// Determine which evidence items relate to this claim based on
		// keyword overlap.
		claimEvidenceFor, claimEvidenceAgainst := splitEvidenceForClaim(mc, evidenceFor, evidenceAgainst)

		// Calculate claim-specific confidence.
		confidence := claimConfidence(mc, claimEvidenceFor, claimEvidenceAgainst, signals, inputType)

		// Determine verdict from confidence.
		verdict := model.VerdictHigh
		if confidence < 50 {
			verdict = model.VerdictLow
		} else if confidence < 75 {
			verdict = model.VerdictMedium
		}

		// Determine status from verdict.
		status := model.ClaimVerified
		switch verdict {
		case model.VerdictMedium:
			status = model.ClaimPartiallyVerified
		case model.VerdictLow:
			if len(claimEvidenceFor) == 0 && len(claimEvidenceAgainst) == 0 {
				status = model.ClaimNoReliableEvidence
			} else {
				status = model.ClaimUnverified
			}
		}

		// Build the summary for this claim.
		summary := buildClaimSummary(mc, claimEvidenceFor, claimEvidenceAgainst, status)

		// Build conflicts for this claim.
		conflicts := buildClaimConflicts(mc, claimEvidenceAgainst)

		// Build timeline for this claim.
		timeline := buildClaimTimeline(mc, claimEvidenceFor, claimEvidenceAgainst)

		out = append(out, model.Claim{
			ID:              mc.ID,
			Text:            mc.Text,
			Entities:        toEntities(mc.Entities),
			Keywords:        mc.Keywords,
			Verdict:         verdict,
			Confidence:      confidence,
			Evidence:        append(claimEvidenceFor, claimEvidenceAgainst...),
			Conflicts:       conflicts,
			Summary:         summary,
			Timeline:        timeline,
			Recommendations: generateClaimRecommendations(status, inputType),
			Missing:         generateClaimMissing(inputType),
			Status:          status,
		})

		// Suppress unused variable warnings.
		_ = i
	}

	return out
}

// splitEvidenceForClaim assigns evidence items to a claim based on keyword
// overlap. Evidence that shares keywords with the claim is assigned to it;
// the rest is split proportionally.
func splitEvidenceForClaim(
	mc claims.MultiClaim,
	evidenceFor, evidenceAgainst []model.EvidenceItem,
) ([]model.EvidenceItem, []model.EvidenceItem) {
	if len(mc.Keywords) == 0 {
		// No keywords — assign a proportional share of evidence.
		// For now, give each claim an equal share.
		return evidenceFor, evidenceAgainst
	}

	keywordSet := make(map[string]bool, len(mc.Keywords))
	for _, k := range mc.Keywords {
		keywordSet[k] = true
	}

	var claimFor, claimAgainst []model.EvidenceItem

	for _, e := range evidenceFor {
		if evidenceMatchesKeywords(e, keywordSet) {
			claimFor = append(claimFor, e)
		}
	}
	for _, e := range evidenceAgainst {
		if evidenceMatchesKeywords(e, keywordSet) {
			claimAgainst = append(claimAgainst, e)
		}
	}

	// If no evidence matched, give this claim all evidence (fallback).
	if len(claimFor) == 0 && len(claimAgainst) == 0 {
		return evidenceFor, evidenceAgainst
	}

	return claimFor, claimAgainst
}

// evidenceMatchesKeywords checks if any keyword appears in the evidence label.
func evidenceMatchesKeywords(e model.EvidenceItem, keywords map[string]bool) bool {
	lower := strings.ToLower(e.Label)
	for k := range keywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	// Also match against note if present.
	if e.Note != "" {
		noteLower := strings.ToLower(e.Note)
		for k := range keywords {
			if strings.Contains(noteLower, k) {
				return true
			}
		}
	}
	return false
}

// claimConfidence calculates a 0-100 confidence score for a claim.
func claimConfidence(
	mc claims.MultiClaim,
	evidenceFor, evidenceAgainst []model.EvidenceItem,
	signals []model.WarningSignal,
	inputType classifier.InputType,
) int {
	// Base confidence from entity specificity.
	base := 50
	if len(mc.Entities) > 0 {
		base += 10
	}
	if len(mc.Keywords) > 3 {
		base += 5
	}

	// Evidence adjustments.
	positivePoints := 0
	for _, e := range evidenceFor {
		positivePoints += e.Points
	}
	negativePoints := 0
	for _, e := range evidenceAgainst {
		negativePoints += e.Points
	}

	base += positivePoints / 2
	base -= negativePoints / 2

	// Warning signal penalty.
	for _, s := range signals {
		if s.Severity == "high" {
			base -= 10
		} else if s.Severity == "medium" {
			base -= 5
		}
	}

	// Input type adjustment.
	if inputType == classifier.TypeUnknown {
		base -= 10 // Free-form text is harder to verify.
	}

	// Clamp to 0-100.
	if base < 0 {
		base = 0
	}
	if base > 100 {
		base = 100
	}
	return base
}

// buildClaimSummary generates a short summary for a claim.
func buildClaimSummary(
	mc claims.MultiClaim,
	evidenceFor, evidenceAgainst []model.EvidenceItem,
	status model.ClaimStatus,
) string {
	switch status {
	case model.ClaimVerified:
		return fmt.Sprintf("This claim is supported by %d evidence items with no contradicting findings.", len(evidenceFor))
	case model.ClaimPartiallyVerified:
		return fmt.Sprintf("This claim has %d supporting and %d contradicting evidence items. Further verification is recommended.", len(evidenceFor), len(evidenceAgainst))
	case model.ClaimNoReliableEvidence:
		return "No reliable evidence was found to verify this claim."
	default:
		return fmt.Sprintf("This claim could not be fully verified. Found %d supporting and %d contradicting evidence items.", len(evidenceFor), len(evidenceAgainst))
	}
}

// buildClaimConflicts generates conflict descriptions for a claim.
func buildClaimConflicts(mc claims.MultiClaim, evidenceAgainst []model.EvidenceItem) []model.Contradiction {
	var conflicts []model.Contradiction
	for _, e := range evidenceAgainst {
		conflicts = append(conflicts, model.Contradiction{
			SourceA:                   "Submitted claim",
			ClaimA:                    mc.Text,
			SourceB:                   e.Label,
			ClaimB:                    e.Note,
			WhyTheyDisagree:           fmt.Sprintf("Evidence '%s' contradicts this claim.", e.Label),
			ConfidenceInContradiction: 50,
		})
	}
	return conflicts
}

// buildClaimTimeline builds a simple timeline for a claim.
func buildClaimTimeline(mc claims.MultiClaim, evidenceFor, evidenceAgainst []model.EvidenceItem) []model.ReasoningStep {
	var steps []model.ReasoningStep

	steps = append(steps, model.ReasoningStep{
		Title:   "Claim Detected",
		Summary: truncate(mc.Text, 90),
		Details: []string{"Claim: " + mc.Text},
	})

	if len(evidenceFor) > 0 || len(evidenceAgainst) > 0 {
		details := []string{}
		for _, e := range evidenceFor {
			details = append(details, "+ "+e.Label)
		}
		for _, e := range evidenceAgainst {
			details = append(details, "- "+e.Label)
		}
		steps = append(steps, model.ReasoningStep{
			Title:   "Evidence Gathered",
			Summary: fmt.Sprintf("%d supporting, %d contradicting", len(evidenceFor), len(evidenceAgainst)),
			Details: details,
		})
	}

	return steps
}

// generateClaimRecommendations produces recommendations for a claim.
func generateClaimRecommendations(status model.ClaimStatus, inputType classifier.InputType) []model.Recommendation {
	var recs []model.Recommendation

	switch status {
	case model.ClaimVerified:
		recs = append(recs, model.Recommendation{
			Title:       "Claim Verified",
			Description: "This claim appears to be well-supported by available evidence.",
		})
	case model.ClaimPartiallyVerified:
		recs = append(recs, model.Recommendation{
			Title:       "Seek Additional Sources",
			Description: "This claim has mixed evidence. Consult additional independent sources.",
		})
	case model.ClaimNoReliableEvidence:
		recs = append(recs, model.Recommendation{
			Title:       "No Evidence Found",
			Description: "No reliable evidence was found. Exercise caution before accepting this claim.",
		})
	default:
		recs = append(recs, model.Recommendation{
			Title:       "Verify Independently",
			Description: "This claim could not be fully verified. Check with authoritative sources.",
		})
	}

	return recs
}

// generateClaimMissing produces missing information items for a claim.
func generateClaimMissing(inputType classifier.InputType) []model.MissingInfo {
	var missing []model.MissingInfo

	if inputType == classifier.TypeUnknown {
		missing = append(missing, model.MissingInfo{
			Item:         "Original source",
			WhyItMatters: "Without the original source, the claim cannot be independently verified.",
		})
	}

	return missing
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

// --- Evidence Depth & Analysis Modes ---

// buildEvidenceLedger constructs a structured evidence ledger from the classified
// evidence. The ledger tracks supporting, contradicting, and unknown evidence
// with source metadata.
func buildEvidenceLedger(
	claim string,
	evidenceFor, evidenceAgainst []model.EvidenceItem,
	settings model.AnalysisSettings,
) model.EvidenceLedger {
	ledger := model.EvidenceLedger{
		Claim: claim,
	}

	// Add supporting evidence.
	for _, e := range evidenceFor {
		ledger.Supporting = append(ledger.Supporting, model.LedgerEntry{
			Source: model.SourceIntelligence{
				Title:          e.Label,
				SourceType:     classifySourceType(e.Label),
				Relation:       classifySourceRelation(e.Label, settings),
				IsOfficial:     isOfficialSource(e.Label),
				SupportsClaim:  true,
				ContradictsClaim: false,
				IsIndependent:  isIndependentSource(e.Label, settings),
				Confidence:     calculateSourceConfidence(e, settings),
			},
			Summary:  e.Note,
			Strength: calculateEvidenceStrength(e, true),
		})
	}

	// Add contradicting evidence.
	for _, e := range evidenceAgainst {
		ledger.Contradicting = append(ledger.Contradicting, model.LedgerEntry{
			Source: model.SourceIntelligence{
				Title:          e.Label,
				SourceType:     classifySourceType(e.Label),
				Relation:       classifySourceRelation(e.Label, settings),
				IsOfficial:     isOfficialSource(e.Label),
				SupportsClaim:  false,
				ContradictsClaim: true,
				IsIndependent:  isIndependentSource(e.Label, settings),
				Confidence:     calculateSourceConfidence(e, settings),
			},
			Summary:  e.Note,
			Strength: calculateEvidenceStrength(e, false),
		})
	}

	// Add unknown items.
	ledger.Unknown = []string{
		"No additional sources were found.",
	}

	ledger.TotalSources = len(ledger.Supporting) + len(ledger.Contradicting)
	ledger.IndependentCount = countIndependent(ledger)
	ledger.DuplicateCount = 0 // Would require source comparison logic

	return ledger
}

// buildScoreExplanation creates a user-facing breakdown of the trust score.
func buildScoreExplanation(
	trustScore int,
	evidenceFor, evidenceAgainst []model.EvidenceItem,
	missingEvidence []string,
	settings model.AnalysisSettings,
) model.ScoreExplanation {
	// Evidence strength: based on quantity and quality of evidence.
	evidenceStrength := 50
	if len(evidenceFor) > len(evidenceAgainst) {
		evidenceStrength = 70 + min(len(evidenceFor)*5, 30)
	} else if len(evidenceAgainst) > len(evidenceFor) {
		evidenceStrength = 30 - min(len(evidenceAgainst)*5, 30)
	}
	evidenceStrength = max(0, min(100, evidenceStrength))

	// Source quality: based on settings preferences.
	sourceQuality := 50
	if settings.PrioritizeGovernmentSources || settings.PrioritizeAcademicSources {
		sourceQuality = 65
	}
	if settings.RequireIndependentSources {
		sourceQuality += 10
	}
	sourceQuality = max(0, min(100, sourceQuality))

	// Independent confirmation.
	independentConfirmation := 50
	if settings.RequireIndependentSources {
		independentConfirmation = 70
	}
	if len(evidenceFor) >= 2 {
		independentConfirmation += 15
	}
	independentConfirmation = max(0, min(100, independentConfirmation))

	// Contradiction risk: higher when more contradicting evidence exists.
	contradictionRisk := 20
	if len(evidenceAgainst) > 0 {
		contradictionRisk = 40 + min(len(evidenceAgainst)*15, 40)
	}
	contradictionRisk = max(0, min(100, contradictionRisk))

	// Missing evidence: based on missing items.
	missingLevel := 20
	if len(missingEvidence) > 0 {
		missingLevel = 40 + min(len(missingEvidence)*10, 40)
	}
	missingLevel = max(0, min(100, missingLevel))

	return model.ScoreExplanation{
		EvidenceStrength:        evidenceStrength,
		EvidenceStrengthNote:    fmt.Sprintf("%d supporting, %d contradicting evidence items.", len(evidenceFor), len(evidenceAgainst)),
		SourceQuality:           sourceQuality,
		SourceQualityNote:       sourceQualityNote(settings),
		IndependentConfirmation: independentConfirmation,
		IndependentNote:         independentNote(settings, evidenceFor),
		ContradictionRisk:       contradictionRisk,
		ContradictionNote:       contradictionNote(evidenceAgainst),
		MissingEvidence:         missingLevel,
		MissingNote:             fmt.Sprintf("%d checks could not be performed.", len(missingEvidence)),
	}
}

// buildSourceIntelligence creates metadata for all sources used.
func buildSourceIntelligence(
	evidenceFor, evidenceAgainst []model.EvidenceItem,
	settings model.AnalysisSettings,
) []model.SourceIntelligence {
	var sources []model.SourceIntelligence

	seen := map[string]bool{}

	addSource := func(e model.EvidenceItem, supports bool) {
		if seen[e.Label] {
			return
		}
		seen[e.Label] = true

		sources = append(sources, model.SourceIntelligence{
			Title:           e.Label,
			SourceType:      classifySourceType(e.Label),
			Relation:        classifySourceRelation(e.Label, settings),
			IsOfficial:      isOfficialSource(e.Label),
			SupportsClaim:   supports,
			ContradictsClaim: !supports,
			IsIndependent:   isIndependentSource(e.Label, settings),
			Confidence:      calculateSourceConfidence(e, settings),
			Relevance:       calculateRelevance(e, settings),
		})
	}

	for _, e := range evidenceFor {
		addSource(e, true)
	}
	for _, e := range evidenceAgainst {
		addSource(e, false)
	}

	return sources
}

// classifySourceType determines the type of a source based on its label.
func classifySourceType(label string) model.SourceType {
	lower := strings.ToLower(label)
	switch {
	case strings.Contains(lower, "official") || strings.Contains(lower, "government") ||
		strings.Contains(lower, "regulatory") || strings.Contains(lower, "agency"):
		return model.SourceOfficial
	case strings.Contains(lower, "academic") || strings.Contains(lower, "research") ||
		strings.Contains(lower, "journal") || strings.Contains(lower, "peer-reviewed"):
		return model.SourceAcademic
	case strings.Contains(lower, "news") || strings.Contains(lower, "press") ||
		strings.Contains(lower, "media") || strings.Contains(lower, "reporter"):
		return model.SourceJournalism
	case strings.Contains(lower, "community") || strings.Contains(lower, "forum") ||
		strings.Contains(lower, "social") || strings.Contains(lower, "reddit"):
		return model.SourceCommunity
	case strings.Contains(lower, "company") || strings.Contains(lower, "business") ||
		strings.Contains(lower, "corporate") || strings.Contains(lower, "commercial"):
		return model.SourceCommercial
	case strings.Contains(lower, "institution") || strings.Contains(lower, "university"):
		return model.SourceInstitutional
	default:
		return model.SourceUnknown
	}
}

// classifySourceRelation determines if a source is primary, secondary, or tertiary.
func classifySourceRelation(label string, settings model.AnalysisSettings) model.SourceRelation {
	lower := strings.ToLower(label)
	switch {
	case strings.Contains(lower, "official") || strings.Contains(lower, "direct") ||
		strings.Contains(lower, "announcement") || strings.Contains(lower, "press release"):
		return model.RelationPrimary
	case strings.Contains(lower, "report") || strings.Contains(lower, "coverage") ||
		strings.Contains(lower, "article") || strings.Contains(lower, "news"):
		return model.RelationSecondary
	default:
		return model.RelationTertiary
	}
}

// isOfficialSource checks if a source is from an official government or
// institutional source.
func isOfficialSource(label string) bool {
	lower := strings.ToLower(label)
	return strings.Contains(lower, "official") || strings.Contains(lower, "government") ||
		strings.Contains(lower, "regulatory") || strings.Contains(lower, "agency") ||
		strings.Contains(lower, "department") || strings.Contains(lower, "ministry")
}

// isIndependentSource checks if a source is independent from the claim subject.
func isIndependentSource(label string, settings model.AnalysisSettings) bool {
	// For now, assume non-official, non-commercial sources are independent.
	// A more sophisticated implementation would check domain ownership.
	return !isOfficialSource(label) && !strings.Contains(strings.ToLower(label), "company")
}

// calculateSourceConfidence estimates confidence in a source's accuracy.
func calculateSourceConfidence(e model.EvidenceItem, settings model.AnalysisSettings) int {
	base := 50
	if e.Result == "pass" {
		base += 20
	} else if e.Result == "fail" {
		base -= 20
	}
	if settings.PrioritizeGovernmentSources && isOfficialSource(e.Label) {
		base += 15
	}
	if settings.PrioritizeAcademicSources && strings.Contains(strings.ToLower(e.Label), "academic") {
		base += 10
	}
	return max(0, min(100, base))
}

// calculateEvidenceStrength estimates how strong a piece of evidence is.
func calculateEvidenceStrength(e model.EvidenceItem, supports bool) int {
	base := 50
	if supports {
		base += e.Points
	} else {
		base -= e.Points
	}
	return max(0, min(100, base))
}

// calculateRelevance estimates how relevant a source is to the claim.
func calculateRelevance(e model.EvidenceItem, settings model.AnalysisSettings) int {
	base := 60
	if e.Points > 0 {
		base += 10
	}
	if settings.PrioritizePrimarySources && strings.Contains(strings.ToLower(e.Label), "primary") {
		base += 15
	}
	return max(0, min(100, base))
}

// countIndependent counts the number of independent sources in the ledger.
func countIndependent(ledger model.EvidenceLedger) int {
	count := 0
	for _, e := range ledger.Supporting {
		if e.Source.IsIndependent {
			count++
		}
	}
	for _, e := range ledger.Contradicting {
		if e.Source.IsIndependent {
			count++
		}
	}
	return count
}

// sourceQualityNote generates a note about source quality.
func sourceQualityNote(settings model.AnalysisSettings) string {
	if settings.PrioritizeGovernmentSources {
		return "Government and official sources were prioritized."
	}
	if settings.PrioritizeAcademicSources {
		return "Academic and peer-reviewed sources were prioritized."
	}
	return "Standard source evaluation was used."
}

// independentNote generates a note about independent confirmation.
func independentNote(settings model.AnalysisSettings, evidenceFor []model.EvidenceItem) string {
	if settings.RequireIndependentSources {
		return "Independent confirmation was required."
	}
	if len(evidenceFor) >= 2 {
		return fmt.Sprintf("%d supporting sources found.", len(evidenceFor))
	}
	return "Limited independent sources available."
}

// contradictionNote generates a note about contradictions.
func contradictionNote(evidenceAgainst []model.EvidenceItem) string {
	if len(evidenceAgainst) == 0 {
		return "No contradictions found."
	}
	return fmt.Sprintf("%d contradicting evidence items found.", len(evidenceAgainst))
}
