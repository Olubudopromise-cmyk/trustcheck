// Package claims extracts the main claim from a user input before any scoring
// happens. It identifies what is actually being asserted and the entities
// (organizations, people, locations, dates) and keywords that matter.
//
// The extractor is deliberately deterministic and heuristic (stdlib only). For
// free-form sentences it performs lightweight named-entity and keyword
// extraction; for structured identifiers (domains, emails, phones, ...) it
// produces the trustworthiness claim under review and extracts whatever entity
// the identifier refers to.
//
// Phase 13 adds ExtractMultiple, which splits long articles into 1-10
// independent factual claims, filters out opinions/jokes/predictions, merges
// duplicates, and detects claim relationships.
package claims

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
)

// Entity is a named thing (organization, person, location, date) extracted from
// the input. Kind is one of the EntityKind constants.
type Entity struct {
	Name string
	Kind string
}

// Entity kinds emitted by the claims extractor.
const (
	EntityOrganization = "organization"
	EntityLocation     = "location"
	EntityPerson       = "person"
	EntityDate         = "date"
)

// Claim is the extracted main claim plus its entities and keywords.
type Claim struct {
	MainClaim string
	Entities  []Entity
	Keywords  []string
}

// stopwords are common English function words never reported as keywords.
var stopwords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true, "was": true,
	"were": true, "be": true, "been": true, "being": true, "and": true,
	"or": true, "but": true, "of": true, "to": true, "in": true, "on": true,
	"at": true, "for": true, "from": true, "with": true, "by": true,
	"that": true, "this": true, "these": true, "those": true, "it": true,
	"its": true, "as": true, "has": true, "have": true, "had": true,
	"do": true, "does": true, "did": true, "not": true, "no": true,
	"so": true, "if": true, "then": true, "than": true, "they": true,
	"he": true, "she": true, "we": true, "you": true, "i": true, "me": true,
	"him": true, "her": true, "them": true, "us": true, "who": true,
	"what": true, "when": true, "where": true, "why": true, "how": true,
	"about": true, "after": true, "before": true, "between": true,
	"during": true, "into": true, "out": true, "over": true, "under": true,
	"up": true, "down": true, "all": true, "any": true, "both": true,
	"each": true, "few": true, "more": true, "most": true, "other": true,
	"some": true, "such": true, "only": true, "own": true, "same": true,
	"too": true, "very": true, "just": true, "will": true,
	"can": true, "could": true, "would": true, "should": true, "may": true,
	"might": true, "must": true, "says": true, "said": true, "say": true,
	"confirm": true, "confirms": true, "confirmed": true, "announces": true,
	"announced": true, "report": true, "reports": true, "reported": true,
}

// knownOrganizations is a small dictionary used for entity extraction. It is
// intentionally partial: the extractor also relies on capitalization heuristics
// so unknown proper nouns are still caught as entities.
var knownOrganizations = map[string]bool{
	"nasa": true, "who": true, "un": true, "eu": true, "bbc": true,
	"cnn": true, "reuters": true, "google": true, "facebook": true,
	"meta": true, "microsoft": true, "apple": true, "amazon": true,
	"openai": true, "tesla": true, "spacex": true, "stripe": true,
	"paystack": true, "flutterwave": true, "instagram": true, "twitter": true,
	"x": true, "netflix": true, "spotify": true, "uber": true, "netlify": true,
	"vercel": true, "github": true, "youtube": true, "reddit": true,
}

// knownLocations is a small dictionary of major cities and countries used for
// entity extraction.
var knownLocations = map[string]bool{
	"lagos": true, "abuja": true, "london": true, "paris": true, "berlin": true,
	"new york": true, "newyork": true, "washington": true, "los angeles": true,
	"chicago": true, "toronto": true, "sydney": true, "tokyo": true, "beijing": true,
	"shanghai": true, "moscow": true, "kolkata": true, "mumbai": true, "delhi": true,
	"nairobi": true, "accra": true, "johannesburg": true, "cairo": true,
	"nigeria": true, "kenya": true, "ghana": true, "south africa": true,
	"usa": true, "us": true, "uk": true, "united states": true,
	"united kingdom": true, "canada": true, "china": true, "india": true,
	"france": true, "germany": true, "brazil": true, "australia": true,
	"europe": true, "africa": true, "asia": true,
}

// months are used to recognize date mentions.
var months = map[string]bool{
	"january": true, "february": true, "march": true, "april": true,
	"may": true, "june": true, "july": true, "august": true,
	"september": true, "october": true, "november": true, "december": true,
}

// dateWords are relative date references.
var dateWords = map[string]bool{
	"yesterday": true, "today": true, "tomorrow": true, "tonight": true,
	"last week": true, "this week": true, "last month": true, "this month": true,
	"last year": true, "this year": true, "next week": true, "next month": true,
	"next year": true, "monday": true, "tuesday": true,
	"wednesday": true, "thursday": true, "friday": true, "saturday": true,
	"sunday": true,
}

var (
	// yearRegex matches a 4-digit year in the plausible range 1900-2099.
	yearRegex = regexp.MustCompile(`\b(19[0-9]{2}|20[0-9]{2})\b`)
	// titleCaseRegex matches a capitalized word (title case), used to catch
	// proper nouns that are not in the known dictionaries.
	titleCaseRegex = regexp.MustCompile(`\b[A-Z][a-z]+\b`)
	// multiWordDate is matched against the raw text to catch phrases like
	// "last week" as a single date entity.
	multiWordDate = regexp.MustCompile(`\b(last|this) (week|month|year|weekend)\b`)
)

// Extract returns the main claim, entities, and keywords for the given input.
// structured identifies identifier-style inputs (domain, url, email, phone,
// ip, company); anything else is treated as free-form text.
func Extract(input string, inputType classifier.InputType) Claim {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return Claim{MainClaim: "No claim could be extracted from empty input."}
	}

	if inputType != classifier.TypeUnknown {
		return extractIdentifier(trimmed, inputType)
	}
	return extractText(trimmed)
}

// MultiClaim is a single extracted factual claim with its metadata.
// It extends Claim with an ID, original position in the text, and
// relationship information.
type MultiClaim struct {
	ID          string
	Text        string
	Entities    []Entity
	Keywords    []string
	Position    int    // sentence index in the original text
	DependsOn   []string // IDs of claims this one depends on
}

// ExtractMultiple splits the input into 1-10 independent factual claims.
// For structured inputs (domain, email, etc.) it returns a single claim.
// For free-form text it splits into sentences, filters for factual claims,
// merges duplicates, and detects dependencies.
func ExtractMultiple(input string, inputType classifier.InputType) []MultiClaim {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return []MultiClaim{{
			ID:   "claim-0",
			Text: "No claim could be extracted from empty input.",
		}}
	}

	// Structured inputs always produce exactly one claim.
	if inputType != classifier.TypeUnknown {
		single := extractIdentifier(trimmed, inputType)
		return []MultiClaim{{
			ID:       "claim-0",
			Text:     single.MainClaim,
			Entities: single.Entities,
			Keywords: single.Keywords,
		}}
	}

	// Split into sentences.
	sentences := splitSentences(trimmed)
	if len(sentences) == 0 {
		return []MultiClaim{{
			ID:   "claim-0",
			Text: normalizeSentence(trimmed),
		}}
	}

	// Filter for factual claims and extract entities/keywords.
	var candidates []MultiClaim
	for i, sent := range sentences {
		if !isFactualClaim(sent) {
			continue
		}
		claim := extractText(sent)
		mc := MultiClaim{
			ID:       fmt.Sprintf("claim-%d", len(candidates)),
			Text:     claim.MainClaim,
			Entities: claim.Entities,
			Keywords: claim.Keywords,
			Position: i,
		}
		mc.Keywords = claim.Keywords
		candidates = append(candidates, mc)
	}

	// If no factual claims found, fall back to the full text as one claim.
	if len(candidates) == 0 {
		claim := extractText(trimmed)
		return []MultiClaim{{
			ID:       "claim-0",
			Text:     claim.MainClaim,
			Entities: claim.Entities,
			Keywords: claim.Keywords,
		}}
	}

	// Cap at 10 claims.
	if len(candidates) > 10 {
		candidates = candidates[:10]
	}

	// Merge duplicates.
	candidates = mergeDuplicates(candidates)

	// Detect dependencies.
	candidates = detectDependencies(candidates)

	return candidates
}

// splitSentences breaks text into individual sentences.
func splitSentences(text string) []string {
	// Split on sentence-ending punctuation followed by whitespace or end.
	re := regexp.MustCompile(`[.!?]+\s+`)
	parts := re.Split(text, -1)

	var sentences []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		sentences = append(sentences, normalizeSentence(p))
	}
	return sentences
}

// isFactualClaim reports whether a sentence is a verifiable factual claim
// rather than an opinion, joke, prediction, rhetorical question, or
// advertisement.
func isFactualClaim(sentence string) bool {
	lower := strings.ToLower(sentence)
	trimmed := strings.TrimSpace(sentence)

	// Too short to be a meaningful claim.
	if len(trimmed) < 10 {
		return false
	}

	// Skip rhetorical questions.
	if strings.HasSuffix(trimmed, "?") {
		return false
	}

	// Skip predictions and future tense statements.
	predictionPrefixes := []string{
		"will", "could", "would", "should", "might", "may",
		"expect", "predict", "forecast", "anticipate",
		"plan to", "going to", "about to",
	}
	for _, prefix := range predictionPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}

	// Skip opinions and subjective statements.
	opinionMarkers := []string{
		"i think", "i believe", "in my opinion", "i feel",
		"it seems", "it appears", "arguably", "supposedly",
		"beautiful", "ugly", "best", "worst", "amazing",
		"terrible", "awesome", "horrible", "love", "hate",
		"favorite", "prefer", "nice", "good", "bad",
		"like", "dislike", "enjoy", "prefer", "tasty",
		"boring", "exciting", "fun", "enjoyable",
	}
	for _, marker := range opinionMarkers {
		if strings.Contains(lower, marker) {
			return false
		}
	}

	// Skip jokes and satire indicators.
	jokeMarkers := []string{
		"just kidding", "jk", "lol", "lmao", "haha",
		"funny", "joke", "satire", "satirical",
		"the onion", "theonion",
	}
	for _, marker := range jokeMarkers {
		if strings.Contains(lower, marker) {
			return false
		}
	}

	// Skip advertisements.
	adMarkers := []string{
		"buy now", "click here", "limited time", "act now",
		"discount", "coupon", "promo code", "free shipping",
		"subscribe", "sign up now",
	}
	for _, marker := range adMarkers {
		if strings.Contains(lower, marker) {
			return false
		}
	}

	// Skip personal feelings and emotions.
	feelingMarkers := []string{
		"i am happy", "i am sad", "i am angry", "i am excited",
		"i am worried", "i am concerned",
	}
	for _, marker := range feelingMarkers {
		if strings.Contains(lower, marker) {
			return false
		}
	}

	// Must contain at least one entity or be a concrete assertion.
	// A factual claim typically mentions a named entity, a number,
	// or a date.
	// commonWords are capitalized words that are NOT real entities
	// (they appear at sentence start due to capitalization rules).
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "this": true, "that": true,
		"these": true, "those": true, "it": true, "its": true, "they": true,
		"their": true, "there": true, "here": true, "where": true, "when": true,
		"how": true, "what": true, "which": true, "who": true, "whom": true,
		"but": true, "and": true, "or": true, "if": true, "so": true,
		"yet": true, "for": true, "nor": true, "not": true, "also": true,
		"just": true, "only": true, "even": true, "still": true, "already": true,
	}
	hasEntity := false
	words := strings.Fields(sentence)
	for _, w := range words {
		clean := strings.ToLower(trimPunct(w))
		if knownOrganizations[clean] || knownLocations[clean] ||
			months[clean] || dateWords[clean] {
			hasEntity = true
			break
		}
		if titleCaseRegex.MatchString(w) && len(clean) > 1 && !commonWords[clean] {
			hasEntity = true
			break
		}
	}
	if !hasEntity {
		// Check for numbers (statistics, dates, quantities).
		hasNumber := false
		for _, w := range words {
			if _, err := strconv.Atoi(trimPunct(w)); err == nil {
				hasNumber = true
				break
			}
		}
		if !hasNumber {
			return false
		}
	}

	return true
}

// mergeDuplicates merges claims that are substantively the same.
// Two claims are considered duplicates if they share >60% of their keywords.
func mergeDuplicates(claims []MultiClaim) []MultiClaim {
	if len(claims) <= 1 {
		return claims
	}

	merged := make([]MultiClaim, 0, len(claims))
	used := make([]bool, len(claims))

	for i := range claims {
		if used[i] {
			continue
		}
		base := claims[i]
		for j := i + 1; j < len(claims); j++ {
			if used[j] {
				continue
			}
			if areSimilar(base, claims[j]) {
				used[j] = true
				// Keep the longer, more detailed version.
				if len(claims[j].Text) > len(base.Text) {
					base = claims[j]
				}
			}
		}
		merged = append(merged, base)
	}

	// Re-number IDs.
	for i := range merged {
		merged[i].ID = fmt.Sprintf("claim-%d", i)
	}
	return merged
}

// areSimilar reports whether two claims are substantively the same.
func areSimilar(a, b MultiClaim) bool {
	if len(a.Keywords) == 0 || len(b.Keywords) == 0 {
		return false
	}

	setB := make(map[string]bool, len(b.Keywords))
	for _, k := range b.Keywords {
		setB[k] = true
	}

	shared := 0
	for _, k := range a.Keywords {
		if setB[k] {
			shared++
		}
	}

	// Threshold: share >60% of the smaller keyword set.
	minLen := len(a.Keywords)
	if len(b.Keywords) < minLen {
		minLen = len(b.Keywords)
	}
	return shared > 0 && float64(shared)/float64(minLen) > 0.6
}

// detectDependencies identifies claims that depend on other claims.
// A claim depends on another if it references a pronoun or entity from
// the earlier claim, or if it uses connectors like "because", "therefore",
// "as a result", "due to".
func detectDependencies(claims []MultiClaim) []MultiClaim {
	if len(claims) <= 1 {
		return claims
	}

	for i := range claims {
		if i == 0 {
			continue
		}
		lower := strings.ToLower(claims[i].Text)

		// Check for dependency connectors.
		dependencyConnectors := []string{
			"because", "therefore", "as a result", "due to",
			"consequently", "thus", "hence", "since",
			"after", "following", "subsequently",
		}
		for _, conn := range dependencyConnectors {
			if strings.Contains(lower, conn) {
				// Depend on the immediately preceding claim.
				claims[i].DependsOn = []string{claims[i-1].ID}
				break
			}
		}

		// Check for pronoun references to entities in previous claims.
		pronouns := []string{"it", "its", "they", "them", "their", "this", "that"}
		for _, pronoun := range pronouns {
			if strings.Contains(" "+lower+" ", " "+pronoun+" ") {
				// Find the previous claim that has entities.
				for j := i - 1; j >= 0; j-- {
					if len(claims[j].Entities) > 0 {
						claims[i].DependsOn = []string{claims[j].ID}
						break
					}
				}
				break
			}
		}
	}

	return claims
}

// extractIdentifier builds the trustworthiness claim for a structured input
// and surfaces the entity it refers to.
func extractIdentifier(input string, inputType classifier.InputType) Claim {
	claim := Claim{MainClaim: trustClaimFor(inputType, input)}

	switch inputType {
	case classifier.TypeCompany:
		claim.Entities = []Entity{{Name: input, Kind: EntityOrganization}}
	case classifier.TypeDomain:
		claim.Entities = []Entity{{Name: input, Kind: EntityOrganization}}
	case classifier.TypeURL:
		if host := hostOf(input); host != "" {
			claim.Entities = []Entity{{Name: host, Kind: EntityOrganization}}
		}
	case classifier.TypeEmail:
		if domain := domainOf(input); domain != "" {
			claim.Entities = []Entity{{Name: domain, Kind: EntityOrganization}}
		}
	case classifier.TypePhone:
		claim.Entities = nil
	}
	return claim
}

// trustClaimFor renders the generic trustworthiness claim for an identifier.
func trustClaimFor(inputType classifier.InputType, input string) string {
	label := string(inputType)
	if inputType == classifier.TypeCompany {
		label = "company"
	}
	return "The claim under review is that " + label + " \"" + input + "\" is legitimate and trustworthy."
}

// extractText performs entity and keyword extraction on free-form sentences.
func extractText(input string) Claim {
	words := strings.Fields(input)
	claim := Claim{MainClaim: normalizeSentence(input)}

	var entities []Entity
	seenEntity := map[string]bool{}

	addEntity := func(name, kind string) {
		key := strings.ToLower(name)
		if seenEntity[key] {
			return
		}
		seenEntity[key] = true
		entities = append(entities, Entity{Name: name, Kind: kind})
	}

	lower := strings.ToLower(input)

	// Multi-word locations and dates first, so they are not split apart.
	for _, phrase := range []string{"new york", "united states", "united kingdom",
		"south africa", "los angeles", "last week", "this week", "last month",
		"this month", "last year", "this year"} {
		if strings.Contains(lower, phrase) {
			if phrase == "last week" || phrase == "this week" || phrase == "last month" ||
				phrase == "this month" || phrase == "last year" || phrase == "this year" {
				addEntity(phrase, EntityDate)
			} else {
				addEntity(strings.Title(phrase), EntityLocation)
			}
		}
	}

	// Single-word entities via dictionaries + capitalization heuristic.
	for i, w := range words {
		clean := strings.ToLower(strings.Trim(w, ".,;:!?\"'()"))
		if clean == "" {
			continue
		}
		switch {
		case knownOrganizations[clean]:
			addEntity(w, EntityOrganization)
		case knownLocations[clean]:
			addEntity(w, EntityLocation)
		case months[clean] || dateWords[clean]:
			addEntity(clean, EntityDate)
		}
		if titleCaseRegex.MatchString(w) {
			// A capitalized word at sentence start that is not a known
			// entity, stopword, or dictionary word is likely a proper noun.
			if i > 0 && !stopwords[clean] &&
				!knownOrganizations[clean] && !knownLocations[clean] &&
				!months[clean] && !dateWords[clean] &&
				len(clean) > 1 {
				addEntity(w, EntityPerson)
			}
		}
	}

	// Year mentions are date entities.
	for _, m := range yearRegex.FindAllString(input, -1) {
		addEntity(m, EntityDate)
	}

	claim.Entities = entities
	claim.Keywords = keywordsFor(words, seenEntity)
	return claim
}

// keywordsFor returns the non-stopword, non-entity words that carry meaning,
// lightly stemmed so plural and past/present tense variants collapse together.
func keywordsFor(words []string, entitySet map[string]bool) []string {
	var keywords []string
	seen := map[string]bool{}
	for _, w := range words {
		clean := strings.ToLower(strings.Trim(w, ".,;:!?\"'()"))
		if clean == "" || stopwords[clean] || entitySet[clean] {
			continue
		}
		stem := stem(clean)
		if len(stem) < 3 || seen[stem] {
			continue
		}
		seen[stem] = true
		keywords = append(keywords, stem)
	}
	return keywords
}

// stem applies a minimal English suffix reduction so related keywords collapse.
func stem(word string) string {
	switch {
	case strings.HasSuffix(word, "ies") && len(word) > 4:
		return word[:len(word)-3] + "y"
	case strings.HasSuffix(word, "es") && len(word) > 3:
		return word[:len(word)-2]
	case strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") && len(word) > 3:
		return word[:len(word)-1]
	case strings.HasSuffix(word, "ing") && len(word) > 5:
		return word[:len(word)-3]
	case strings.HasSuffix(word, "ed") && len(word) > 4:
		return word[:len(word)-2]
	default:
		return word
	}
}

// trimPunct strips leading and trailing punctuation from a word.
func trimPunct(w string) string {
	return strings.TrimFunc(w, func(r rune) bool {
		switch r {
		case '.', ',', ';', ':', '!', '?', '\'', '"', '(', ')':
			return true
		}
		return false
	})
}

// normalizeSentence collapses whitespace and strips trailing punctuation so
// the main claim is a clean sentence.
func normalizeSentence(input string) string {
	s := strings.Join(strings.Fields(input), " ")
	s = strings.TrimRight(s, ".,;:!?")
	if s != "" && !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s
}

// hostOf returns the host of a URL without the scheme and path.
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

// domainOf returns the domain part of an email address.
func domainOf(email string) string {
	if i := strings.LastIndex(email, "@"); i >= 0 {
		return email[i+1:]
	}
	return ""
}
