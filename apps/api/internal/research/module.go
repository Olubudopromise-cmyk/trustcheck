package research

import (
	"context"
	"fmt"
	"strings"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

// Module is a pluggable extension that performs research for text claims.
// It searches for evidence online and adds it to the result.
type Module struct {
	engine *ResearchEngine
}

// NewModule creates a new research module with the given provider and config.
func NewModule(provider SearchProvider, config SearchConfig) *Module {
	return &Module{
		engine: NewResearchEngine(provider, config),
	}
}

// Name returns the module name for logging.
func (m *Module) Name() string {
	return "research"
}

// Enrich searches for evidence online and adds it to the result.
// It only runs for text claims (unknown input type) and skips domain/URL/email/etc.
func (m *Module) Enrich(ctx context.Context, input string, result *model.Result) error {
	// Only search for text claims, not for domains/URLs/etc.
	if result.Type != string(classifier.TypeUnknown) {
		return nil
	}


	// Extract keywords from the claim
	keywords := extractKeywords(input)

	// Perform research
	evidence := m.engine.Research(ctx, input, keywords)

	// Handle search failure honestly
	switch evidence.SearchStatus {
	case SearchStatusFailed:
		result.WarningSignals = append(result.WarningSignals, model.WarningSignal{
			Label:       "Research failed",
			Severity:    "high",
			Description: "Unable to perform web research. The claim could not be verified.",
		})
		// Don't fabricate evidence - just report the failure
		return nil

	case SearchStatusNoResults:
		result.WarningSignals = append(result.WarningSignals, model.WarningSignal{
			Label:       "No evidence found",
			Severity:    "medium",
			Description: "No web search results were found for this claim. This does not mean the claim is false - it means we couldn't find evidence.",
		})
		// Don't convert lack of evidence into proof that the claim is false
		return nil

	case SearchStatusPartial:
		// Some searches failed, but we have partial results
		if len(evidence.SearchErrors) > 0 {
			result.WarningSignals = append(result.WarningSignals, model.WarningSignal{
				Label:       "Partial research",
				Severity:    "low",
				Description: fmt.Sprintf("Some searches failed. Found %d sources.", evidence.TotalSources),
			})
		}
	}

	// Add evidence to result
	for _, r := range evidence.Supporting {
		evidenceItem := model.EvidenceItem{
			Label:  fmt.Sprintf("Web search: %s", r.Domain),
			Result: "info",
			Points: calculateEvidencePoints(r, true),
			Note:   formatEvidenceNote(r, true),
		}
		result.EvidenceFor = append(result.EvidenceFor, evidenceItem)
	}

	for _, r := range evidence.Contradicting {
		evidenceItem := model.EvidenceItem{
			Label:  fmt.Sprintf("Web search: %s", r.Domain),
			Result: "warning",
			Points: calculateEvidencePoints(r, false),
			Note:   formatEvidenceNote(r, false),
		}
		result.EvidenceAgainst = append(result.EvidenceAgainst, evidenceItem)
	}

	// Add source intelligence
	for _, r := range evidence.Supporting {
		result.SourceIntelligence = append(result.SourceIntelligence, model.SourceIntelligence{
			Title:           r.Title,
			Domain:          r.Domain,
			SourceType:      convertSourceType(r.SourceType),
			Relation:        classifyRelation(r),
			IsOfficial:      r.IsOfficial,
			SupportsClaim:   true,
			ContradictsClaim: false,
			IsIndependent:   !r.IsOfficial,
			Confidence:      r.Confidence,
		})
	}

	for _, r := range evidence.Contradicting {
		result.SourceIntelligence = append(result.SourceIntelligence, model.SourceIntelligence{
			Title:           r.Title,
			Domain:          r.Domain,
			SourceType:      convertSourceType(r.SourceType),
			Relation:        classifyRelation(r),
			IsOfficial:      r.IsOfficial,
			SupportsClaim:   false,
			ContradictsClaim: true,
			IsIndependent:   !r.IsOfficial,
			Confidence:      r.Confidence,
		})
	}

	// Update confidence based on evidence quality
	if evidence.TotalSources > 0 {
		// More sources = higher confidence
		confidenceBoost := min(evidence.TotalSources*3, 15)
		result.Confidence = min(100, result.Confidence+confidenceBoost)

		// Official sources boost confidence more
		officialCount := 0
		for _, r := range evidence.Supporting {
			if r.IsOfficial {
				officialCount++
			}
		}
		if officialCount > 0 {
			result.Confidence = min(100, result.Confidence+10)
		}
	}

	// Drive the score based on evidence
	// This is the key change: evidence now drives the score
	if len(evidence.Supporting) > 0 || len(evidence.Contradicting) > 0 {
		// Calculate score from evidence
		supportScore := len(evidence.Supporting) * 15
		contradictScore := len(evidence.Contradicting) * 15

		// Start from a neutral base (50) and adjust based on evidence
		newScore := 50

		if supportScore > contradictScore {
			// More supporting evidence - increase score
			newScore = 50 + min(supportScore-contradictScore, 40)
		} else if contradictScore > supportScore {
			// More contradicting evidence - decrease score
			newScore = 50 - min(contradictScore-supportScore, 40)
		}
		// If equal, stay at 50 (neutral)

		// Adjust score if we have any evidence (not just 2+)
		result.TrustScore = max(0, min(100, newScore))
	} else {
		// No evidence found - keep the base score (10 for unknown input)
		// This honestly reflects that we couldn't verify the claim
	}

	return nil
}

// extractKeywords extracts keywords from a claim.
func extractKeywords(claim string) []string {
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "can": true, "shall": true, "to": true,
		"of": true, "in": true, "for": true, "on": true, "with": true,
		"at": true, "by": true, "from": true, "as": true, "into": true,
		"through": true, "during": true, "before": true, "after": true,
		"and": true, "but": true, "or": true, "nor": true, "not": true,
		"so": true, "yet": true, "both": true, "either": true, "neither": true,
		"each": true, "every": true, "all": true, "any": true, "few": true,
		"more": true, "most": true, "other": true, "some": true, "such": true,
		"no": true, "only": true, "own": true, "same": true, "than": true,
		"too": true, "very": true, "just": true, "because": true, "if": true,
		"when": true, "where": true, "how": true, "what": true, "which": true,
		"who": true, "whom": true, "this": true, "that": true, "these": true,
		"those": true, "i": true, "you": true, "he": true, "she": true,
		"it": true, "we": true, "they": true, "me": true, "him": true,
		"her": true, "us": true, "them": true, "my": true, "your": true,
		"his": true, "its": true, "our": true, "their": true, "mine": true,
		"yours": true, "hers": true, "ours": true, "theirs": true,
	}

	words := strings.Fields(strings.ToLower(claim))
	var keywords []string
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:\"'()[]{}")
		if len(word) > 2 && !commonWords[word] {
			keywords = append(keywords, word)
		}
	}
	return keywords
}

// calculateEvidencePoints calculates the points for an evidence item.
func calculateEvidencePoints(r SearchResult, supports bool) int {
	base := 5
	if r.IsOfficial {
		base += 5
	}
	if r.IsAcademic {
		base += 3
	}
	if r.IsNews {
		base += 2
	}
	return base
}

// formatEvidenceNote formats a note for an evidence item.
func formatEvidenceNote(r SearchResult, supports bool) string {
	note := r.Title
	if r.Snippet != "" {
		note += " - " + truncate(r.Snippet, 100)
	}
	if r.PublicationDate != "" {
		note += " (" + r.PublicationDate + ")"
	}
	return note
}

// truncate truncates a string to a maximum length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// convertSourceType converts our SourceType to the model SourceType.
func convertSourceType(st SourceType) model.SourceType {
	switch st {
	case SourceTypeOfficial:
		return model.SourceOfficial
	case SourceTypeAcademic:
		return model.SourceAcademic
	case SourceTypeJournalism:
		return model.SourceJournalism
	case SourceTypeFactChecker:
		return model.SourceJournalism // Closest match
	case SourceTypeCommunity:
		return model.SourceCommunity
	case SourceTypeCommercial:
		return model.SourceCommercial
	default:
		return model.SourceUnknown
	}
}

// classifyRelation determines if a source is primary, secondary, or tertiary.
func classifyRelation(r SearchResult) model.SourceRelation {
	if r.IsOfficial {
		return model.RelationPrimary
	}
	if r.IsNews {
		return model.RelationSecondary
	}
	return model.RelationTertiary
}
