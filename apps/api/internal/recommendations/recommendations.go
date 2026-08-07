// Package recommendations suggests actionable next steps for the user after an
// model. Recommendations are tailored to the input type and the severity of
// the verdict, so a suspicious domain gets different advice from a free-form
// news claim.
package recommendations

import (
	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

// baseByType returns the recommendations that apply to a given input type.
func baseByType(inputType classifier.InputType) []model.Recommendation {
	switch inputType {
	case classifier.TypeDomain:
		return []model.Recommendation{
			{Title: "Check WHOIS registration", Description: "Look up who registered the domain and when. Recently registered domains are riskier."},
			{Title: "Run a link scanner", Description: "Scan the domain with a service like VirusTotal or Google Safe Browsing."},
			{Title: "Compare with the official site", Description: "Visit the organization's official website to confirm this domain is genuine."},
		}
	case classifier.TypeURL:
		return []model.Recommendation{
			{Title: "Scan the link before opening", Description: "Use a URL scanner to check the destination for malware or phishing before visiting."},
			{Title: "Check the domain registration", Description: "Verify the underlying domain's age and ownership."},
			{Title: "Preview the destination", Description: "Use a URL previewer to see where the link really points."},
		}
	case classifier.TypeEmail:
		return []model.Recommendation{
			{Title: "Verify through a second channel", Description: "Contact the sender through an official channel you already trust, not via the email."},
			{Title: "Check breach databases", Description: "Search the domain or address in breach databases to see if it is associated with leaks."},
			{Title: "Watch for spoofing", Description: "Compare the display name with the actual sending address — spoofed emails often mismatch."},
		}
	case classifier.TypePhone:
		return []model.Recommendation{
			{Title: "Check with the official operator", Description: "Confirm the number against the official contact number of the organization it claims to be."},
			{Title: "Beware of vishing", Description: "Never share passwords or OTPs over the phone; scam callers impersonate support teams."},
			{Title: "Search the number", Description: "A quick search may reveal whether the number is reported as a scam."},
		}
	case classifier.TypeCompany:
		return []model.Recommendation{
			{Title: "Check official registries", Description: "Verify the company in the official business registry for its country of operation."},
			{Title: "Look for real reviews", Description: "Search independent review platforms for consistent, verifiable customer feedback."},
			{Title: "Confirm the website", Description: "Make sure the company's website uses HTTPS and matches its registered name."},
		}
	default:
		return []model.Recommendation{
			{Title: "Compare with official sources", Description: "Check the claim against official government, regulatory, or organizational sources."},
			{Title: "Search independent news outlets", Description: "See whether credible news organizations independently report the same thing."},
			{Title: "Verify quoted statistics", Description: "Find the original study or dataset behind any numbers and confirm they are quoted accurately."},
			{Title: "Check the publication date", Description: "Old or recycled news is often recirculated out of context."},
			{Title: "Read the original study or article", Description: "Headlines and summaries can misrepresent the source material."},
		}
	}
}

// Generate returns the recommended next steps for a verdict. The base
// recommendations for the input type are always returned; when the verdict is
// not High, an extra verification reminder is prepended.
func Generate(inputType classifier.InputType, verdict model.Verdict) []model.Recommendation {
	recs := baseByType(inputType)

	if verdict != model.VerdictHigh {
		recs = append([]model.Recommendation{{
			Title:       "Treat with caution",
			Description: "The assessment is not fully positive — verify before acting or sharing.",
		}}, recs...)
	}

	return recs
}
