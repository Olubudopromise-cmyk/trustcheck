package verifier

import (
	"regexp"
	"strings"
)

// phoneFormatRegex strips the formatting characters that may appear inside an
// E.164 number (spaces, dashes, dots, parentheses) while keeping the leading
// "+".
var phoneFormatRegex = regexp.MustCompile(`[\s.\-()]`)

// phoneE164Regex matches a normalized E.164 number: a leading "+" followed by
// 6-15 digits.
var phoneE164Regex = regexp.MustCompile(`^\+\d{6,15}$`)

// countryCodes maps an E.164 country calling code (without "+") to its display
// name.
var countryCodes = map[string]string{
	"1":   "USA/Canada",
	"44":  "United Kingdom",
	"234": "Nigeria",
	"91":  "India",
	"81":  "Japan",
	"61":  "Australia",
	"49":  "Germany",
	"33":  "France",
}

// countryCodeOrder lists the supported calling codes longest-first so the
// longest prefix wins (e.g. +234 must never be read as +2 or +23).
var countryCodeOrder = []string{"234", "44", "91", "81", "61", "49", "33", "1"}

// phoneVerifier implements the Verifier interface for phone inputs.
type phoneVerifier struct{}

// Verify validates and scores a phone number.
//
// Steps (Go standard library only):
//  1. Normalize: trim whitespace, then strip spaces, dashes, dots and
//     parentheses while keeping the leading "+".
//  2. Validate against E.164 (^\+\d{6,15}$); anything else => invalid / 0 /
//     "Invalid phone number.".
//  3. Detect the country from the calling-code prefix (longest match first).
//     Unknown prefixes stay valid but unrecognized.
//  4. Score: known country => verified / 80, unknown country => warning / 60.
func (phoneVerifier) Verify(input string) Result {
	// 1. Normalize.
	normalized := phoneFormatRegex.ReplaceAllString(strings.TrimSpace(input), "")

	// 2. Validate.
	if !phoneE164Regex.MatchString(normalized) {
		return Result{Status: "invalid", TrustScore: 0, Summary: "Invalid phone number."}
	}

	// 3. Country detection (drop the leading "+").
	if name, ok := detectCountry(normalized[1:]); ok {
		return Result{
			Status:     "verified",
			TrustScore: 80,
			Summary:    "Phone number format is valid (" + name + ").",
		}
	}

	// 4. Unknown country.
	return Result{
		Status:     "warning",
		TrustScore: 60,
		Summary:    "Phone number format is valid but country is unknown.",
	}
}

// detectCountry returns the display name of the country whose calling code is
// the longest prefix of digits.
func detectCountry(digits string) (string, bool) {
	for _, code := range countryCodeOrder {
		if strings.HasPrefix(digits, code) {
			return countryCodes[code], true
		}
	}
	return "", false
}
