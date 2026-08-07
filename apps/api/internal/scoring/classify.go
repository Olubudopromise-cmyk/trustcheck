// Shared evidence classification: the single place that decides which scored
// checks count as supporting, contradicting, or neutral evidence, and that
// filters out generic engine fallback messages. Every consumer (the pipeline,
// the timeline, the perspectives sections, the API response) derives its
// evidence from this one definition so the analysis can never disagree with
// itself.
package scoring

import "github.com/pamierin/trustcheck/apps/api/internal/model"

// EnginePlaceholderLabels are generic engine messages emitted when a heuristic
// has nothing concrete to report ("I could not suggest anything") rather than
// evidence about the input. They are filtered out before any consumer sees
// them so a placeholder is never presented as a fact, a pass, or a conflict.
var EnginePlaceholderLabels = map[string]bool{
	"No Suggestion":         true,
	"Suggestion Generated":  true,
	"Input Provided":        true,
}

// IsEnginePlaceholder reports whether a check label is a generic engine
// fallback rather than evidence about the claim.
func IsEnginePlaceholder(label string) bool {
	return EnginePlaceholderLabels[label]
}

// EvidenceSet is the single classification of scored evidence into supporting,
// contradicting, and neutral buckets. One call to ClassifyEvidence produces
// every list a downstream consumer needs, so counts can never drift apart.
type EvidenceSet struct {
	Supporting    []model.EvidenceItem
	Contradicting []model.EvidenceItem
	Neutral       []model.EvidenceItem
}

// ClassifyEvidence buckets scored checks into supporting, contradicting, and
// neutral evidence. Engine placeholder labels are dropped first: they are not
// evidence about the claim. A pass supports the claim; a fail or warning
// contradicts it; an info check is neutral (it carries no score weight).
func ClassifyEvidence(evidence []Evidence) EvidenceSet {
	var set EvidenceSet
	for _, e := range evidence {
		if IsEnginePlaceholder(e.Label) {
			continue
		}
		item := model.EvidenceItem{Label: e.Label, Result: e.Result, Points: e.Points}
		switch e.Result {
		case "pass":
			set.Supporting = append(set.Supporting, item)
		case "fail", "warning":
			set.Contradicting = append(set.Contradicting, item)
		case "info":
			set.Neutral = append(set.Neutral, item)
		}
	}
	return set
}

// FilterEnginePlaceholders returns the evidence with generic engine fallback
// messages removed, preserving order. It is used where the raw scored checks
// are exposed (the legacy /verify "evidence" array) so placeholders never
// surface anywhere.
func FilterEnginePlaceholders(evidence []Evidence) []Evidence {
	filtered := make([]Evidence, 0, len(evidence))
	for _, e := range evidence {
		if !IsEnginePlaceholder(e.Label) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
