// Package scoring centralizes the trust-score math shared by every verifier.
//
// A verifier collects observations, accumulates them with a Builder (which
// clamps automatically), and derives the final status from the score. Every
// score value lives here as a named constant so no verifier hardcodes a
// number, and clamping lives in exactly one place.
package scoring

// Status band thresholds used by StatusFromScore.
const (
	VerifiedThreshold = 90
	GoodThreshold     = 70
	WarningThreshold  = 50
	PoorThreshold     = 20
)

// HighConfidenceScore is the "verified" threshold used by the email and
// company engines. It predates the five-band StatusFromScore mapping and is
// kept constant so those engines do not change observable behavior.
const HighConfidenceScore = 80

// Domain and URL scoring values.
const (
	UnreachableScore      = 15
	HTTPSBonus            = 20 // HTTPS available (domain), verified HTTPS (company)
	ResolveBonus          = 20 // domain resolves (company)
	ValidCertBonus        = 20 // valid certificate present (domain)
	ExpiredCertPenalty    = 30
	HTTPFallbackBonus     = 10
	StatusOKBonus         = 20 // 2xx-3xx responses
	StatusClientBonus     = 10 // 4xx responses
	StatusServerBonus     = 0  // 5xx and everything else
	URLBaseScore          = 40 // valid URL + host resolves + request OK
	URLValidCertBonus     = 10
	TLSBonus              = 15
	TLSDowngradePenalty   = 10
	RedirectPenalty       = 10
	MaxRedirectPenalty    = 20
	URLRequestFailedScore = 25
)

// Email scoring values.
const (
	MXBonus           = 40
	ARecordBonus      = 20
	DisposablePenalty = 40
)

// IP scoring values.
const (
	LoopbackBonus      = 100
	PrivateIPBonus     = 90
	LinkLocalBonus     = 80
	GlobalUnicastBonus = 70
	MulticastScore     = 50
)

// Phone scoring values.
const (
	PhoneVerifiedScore = 80
	PhoneUnknownScore  = 60
)

// Company scoring values.
const (
	CompanyBaseScore   = 40
	CompanySuffixBonus = 10
)

// Unknown-input scoring values.
const (
	UnknownSuggestionScore = 20
	NoSuggestionScore      = 10
)

// Clamp bounds a score to the [0, 100] trust range.
func Clamp(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

// StatusFromScore maps a score to a status band.
//
//	90-100 -> verified
//	70-89  -> good
//	50-69  -> warning
//	20-49  -> poor
//	0-19   -> invalid
func StatusFromScore(score int) string {
	switch {
	case score >= VerifiedThreshold:
		return "verified"
	case score >= GoodThreshold:
		return "good"
	case score >= WarningThreshold:
		return "warning"
	case score >= PoorThreshold:
		return "poor"
	default:
		return "invalid"
	}
}

// Evaluation is the outcome of a scored verification.
type Evaluation struct {
	Score   int
	Status  string
	Summary string
}

// Builder accumulates a trust score and clamps automatically after every
// operation, so a verifier never has to clamp by hand. Every scoring step can
// also append Evidence via Pass/Warning/Fail/Info, which record how the score
// was calculated.
type Builder struct {
	score    int
	evidence []Evidence
}

// New returns an empty score Builder.
func New() *Builder { return &Builder{} }

// Add increases the score by n.
func (b *Builder) Add(n int) { b.score = Clamp(b.score + n) }

// Sub decreases the score by n.
func (b *Builder) Sub(n int) { b.score = Clamp(b.score - n) }

// Pass records a passing check that contributes points to the score.
func (b *Builder) Pass(label string, points int) {
	b.Add(points)
	b.evidence = append(b.evidence, Evidence{Label: label, Result: "pass", Points: points})
}

// Warning records a check that succeeded but signals caution. points may be
// negative when the check deducts from the score.
func (b *Builder) Warning(label string, points int) {
	b.Add(points)
	b.evidence = append(b.evidence, Evidence{Label: label, Result: "warning", Points: points})
}

// Fail records a failed check. points may be negative when the failure
// deducts from the score.
func (b *Builder) Fail(label string, points int) {
	b.Add(points)
	b.evidence = append(b.evidence, Evidence{Label: label, Result: "fail", Points: points})
}

// Info records an informational check that does not affect the score.
func (b *Builder) Info(label string) {
	b.evidence = append(b.evidence, Evidence{Label: label, Result: "info", Points: 0})
}

// Score returns the current clamped score.
func (b *Builder) Score() int { return b.score }

// Evidence returns a copy of the checks recorded so far, in the order they
// were appended.
func (b *Builder) Evidence() []Evidence {
	return append([]Evidence(nil), b.evidence...)
}

// StatusSummary returns a generic fallback summary for a status. Verifiers
// may override it with a more specific message when they need one.
func StatusSummary(status string) string {
	switch status {
	case "verified":
		return "Input verified successfully."
	case "good":
		return "Input is likely trustworthy."
	case "warning":
		return "Input could not be fully verified."
	case "poor":
		return "Input looks suspicious."
	case "invalid":
		return "Input is not valid."
	case "unreachable":
		return "Input could not be reached."
	case "local":
		return "Input refers to a local address."
	case "private":
		return "Input refers to a private address."
	case "suggestion":
		return "A valid input was suggested."
	case "unknown":
		return "Unable to classify the input."
	default:
		return "No summary available."
	}
}
