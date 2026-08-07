// Package claims extracts the main claim from a user input before any scoring
// happens. It identifies what is actually being asserted and the entities
// (organizations, people, locations, dates) and keywords that matter.
//
// The extractor is deliberately deterministic and heuristic (stdlib only). For
// free-form sentences it performs lightweight named-entity and keyword
// extraction; for structured identifiers (domains, emails, phones, ...) it
// produces the trustworthiness claim under review and extracts whatever entity
// the identifier refers to.
package claims

import (
	"regexp"
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
	"last year": true, "this year": true, "monday": true, "tuesday": true,
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
