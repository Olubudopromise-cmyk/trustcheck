// Timeline builder: turns the assembled analysis into the six-step reasoning
// timeline shown at the top of every result, so the user reviews an
// investigation rather than an unexplained score.
//
// The steps are: Claim Detected → Evidence Gathered → Conflicts Identified →
// Risk Signals Detected → AI Reasoning → Final Assessment. Every summary and
// detail line is grounded in observable data from the analysis; when nothing
// was found the step says so explicitly. No chain-of-thought or internal
// reasoning is ever exposed.
package analysis

import (
	"fmt"
	"strings"

	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

// Estimated per-signal score impact, used for the risk-signal step. These are
// documented estimates applied consistently so users see an effect on the
// score; they are presented as estimates, never as fabricated measurements.
const (
	signalImpactHigh   = 15
	signalImpactMedium = 8
	signalImpactLow    = 3
)

// buildTimeline assembles the six reasoning steps for a completed analysis.
func buildTimeline(result model.Result, neutral int) []model.ReasoningStep {
	return []model.ReasoningStep{
		claimStep(result),
		evidenceStep(result, neutral),
		conflictsStep(result),
		riskSignalsStep(result),
		reasoningStep(result),
		finalAssessmentStep(result),
	}
}

// claimStep presents the extracted main claim, entities, and keywords.
func claimStep(result model.Result) model.ReasoningStep {
	details := []string{"Claim: " + result.KeyClaim}
	details = append(details, entityLines(result.Entities)...)
	if len(result.Keywords) > 0 {
		details = append(details, "Keywords: "+strings.Join(result.Keywords, ", "))
	}

	return model.ReasoningStep{
		Title:   "Claim Detected",
		Summary: truncate(result.KeyClaim, 90),
		Details: details,
	}
}

// entityLines groups entities by kind into readable detail lines.
func entityLines(entities []model.Entity) []string {
	byKind := map[string][]string{}
	order := []string{}
	for _, e := range entities {
		if _, ok := byKind[e.Kind]; !ok {
			order = append(order, e.Kind)
		}
		byKind[e.Kind] = append(byKind[e.Kind], e.Name)
	}

	kindLabel := map[string]string{
		model.EntityPerson:       "People",
		model.EntityOrganization: "Organizations",
		model.EntityLocation:     "Locations",
		model.EntityDate:         "Dates",
	}

	var lines []string
	for _, kind := range order {
		label := kindLabel[kind]
		if label == "" {
			label = strings.Title(kind)
		}
		lines = append(lines, fmt.Sprintf("%s: %s", label, strings.Join(byKind[kind], ", ")))
	}
	return lines
}

// evidenceStep reports how much supporting, contradicting, and neutral
// evidence was found. When nothing was found it says so explicitly.
func evidenceStep(result model.Result, neutral int) model.ReasoningStep {
	supporting := len(result.EvidenceFor)
	contradicting := len(result.EvidenceAgainst)
	total := supporting + contradicting + neutral

	summary := fmt.Sprintf("%d supporting, %d contradicting, %d neutral references.", supporting, contradicting, neutral)
	details := []string{
		fmt.Sprintf("✓ Supporting evidence: %d", supporting),
		fmt.Sprintf("✗ Contradicting evidence: %d", contradicting),
		fmt.Sprintf("• Neutral references: %d", neutral),
	}
	if total == 0 {
		details = append(details, "No external sources could be located. None were invented.")
	}

	return model.ReasoningStep{
		Title:   "Evidence Gathered",
		Summary: summary,
		Details: details,
	}
}

// conflictExclusions lists check labels that are engine fallback messages
// ("I could not suggest anything") rather than evidence about the claim, so
// they are never presented as conflicting information. Real failed checks and
// warnings still appear.
var conflictExclusions = map[string]bool{
	"No Suggestion": true,
}

// conflictsStep reports where evidence disagrees, or states that it does not.
func conflictsStep(result model.Result) model.ReasoningStep {
	var conflicts []model.EvidenceItem
	for _, e := range result.EvidenceAgainst {
		if !conflictExclusions[e.Label] {
			conflicts = append(conflicts, e)
		}
	}

	if len(conflicts) == 0 {
		return model.ReasoningStep{
			Title:   "Conflicts Identified",
			Summary: "No conflicting evidence identified.",
			Details: []string{"All checks agreed; no conflicting information was found."},
		}
	}

	details := make([]string, 0, len(conflicts))
	for _, e := range conflicts {
		details = append(details, "• "+e.Label)
	}
	return model.ReasoningStep{
		Title:   "Conflicts Identified",
		Summary: fmt.Sprintf("%d conflicting point(s) found.", len(conflicts)),
		Details: details,
	}
}

// riskSignalsStep lists the credibility concerns with their severity and their
// estimated effect on the score. If none were detected it says so.
func riskSignalsStep(result model.Result) model.ReasoningStep {
	if len(result.WarningSignals) == 0 {
		return model.ReasoningStep{
			Title:   "Risk Signals Detected",
			Summary: "No risk signals detected.",
			Details: []string{"No credibility concerns were detected in the input."},
		}
	}

	details := make([]string, 0, len(result.WarningSignals))
	for _, s := range result.WarningSignals {
		details = append(details, fmt.Sprintf(
			"⚠ %s (%s) — estimated impact on trust score: -%d points.",
			s.Label, s.Severity, signalImpact(s.Severity),
		))
	}
	return model.ReasoningStep{
		Title:   "Risk Signals Detected",
		Summary: fmt.Sprintf("%d risk signal(s) detected.", len(result.WarningSignals)),
		Details: details,
	}
}

// signalImpact maps a signal severity to its estimated score impact.
func signalImpact(severity string) int {
	switch severity {
	case "high":
		return signalImpactHigh
	case "medium":
		return signalImpactMedium
	default:
		return signalImpactLow
	}
}

// reasoningStep exposes a concise, non-technical explanation of the factors
// that influenced the assessment. The detail lines are the grounded reasoning
// bullets; no chain-of-thought is exposed.
func reasoningStep(result model.Result) model.ReasoningStep {
	return model.ReasoningStep{
		Title:   "AI Reasoning",
		Summary: reasoningSummary(result),
		Details: result.Reasoning,
	}
}

// reasoningSummary condenses the assessment into one user-facing sentence.
func reasoningSummary(result model.Result) string {
	switch result.Verdict {
	case model.VerdictHigh:
		return "The claim passed most checks with little contradicting evidence, so confidence is high."
	case model.VerdictMedium:
		return "Supporting and contradicting signals balance each other, so the assessment is mixed."
	default:
		return "The claim failed checks and/or carried risk signals, so confidence is low."
	}
}

// finalAssessmentStep is the last step: the score, verdict, confidence,
// strengths, weaknesses, and the top recommendation.
func finalAssessmentStep(result model.Result) model.ReasoningStep {
	details := []string{
		fmt.Sprintf("Trust score: %d / 100", result.TrustScore),
		fmt.Sprintf("Verdict: %s", result.Verdict),
		fmt.Sprintf("Analysis confidence: %d%%", result.Confidence),
	}

	if len(result.EvidenceFor) > 0 {
		labels := make([]string, 0, len(result.EvidenceFor))
		for _, e := range result.EvidenceFor {
			labels = append(labels, e.Label)
		}
		details = append(details, "Key strengths: "+strings.Join(labels, ", "))
	} else {
		details = append(details, "Key strengths: none — no supporting evidence was found.")
	}

	if len(result.EvidenceAgainst) > 0 || len(result.WarningSignals) > 0 {
		details = append(details, fmt.Sprintf("Key weaknesses: %d contradicting check(s) and %d risk signal(s).", len(result.EvidenceAgainst), len(result.WarningSignals)))
	} else {
		details = append(details, "Key weaknesses: none identified.")
	}

	if len(result.Recommendations) > 0 {
		details = append(details, "Top recommendation: "+result.Recommendations[0].Title+".")
	}

	return model.ReasoningStep{
		Title:   "Final Assessment",
		Summary: fmt.Sprintf("Final score %d/100 — %s verdict, %d%% confidence.", result.TrustScore, result.Verdict, result.Confidence),
		Details: details,
	}
}

// truncate shortens a string to max runes, appending an ellipsis.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
