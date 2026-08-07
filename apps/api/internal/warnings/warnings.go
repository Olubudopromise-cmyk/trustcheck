// Package warnings detects common misinformation indicators in an input and
// returns them as structured signals.
//
// Each signal has a label, a severity (high/medium/low), and a plain-English
// description. Detection is deterministic and heuristic (stdlib only): it looks
// for sensational and clickbait phrasing, emotional language, missing dates and
// citations, anonymous authorship, AI-generation patterns, manipulated
// statistics, and — for structured identifiers — deception-prone domain
// characteristics.
package warnings

import (
	"regexp"
	"strings"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

// Severity levels for warning signals.
const (
	SeverityHigh   = "high"
	SeverityMedium = "medium"
	SeverityLow    = "low"
)

var (
	// sensationalWords are capitalized, shock-value words typical of
	// sensational headlines.
	sensationalWords = regexp.MustCompile(`(?i)\b(shocking|breaking|urgent|exclusive|bombshell|explosive|unbelievable|outrageous|incredible)\b`)

	// clickbaitWords are phrases designed to drive clicks rather than inform.
	clickbaitWords = regexp.MustCompile(`(?i)\b(you won't believe|what happens next|doctors hate|number 1|one weird trick|secret|mind.blown|click here)\b`)

	// emotionalWords are strongly affective adjectives/adverbs.
	emotionalWords = regexp.MustCompile(`(?i)\b(amazing|terrible|horrifying|best|worst|astonishing|disgusting|heartbreaking|miraculous)\b`)

	// aiPatterns are phrasing patterns common in machine-generated text.
	aiPatterns = regexp.MustCompile(`(?i)\b(as an ai|as an ai language model|i cannot|i'm sorry, but i cannot|as a language model)\b`)

	// statPatterns match unsupported statistical claims.
	statPatterns = regexp.MustCompile(`(?i)\b(100% guaranteed|guaranteed|9 out of 10|double your|triple your|proven by science|clinically proven)\b`)

	// citationPhrases indicate that a source is referenced in the text.
	citationPhrases = regexp.MustCompile(`(?i)\b(according to|reported by|source|citation|study|research|official statement|evidence shows|cited)\b`)

	// dateTokens are inline markers that a date is present.
	dateTokens = regexp.MustCompile(`(?i)\b(january|february|march|april|may|june|july|august|september|october|november|december|yesterday|today|tomorrow)\b|\b(19[0-9]{2}|20[0-9]{2})\b`)

	// capsWord matches a fully uppercase word of 3+ letters.
	capsWord = regexp.MustCompile(`\b[A-Z]{3,}\b`)

	// unusualTLD matches deception-prone top-level domains.
	unusualTLD = regexp.MustCompile(`\.(xyz|top|click|link|buzz|loan|win|review|rest|tokyo|site|icu|info|online)$`)
)

// Detect returns the warning signals present in the input. For free-form text
// it runs the content-based detectors; for structured identifiers it checks
// deception-prone characteristics of the identifier itself.
func Detect(input string, inputType classifier.InputType) []model.WarningSignal {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil
	}

	var signals []model.WarningSignal
	if inputType == classifier.TypeUnknown {
		signals = detectText(trimmed)
	} else {
		signals = detectIdentifier(trimmed, inputType)
	}
	return signals
}

// detectText applies the content-based misinformation detectors.
func detectText(input string) []model.WarningSignal {
	var signals []model.WarningSignal
	hasCaps := capsWord.MatchString(input)
	hasSensational := sensationalWords.MatchString(input)

	if hasSensational || hasCaps {
		signals = append(signals, model.WarningSignal{
			Label:       "Sensational headline",
			Severity:    SeverityMedium,
			Description: "The headline uses sensational words or SHOUTED text to grab attention rather than inform.",
		})
	}

	if clickbaitWords.MatchString(input) {
		signals = append(signals, model.WarningSignal{
			Label:       "Clickbait wording",
			Severity:    SeverityMedium,
			Description: "The wording is engineered to generate clicks and shares instead of presenting facts.",
		})
	}

	if emotionalWords.MatchString(input) || strings.Count(input, "!") >= 2 {
		signals = append(signals, model.WarningSignal{
			Label:       "Excessive emotional language",
			Severity:    SeverityLow,
			Description: "Strongly emotional language can replace evidence and bias the reader's judgment.",
		})
	}

	if !dateTokens.MatchString(input) {
		signals = append(signals, model.WarningSignal{
			Label:       "No publication date",
			Severity:    SeverityMedium,
			Description: "No date is mentioned, so the information cannot be checked for recency or timeliness.",
		})
	}

	if !citationPhrases.MatchString(input) {
		signals = append(signals, model.WarningSignal{
			Label:       "No citations or sources",
			Severity:    SeverityHigh,
			Description: "The text does not reference any source, study, or authority, so claims cannot be independently verified.",
		})
	}

	if strings.Contains(strings.ToLower(input), "anonymous") ||
		strings.Contains(strings.ToLower(input), "unnamed sources") {
		signals = append(signals, model.WarningSignal{
			Label:       "Anonymous author",
			Severity:    SeverityMedium,
			Description: "The authorship is anonymous or attributed to unnamed sources, making accountability unclear.",
		})
	}

	if aiPatterns.MatchString(input) {
		signals = append(signals, model.WarningSignal{
			Label:       "Possible AI-generated pattern",
			Severity:    SeverityLow,
			Description: "The phrasing contains patterns typical of AI-generated text, which may lack editorial oversight.",
		})
	}

	if statPatterns.MatchString(input) {
		signals = append(signals, model.WarningSignal{
			Label:       "Manipulated statistics",
			Severity:    SeverityHigh,
			Description: "Statistics are stated as guarantees without citing the underlying data or study.",
		})
	}

	return signals
}

// detectIdentifier checks structured inputs for deception-prone characteristics.
func detectIdentifier(input string, inputType classifier.InputType) []model.WarningSignal {
	var signals []model.WarningSignal

	switch inputType {
	case classifier.TypeDomain, classifier.TypeURL:
		host := input
		if inputType == classifier.TypeURL {
			host = hostOf(input)
		}
		lower := strings.ToLower(host)

		if unusualTLD.MatchString(lower) {
			signals = append(signals, model.WarningSignal{
				Label:       "Unusual top-level domain",
				Severity:    SeverityMedium,
				Description: "The domain uses a TLD that is cheap and commonly associated with spam or scam sites.",
			})
		}

		body := strings.TrimSuffix(lower, tldSuffix(lower))
		if strings.Contains(body, "--") || strings.Count(body, "-") >= 3 ||
			regexp.MustCompile(`\d`).MatchString(body) {
			signals = append(signals, model.WarningSignal{
				Label:       "Deception-prone domain pattern",
				Severity:    SeverityLow,
				Description: "Excessive hyphens or digits in the domain name are a common pattern in scam domains.",
			})
		}
	}

	return signals
}

// hostOf returns the host of a URL without scheme and path.
func hostOf(rawURL string) string {
	i := strings.Index(rawURL, "://")
	if i >= 0 {
		rest := rawURL[i+3:]
		if j := strings.IndexAny(rest, "/?#"); j >= 0 {
			rest = rest[:j]
		}
		return rest
	}
	return ""
}

// tldSuffix returns the last dotted label of a host, if any.
func tldSuffix(host string) string {
	if i := strings.LastIndex(host, "."); i >= 0 {
		return host[i:]
	}
	return ""
}
