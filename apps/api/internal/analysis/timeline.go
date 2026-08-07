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

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

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
			label = cases.Title(language.Und).String(kind)
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

// conflictsStep reports where evidence disagrees, or states that it does not.
// The result's EvidenceAgainst is already filtered by the pipeline's shared
// classification (engine fallback messages never reach this section), so every
// contradicting item here is real evidence about the claim.
func conflictsStep(result model.Result) model.ReasoningStep {
	conflicts := result.EvidenceAgainst

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

// reasoningSummary condenses the assessment into one user-facing sentence
// derived from the evidence counts, so it can never contradict the sections
// above it.
func reasoningSummary(result model.Result) string {
	support := len(result.EvidenceFor)
	against := len(result.EvidenceAgainst)

	switch {
	case support == 0 && against == 0:
		return "No scored evidence could be gathered, so this assessment rests on the input itself."
	case support > against:
		return fmt.Sprintf(
			"Supporting evidence outweighs contradicting evidence (%d supporting vs %d contradicting), so the assessment leans positive.",
			support, against,
		)
	case against > support:
		return fmt.Sprintf(
			"Contradicting evidence outweighs supporting evidence (%d supporting vs %d contradicting), so the assessment leans negative.",
			support, against,
		)
	default:
		return fmt.Sprintf(
			"Supporting and contradicting evidence are evenly balanced (%d each), so the assessment is mixed.",
			support,
		)
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

// truncate shortens a string to max runes, appending an ellipsis. It operates
// on runes (not bytes) so multibyte characters like emoji are never split.
func truncate(s string, max int) string {
	if max < 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
