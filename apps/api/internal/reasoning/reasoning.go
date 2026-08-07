// Package reasoning explains, in plain bullets, why a trust score is what it
// is. It turns the supporting and contradicting evidence into a readable
// account so users understand exactly what moved the score up and down.
package reasoning

import (
	"fmt"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

// Explain returns the ordered list of reasoning bullets for a score.
//
// The first bullet states the score; subsequent bullets list the supporting
// checks ("+ ...") and the contradicting checks ("- ..."). When checks could
// not be run the explanation says so explicitly rather than inventing evidence.
func Explain(score int, inputType classifier.InputType, evidenceFor, evidenceAgainst []model.EvidenceItem) []string {
	reasoning := []string{
		fmt.Sprintf("Trust score of %d out of 100 reflects the checks that could be run against the input.", score),
	}

	for _, e := range evidenceFor {
		reasoning = append(reasoning, "+ "+e.Label)
	}
	for _, e := range evidenceAgainst {
		reasoning = append(reasoning, "- "+e.Label)
	}

	if len(evidenceAgainst) == 0 && len(evidenceFor) > 0 {
		reasoning = append(reasoning, "No contradicting evidence was found.")
	}
	if len(evidenceFor) == 0 && len(evidenceAgainst) == 0 {
		reasoning = append(reasoning, "No concrete evidence could be gathered, so the score is provisional.")
	}
	if inputType == classifier.TypeUnknown {
		reasoning = append(reasoning, "The input is unstructured text; the score is based on the wording alone, not verifiable sources.")
	}

	return reasoning
}
