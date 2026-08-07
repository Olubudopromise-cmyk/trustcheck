// Package perspectives builds the multi-perspective fact-analysis sections of
// a TrustCheck result: evidence grouped by source, contradictions, missing
// information, a user-facing confidence breakdown, an AI summary, suggested
// reading, and the "what changed" story timeline.
//
// Honesty rules. Every section is grounded in observable findings. When a
// section has nothing real to show (no dated source history, no specific
// articles that could be identified) it returns an explicit statement instead
// of fabricating evidence, dates, citations, links, or timelines. No internal
// reasoning or hidden scoring algorithm is ever exposed — the confidence
// breakdown is a set of user-friendly metrics with plain-English notes.
package perspectives

import (
	"fmt"
	"strings"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

// credibilityFromResult maps a scored check result to a user-facing
// credibility level.
func credibilityFromResult(result string) string {
	switch result {
	case "pass":
		return "high"
	case "warning":
		return "medium"
	case "fail":
		return "low"
	default:
		return "unknown"
	}
}

// sourceCategory maps an evidence label to the category of check it came from.
// Categories describe where the evidence actually originated (infrastructure,
// security, ...) rather than pretending to be named external publications.
func sourceCategory(label string) string {
	low := strings.ToLower(label)
	switch {
	case strings.Contains(low, "dns") || strings.Contains(low, "mx record") ||
		strings.Contains(low, "record") || strings.Contains(low, "resolves") ||
		strings.Contains(low, "unicast") || strings.Contains(low, "loopback") ||
		strings.Contains(low, "private network") || strings.Contains(low, "link-local") ||
		strings.Contains(low, "multicast") || strings.Contains(low, "reserved") ||
		strings.Contains(low, "unspecified") || strings.Contains(low, "ip"):
		return "Network infrastructure"
	case strings.Contains(low, "tls") || strings.Contains(low, "https") ||
		strings.Contains(low, "certificate") || strings.Contains(low, "ssl") ||
		strings.Contains(low, "downgrade"):
		return "Transport security"
	case strings.Contains(low, "company name") || strings.Contains(low, "legal suffix") ||
		strings.Contains(low, "e.164") || strings.Contains(low, "syntax") ||
		strings.Contains(low, "disposable") || strings.Contains(low, "mail domain") ||
		strings.Contains(low, "format") || strings.Contains(low, "valid"):
		return "Format and identity"
	case strings.Contains(low, "http") || strings.Contains(low, "status") ||
		strings.Contains(low, "redirect") || strings.Contains(low, "fallback") ||
		strings.Contains(low, "request") || strings.Contains(low, "website") ||
		strings.Contains(low, "url") || strings.Contains(low, "error"):
		return "Web reachability"
	case strings.Contains(low, "suggestion") || strings.Contains(low, "input provided") ||
		strings.Contains(low, "normalized") || strings.Contains(low, "country"):
		return "Input analysis"
	default:
		return "Verification checks"
	}
}

// sourceDescription is the plain-English origin of a check for a category.
func sourceDescription(category string) string {
	switch category {
	case "Network infrastructure":
		return "Live DNS and network records"
	case "Transport security":
		return "TLS certificate chain"
	case "Format and identity":
		return "Format and identity validation"
	case "Web reachability":
		return "Live HTTP reachability check"
	case "Input analysis":
		return "Input classification heuristics"
	default:
		return "Automated verification checks"
	}
}

// evidenceSummary is a plain-English account of what a check found.
func evidenceSummary(item model.EvidenceItem) string {
	var verb string
	switch item.Result {
	case "pass":
		verb = "passed"
	case "warning":
		verb = "flagged a concern"
	case "fail":
		verb = "failed"
	default:
		verb = "reported"
	}
	if item.Note != "" {
		return fmt.Sprintf("The %s check %s: %s", item.Label, verb, item.Note)
	}
	return fmt.Sprintf("The %s check %s.", item.Label, verb)
}

// SupportingEvidence groups the supporting scored checks by the category of
// source they came from. Each item carries a credibility level and a summary;
// PublicationDate is empty because no dated publication was observed.
func SupportingEvidence(evidenceFor []model.EvidenceItem) []model.SourceGroup {
	var groups []model.SourceGroup
	index := map[string]int{}

	for _, item := range evidenceFor {
		category := sourceCategory(item.Label)
		i, ok := index[category]
		if !ok {
			groups = append(groups, model.SourceGroup{Category: category})
			i = len(groups) - 1
			index[category] = i
		}
		groups[i].Items = append(groups[i].Items, model.SourceEvidence{
			Title:       item.Label,
			Source:      sourceDescription(category),
			Credibility: credibilityFromResult(item.Result),
			Summary:     evidenceSummary(item),
		})
	}
	return groups
}

// ContradictingEvidence turns each contradicting check into a structured
// disagreement between the submitted claim and what the check observed. If no
// contradicting evidence was found the list is empty; the caller never claims
// a conflict exists when none was observed.
func ContradictingEvidence(keyClaim string, evidenceAgainst []model.EvidenceItem) []model.Contradiction {
	if len(evidenceAgainst) == 0 {
		return nil
	}

	contradictions := make([]model.Contradiction, 0, len(evidenceAgainst))
	for _, item := range evidenceAgainst {
		confidence := 55 // warning
		if item.Result == "fail" {
			confidence = 80
		}
		contradictions = append(contradictions, model.Contradiction{
			SourceA: "Submitted claim",
			ClaimA:  keyClaim,
			SourceB: sourceDescription(sourceCategory(item.Label)),
			ClaimB:  item.Label,
			WhyTheyDisagree: fmt.Sprintf(
				"The %s check %s, which contradicts the claim's implication that this aspect is sound.",
				item.Label, contradictionVerb(item.Result),
			),
			ConfidenceInContradiction: confidence,
		})
	}
	return contradictions
}

func contradictionVerb(result string) string {
	switch result {
	case "fail":
		return "failed"
	default:
		return "raised a concern"
	}
}

// MissingInformation assembles the "What is missing?" section from the checks
// that could not be run, the facts that are genuinely unknown, and the
// detected warnings. Every item is grounded; none is invented.
func MissingInformation(missingEvidence, unknownInformation []string, warnings []model.WarningSignal) []model.MissingInfo {
	var items []model.MissingInfo
	seen := map[string]bool{}
	add := func(item, why string) {
		if item == "" || seen[item] {
			return
		}
		seen[item] = true
		items = append(items, model.MissingInfo{Item: item, WhyItMatters: why})
	}

	for _, statement := range missingEvidence {
		clean := strings.TrimSpace(strings.TrimPrefix(statement, "Not verified: "))
		clean = strings.TrimSuffix(clean, ".")
		add(clean, "This check could not be run, so this aspect remains unverified.")
	}
	for _, statement := range unknownInformation {
		add(strings.TrimSuffix(strings.TrimSpace(statement), "."),
			"This fact could not be established from the available evidence.")
	}
	for _, w := range warnings {
		switch w.Label {
		case "No citations or sources":
			add("Citations or references", "Without citations the claim cannot be independently traced or verified.")
		case "No publication date":
			add("Publication date", "Without a date the information cannot be checked for recency or timeliness.")
		case "Anonymous author":
			add("Author identity", "Anonymous authorship makes accountability unclear.")
		case "Manipulated statistics":
			add("Supporting study or data", "Statistics stated without data cannot be verified.")
		}
	}
	return items
}

// ConfidenceBreakdown computes the user-facing, 0-100 confidence metrics. These
// are explanatory sub-scores derived from observable findings, not the hidden
// scoring algorithm. Overall mirrors the analysis confidence already reported.
func ConfidenceBreakdown(result model.Result) model.ConfidenceBreakdown {
	forWeight, againstWeight := weightOf(result.EvidenceFor), weightOf(result.EvidenceAgainst)
	total := len(result.EvidenceFor) + len(result.EvidenceAgainst)

	// Source credibility: share of scored weight that supported the claim.
	credibility := 0
	if forWeight+againstWeight > 0 {
		credibility = percent(forWeight, forWeight+againstWeight)
	}
	credibilityNote := "Share of scored evidence weight that supported the claim."
	if total == 0 {
		credibilityNote = "No scored evidence was available to judge source credibility."
	}

	// Citation quality: whether the input references sources.
	citation, citationNote := citationMetric(result)

	// Evidence consistency: how aligned the checks were with each other.
	consistency, consistencyNote := consistencyMetric(result)

	// Language neutrality: penalized by sensational/emotional wording.
	neutrality, neutralityNote := neutralityMetric(result)

	// Transparency: share of expected checks that could actually be run.
	transparency, transparencyNote := transparencyMetric(result)

	// Freshness: only ever based on observed dates.
	freshness, freshnessNote := freshnessMetric(result)

	metrics := []model.ConfidenceMetric{
		{Name: "Source credibility", Score: credibility, Note: credibilityNote},
		{Name: "Citation quality", Score: citation, Note: citationNote},
		{Name: "Evidence consistency", Score: consistency, Note: consistencyNote},
		{Name: "Language neutrality", Score: neutrality, Note: neutralityNote},
		{Name: "Transparency", Score: transparency, Note: transparencyNote},
		{Name: "Freshness", Score: freshness, Note: freshnessNote},
	}
	return model.ConfidenceBreakdown{Overall: result.Confidence, Metrics: metrics}
}

// weightOf sums the absolute contribution of a list of evidence items.
func weightOf(items []model.EvidenceItem) int {
	total := 0
	for _, item := range items {
		if item.Points < 0 {
			total -= item.Points
		} else {
			total += item.Points
		}
	}
	return total
}

func percent(part, whole int) int {
	if whole <= 0 {
		return 0
	}
	return part * 100 / whole
}

func citationMetric(result model.Result) (int, string) {
	for _, w := range result.WarningSignals {
		if w.Label == "No citations or sources" {
			return 15, "The input references no citations or sources, so claims cannot be independently verified."
		}
	}
	if len(result.EvidenceFor) == 0 && len(result.EvidenceAgainst) == 0 {
		return 30, "No evidence was collected to assess citation quality."
	}
	return 65, "Scored checks serve as the citations for this assessment; no external citations were referenced."
}

func consistencyMetric(result model.Result) (int, string) {
	total := len(result.EvidenceFor) + len(result.EvidenceAgainst)
	if total == 0 {
		return 0, "No checks were run, so consistency could not be assessed."
	}
	support := len(result.EvidenceFor)
	ratio := float64(support) / float64(total)
	consistency := int(100 * 2 * abs(ratio-0.5))
	if consistency > 100 {
		consistency = 100
	}
	note := "How strongly the scored checks agreed on a single direction."
	if consistency >= 90 {
		note = "All scored checks pointed the same way."
	}
	return consistency, note
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func neutralityMetric(result model.Result) (int, string) {
	score := 100
	for _, w := range result.WarningSignals {
		switch w.Label {
		case "Sensational headline", "Clickbait wording", "Excessive emotional language":
			score -= 20
		}
	}
	if score < 0 {
		score = 0
	}
	note := "No sensational, clickbait, or emotionally loaded language was detected."
	if score < 100 {
		note = "Sensational, clickbait, or emotional wording reduced the neutrality score."
	}
	return score, note
}

func transparencyMetric(result model.Result) (int, string) {
	run := len(result.EvidenceFor) + len(result.EvidenceAgainst)
	missing := len(result.MissingEvidence)
	if run+missing == 0 {
		return 50, "No checks were expected or run for this input type."
	}
	return percent(run, run+missing), "Share of the expected checks that could actually be performed."
}

func freshnessMetric(result model.Result) (int, string) {
	if len(result.EvidenceFor) == 0 && len(result.EvidenceAgainst) == 0 {
		return 0, "No date information was observed, so freshness could not be assessed."
	}
	return 60, "Checks were performed at verification time; no publication dates were observed for the underlying sources."
}

// AISummary writes a short (≤120 word) user-facing paragraph. It is clearly a
// summary of the structured sections above — it never invents facts beyond
// what those sections report.
func AISummary(result model.Result) string {
	support, against := len(result.EvidenceFor), len(result.EvidenceAgainst)

	var b strings.Builder
	fmt.Fprintf(&b, "This claim is assessed as %s with a trust score of %d out of 100 (%s). ",
		result.Verdict, result.TrustScore, result.Status)
	switch {
	case support > 0 && against == 0:
		fmt.Fprintf(&b, "The evidence consistently supports the claim, with %d supporting check(s) and no contradicting evidence. ", support)
	case support == 0 && against > 0:
		fmt.Fprintf(&b, "The evidence contradicts the claim, with %d contradicting check(s) and no supporting evidence. ", against)
	case support > 0 && against > 0:
		fmt.Fprintf(&b, "The evidence is mixed: %d supporting and %d contradicting check(s) were recorded. ", support, against)
	default:
		b.WriteString("No scored evidence could be gathered, so this assessment rests on the wording alone. ")
	}
	if len(result.WarningSignals) > 0 {
		fmt.Fprintf(&b, "%d risk signal(s) were detected, including %q. ", len(result.WarningSignals), result.WarningSignals[0].Label)
	}
	if len(result.MissingEvidence) > 0 || len(result.UnknownInformation) > 0 {
		b.WriteString("Several important facts remain unverified, including missing information and unknown details. ")
	}
	fmt.Fprintf(&b, "Overall confidence in this assessment is %d%%.", result.Confidence)

	return limitWords(b.String(), 120)
}

func limitWords(s string, max int) string {
	words := strings.Fields(s)
	if len(words) <= max {
		return s
	}
	return strings.Join(words[:max], " ")
}

// SuggestedReading recommends material to consult. No specific articles could
// be identified without a live search, so the section carries an honest note
// and generic, verifiable reading guidance per input type — never a fabricated
// title or link.
func SuggestedReading(inputType classifier.InputType, entities []model.Entity) ([]model.SuggestedReading, string) {
	var who string
	for _, e := range entities {
		if e.Kind == model.EntityOrganization {
			who = e.Name
			break
		}
	}

	var items []model.SuggestedReading
	if who != "" {
		items = append(items, model.SuggestedReading{
			Title:      "Official statements from " + who,
			Publisher:  "Official sources",
			WhyItHelps: "A primary source can confirm or deny the claim authoritatively.",
		})
	}
	items = append(items, model.SuggestedReading{
		Title:      "Independent fact-checking of " + string(inputType) + " claims",
		Publisher:  "Independent journalism",
		WhyItHelps: "Independent outlets verify claims against multiple primary sources.",
	})
	if len(entities) > 0 {
		items = append(items, model.SuggestedReading{
			Title:      "Original reporting on " + entityNames(entities),
			Publisher:  "Original publications",
			WhyItHelps: "The original report may contain context missing from this submission.",
		})
	}

	note := "No specific articles could be identified for this input. " +
		"The suggestions above are search targets, not verified links."
	return items, note
}

func entityNames(entities []model.Entity) string {
	names := make([]string, 0, len(entities))
	for _, e := range entities {
		names = append(names, e.Name)
	}
	return strings.Join(names, ", ")
}

// WhatChanged reconstructs the dated evolution of the story. Only observed
// dates are ever included; this engine has no live source history, so it
// returns an explicit honest statement instead of a fabricated timeline.
func WhatChanged(result model.Result) ([]model.ChangeEvent, string) {
	return nil, "No reliable evidence found. " +
		"A dated story timeline requires tracking multiple sources over time, which this analysis does not perform."
}
