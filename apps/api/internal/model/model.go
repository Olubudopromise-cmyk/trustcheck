// Package model defines the shared data types of the explainable-AI result.
//
// It is a leaf package imported by every analysis module (claims, warnings,
// interpretations, recommendations, reasoning) and by the analysis pipeline,
// so no module depends on another and new modules can be added without touching
// the existing ones.
package model

// Verdict is the coarse human-facing trust band derived from the numeric score.
type Verdict string

const (
	VerdictHigh   Verdict = "High"
	VerdictMedium Verdict = "Medium"
	VerdictLow    Verdict = "Low"
)

// VerdictFromScore maps a 0-100 trust score to a High/Medium/Low verdict.
//
//	70-100 -> High
//	40-69  -> Medium
//	0-39   -> Low
func VerdictFromScore(score int) Verdict {
	switch {
	case score >= 70:
		return VerdictHigh
	case score >= 40:
		return VerdictMedium
	default:
		return VerdictLow
	}
}

// StatusFromVerdict maps a verdict to the canonical status string used for
// badges and report wording. Status is derived from the verdict (which is
// derived from the trust score) so the two can never disagree.
//
//	High   -> "verified"
//	Medium -> "warning"
//	Low    -> "invalid"
func StatusFromVerdict(v Verdict) string {
	switch v {
	case VerdictHigh:
		return "verified"
	case VerdictMedium:
		return "warning"
	default:
		return "invalid"
	}
}

// Claim is a single factual claim extracted from the user input.
type Claim struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	Entities    []Entity `json:"entities"`
	Keywords    []string `json:"keywords"`
	Verdict     Verdict  `json:"verdict"`
	Confidence  int      `json:"confidence"`
	Evidence    []EvidenceItem `json:"evidence"`
	Conflicts   []Contradiction `json:"conflicts"`
	Summary     string   `json:"summary"`
	Timeline    []ReasoningStep `json:"timeline"`
	Recommendations []Recommendation `json:"recommendations"`
	Missing     []MissingInfo `json:"missingInformation"`
	Status      ClaimStatus `json:"status"`
}

// ClaimStatus is the verification status of a claim.
type ClaimStatus string

const (
	ClaimVerified   ClaimStatus = "verified"
	ClaimPartiallyVerified ClaimStatus = "partially_verified"
	ClaimUnverified ClaimStatus = "unverified"
	ClaimNoReliableEvidence ClaimStatus = "no_reliable_evidence"
)

// ClaimStatusCounts is a summary of claims by status.
type ClaimStatusCounts struct {
	Verified   int `json:"verified"`
	PartiallyVerified int `json:"partially_verified"`
	Unverified   int `json:"unverified"`
	NoReliableEvidence int `json:"no_reliable_evidence"`
}

// ClaimCard is a display-friendly version of a claim.
type ClaimCard struct {
	Claim     Claim
	Status    ClaimStatus
	Confidence  int
	EvidenceCount int
	ConflictCount int
	Evidence    []EvidenceItem
	Conflicts   []Contradiction
	Summary     string
	Timeline    []ReasoningStep
	Recommendations []Recommendation
	Missing     []MissingInfo
	SupportingEvidence  []SourceGroup
	ContradictingEvidence []Contradiction
	MissingInformation []MissingInfo
	ConfidenceBreakdown ConfidenceBreakdown
}



// --- Phase: Evidence Depth & Analysis Modes ---

// AnalysisMode defines the depth and focus of the verification analysis.
type AnalysisMode string

const (
	// ModeQuick is optimized for speed with standard evidence search.
	ModeQuick AnalysisMode = "quick"
	// ModeDeepResearch performs extensive evidence search, including
	// contradiction search and primary source identification.
	ModeDeepResearch AnalysisMode = "deep_research"
	// ModeGovernmentOfficial prioritizes official government and institutional
	// sources while still searching for independent confirmation.
	ModeGovernmentOfficial AnalysisMode = "government_official"
	// ModeSecurityReview performs defensive security analysis on source code.
	ModeSecurityReview AnalysisMode = "security_review"
)

// AnalysisSettings configures how the analysis pipeline operates.
// Different modes produce different evidence depths and explanations.
type AnalysisSettings struct {
	// Mode is the analysis mode (quick, deep_research, government_official).
	Mode AnalysisMode `json:"mode"`

	// SearchDepth controls how many sources to consider (1-10).
	// Quick=1, Deep=5, Government=3.
	SearchDepth int `json:"searchDepth"`

	// MaxSources is the maximum number of sources to retrieve.
	MaxSources int `json:"maxSources"`

	// RequireIndependentSources requires independent confirmation.
	RequireIndependentSources bool `json:"requireIndependentSources"`

	// SearchContradictions actively searches for contradicting evidence.
	SearchContradictions bool `json:"searchContradictions"`

	// PrioritizeGovernmentSources favors official government sources.
	PrioritizeGovernmentSources bool `json:"prioritizeGovernmentSources"`

	// PrioritizeAcademicSources favors peer-reviewed academic sources.
	PrioritizeAcademicSources bool `json:"prioritizeAcademicSources"`

	// PrioritizePrimarySources favors primary over secondary sources.
	PrioritizePrimarySources bool `json:"prioritizePrimarySources"`

	// MinimumEvidenceThreshold is the minimum evidence items required.
	MinimumEvidenceThreshold int `json:"minimumEvidenceThreshold"`
}

// DefaultSettings returns the default settings for a given mode.
func DefaultSettings(mode AnalysisMode) AnalysisSettings {
	switch mode {
	case ModeDeepResearch:
		return AnalysisSettings{
			Mode:                        ModeDeepResearch,
			SearchDepth:                 5,
			MaxSources:                  15,
			RequireIndependentSources:   true,
			SearchContradictions:        true,
			PrioritizeGovernmentSources: false,
			PrioritizeAcademicSources:   true,
			PrioritizePrimarySources:    true,
			MinimumEvidenceThreshold:    3,
		}
	case ModeGovernmentOfficial:
		return AnalysisSettings{
			Mode:                        ModeGovernmentOfficial,
			SearchDepth:                 3,
			MaxSources:                  10,
			RequireIndependentSources:   true,
			SearchContradictions:        true,
			PrioritizeGovernmentSources: true,
			PrioritizeAcademicSources:   true,
			PrioritizePrimarySources:    true,
			MinimumEvidenceThreshold:    2,
		}
	case ModeSecurityReview:
		return AnalysisSettings{
			Mode:                        ModeSecurityReview,
			SearchDepth:                 3,
			MaxSources:                  10,
			RequireIndependentSources:   true,
			SearchContradictions:        false,
			PrioritizeGovernmentSources: false,
			PrioritizeAcademicSources:   false,
			PrioritizePrimarySources:    false,
			MinimumEvidenceThreshold:    1,
		}
	default: // ModeQuick
		return AnalysisSettings{
			Mode:                        ModeQuick,
			SearchDepth:                 1,
			MaxSources:                  5,
			RequireIndependentSources:   false,
			SearchContradictions:        false,
			PrioritizeGovernmentSources: false,
			PrioritizeAcademicSources:   false,
			PrioritizePrimarySources:    false,
			MinimumEvidenceThreshold:    1,
		}
	}
}

// SourceType represents the type of a source.
type SourceType string

const (
	SourceOfficial      SourceType = "official"      // Government agencies, regulatory bodies
	SourceInstitutional  SourceType = "institutional"  // Universities, research institutions
	SourceJournalism     SourceType = "journalism"     // News organizations
	SourceCommunity      SourceType = "community"      // Forums, social media
	SourceAcademic       SourceType = "academic"       // Peer-reviewed journals
	SourceCommercial     SourceType = "commercial"     // Company websites, blogs
	SourceUnknown        SourceType = "unknown"        // Unclassified sources
)

// SourceRelation indicates the relationship of a source to the claim.
type SourceRelation string

const (
	RelationPrimary   SourceRelation = "primary"   // First-hand account, official announcement
	RelationSecondary SourceRelation = "secondary" // Reporting on primary source
	RelationTertiary  SourceRelation = "tertiary"  // Compilation or summary
)

// SourceIntelligence contains metadata about a source used in verification.
type SourceIntelligence struct {
	Title           string         `json:"title"`
	Domain          string         `json:"domain"`
	PublicationDate string         `json:"publicationDate,omitempty"`
	SourceType      SourceType     `json:"sourceType"`
	Relation        SourceRelation `json:"relation"`
	IsOfficial      bool           `json:"isOfficial"`
	Author          string         `json:"author,omitempty"`
	Citation        string         `json:"citation,omitempty"`
	Relevance       int            `json:"relevance"`       // 0-100
	SupportsClaim   bool           `json:"supportsClaim"`
	ContradictsClaim bool          `json:"contradictsClaim"`
	IsIndependent   bool           `json:"isIndependent"`   // Not affiliated with claim subject
	Confidence      int            `json:"confidence"`      // 0-100, how confident we are in this source's accuracy
}

// EvidenceLedger is a structured record of all evidence for and against a claim.
// The final verdict must be generated from this ledger.
type EvidenceLedger struct {
	Claim            string              `json:"claim"`
	Supporting       []LedgerEntry       `json:"supporting"`
	Contradicting    []LedgerEntry       `json:"contradicting"`
	Unknown          []string            `json:"unknown"`
	TotalSources     int                 `json:"totalSources"`
	IndependentCount int                 `json:"independentCount"`
	DuplicateCount   int                 `json:"duplicateCount"`
}

// LedgerEntry is a single piece of evidence in the ledger.
type LedgerEntry struct {
	Source    SourceIntelligence `json:"source"`
	Summary   string            `json:"summary"`
	Strength  int               `json:"strength"`   // 0-100, how strong this evidence is
	Notes     string            `json:"notes,omitempty"`
}

// ScoreExplanation is a user-facing breakdown of how the trust score was
type ScoreExplanation struct {
	EvidenceStrength        int    `json:"evidenceStrength"`        // 0-100
	EvidenceStrengthNote    string `json:"evidenceStrengthNote"`
	SourceQuality           int    `json:"sourceQuality"`           // 0-100
	SourceQualityNote       string `json:"sourceQualityNote"`
	IndependentConfirmation int    `json:"independentConfirmation"` // 0-100
	IndependentNote         string `json:"independentNote"`
	ContradictionRisk       int    `json:"contradictionRisk"`       // 0-100 (higher = more risk)
	ContradictionNote       string `json:"contradictionNote"`
	MissingEvidence         int    `json:"missingEvidence"`         // 0-100 (higher = more missing)
	MissingNote             string `json:"missingNote"`
}

// --- Security Intelligence Engine ---

// Severity represents the severity level of a security finding.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityInfo     Severity = "INFO"
)

// FindingStatus represents the verification status of a security finding.
type FindingStatus string

const (
	FindingConfirmed    FindingStatus = "CONFIRMED"
	FindingRequiresReview FindingStatus = "REQUIRES_REVIEW"
	FindingFalsePositive FindingStatus = "FALSE_POSITIVE"
	FindingNotFixed     FindingStatus = "NOT_FIXED"
	FindingPartiallyFixed FindingStatus = "PARTIALLY_FIXED"
	FindingFixed        FindingStatus = "FIXED"
	FindingUnverified   FindingStatus = "UNVERIFIED"
)

// SecurityCategory represents the category of a security vulnerability.
type SecurityCategory string

const (
	CategoryInjection        SecurityCategory = "injection"
	CategoryAuthWeakness     SecurityCategory = "authentication_weakness"
	CategoryAuthzFlaw        SecurityCategory = "authorization_flaw"
	CategoryIDOR             SecurityCategory = "insecure_direct_object_reference"
	CategorySSRF             SecurityCategory = "server_side_request_forgery"
	CategoryXSS              SecurityCategory = "cross_site_scripting"
	CategoryCSRF             SecurityCategory = "cross_site_request_forgery"
	CategoryDeserialization  SecurityCategory = "insecure_deserialization"
	CategoryPathTraversal    SecurityCategory = "path_traversal"
	CategoryCommandExecution SecurityCategory = "command_execution"
	CategorySecretsExposure  SecurityCategory = "secrets_exposure"
	CategoryCryptoWeakness   SecurityCategory = "insecure_cryptography"
	CategoryWeakPasswords    SecurityCategory = "weak_password_handling"
	CategoryUnsafeFileHandling SecurityCategory = "unsafe_file_handling"
	CategoryInsecureHTTP     SecurityCategory = "insecure_http_configuration"
	CategoryDependencyRisk   SecurityCategory = "dependency_vulnerability"
	CategoryExcessivePerms    SecurityCategory = "excessive_permissions"
	CategoryUnsafeErrorHandling SecurityCategory = "unsafe_error_handling"
	CategoryMissingControls  SecurityCategory = "missing_security_controls"
)

// SecurityFinding represents a single security vulnerability or issue found
// in the analyzed code. Every finding must contain concrete evidence.
type SecurityFinding struct {
	ID               string          `json:"id"`
	Title            string          `json:"title"`
	Severity         Severity        `json:"severity"`
	Confidence       int             `json:"confidence"` // 0-100
	Category         SecurityCategory `json:"category"`
	File             string          `json:"file"`
	Line             int             `json:"line,omitempty"`
	EndLine          int             `json:"endLine,omitempty"`
	Description      string          `json:"description"`
	SecurityImpact   string          `json:"securityImpact"`
	Evidence         string          `json:"evidence"`
	Remediation      string          `json:"remediation"`
	Patch            string          `json:"patch,omitempty"`
	References       []string        `json:"references,omitempty"`
	Status           FindingStatus   `json:"status"`
	EvidenceType     string          `json:"evidenceType"` // "local_code" or "external_research"
}

// SecurityReport is the complete security analysis report.
type SecurityReport struct {
	// ExecutiveSummary is a high-level overview of the security posture.
	ExecutiveSummary string `json:"executiveSummary"`

	// SecurityScore is derived from actual findings (0-100, higher = more secure).
	SecurityScore int `json:"securityScore"`

	// Finding counts by severity.
	CriticalCount int `json:"criticalCount"`
	HighCount     int `json:"highCount"`
	MediumCount   int `json:"mediumCount"`
	LowCount      int `json:"lowCount"`
	InfoCount     int `json:"infoCount"`

	// Findings is the list of all security findings.
	Findings []SecurityFinding `json:"findings"`

	// DependencyRisks lists vulnerabilities in dependencies.
	DependencyRisks []DependencyRisk `json:"dependencyRisks"`

	// RecommendedFixes provides prioritized remediation steps.
	RecommendedFixes []RecommendedFix `json:"recommendedFixes"`

	// VerificationResults shows results of applying fixes.
	VerificationResults []VerificationResult `json:"verificationResults,omitempty"`

	// RemainingRisks lists risks that could not be fully mitigated.
	RemainingRisks []string `json:"remainingRisks"`

	// EvidenceType indicates whether findings are from local code or external research.
	EvidenceType string `json:"evidenceType"`
}

// DependencyRisk represents a vulnerability in a dependency.
type DependencyRisk struct {
	Package             string   `json:"package"`
	Version             string   `json:"version"`
	Vulnerability       string   `json:"vulnerability"`
	Severity            Severity `json:"severity"`
	AffectedVersions    string   `json:"affectedVersions"`
	RecommendedUpgrade  string   `json:"recommendedUpgrade"`
	IsAffected          bool     `json:"isAffected"`
	AdvisorySource      string   `json:"advisorySource"`
	RetrievalDate       string   `json:"retrievalDate"`
	Description         string   `json:"description,omitempty"`
}

// RecommendedFix represents a recommended security fix.
type RecommendedFix struct {
	FindingID   string `json:"findingId"`
	Priority    int    `json:"priority"` // 1 = highest
	Explanation string `json:"explanation"`
	Patch       string `json:"patch,omitempty"`
}

// VerificationResult shows the result of applying a fix.
type VerificationResult struct {
	FindingID string `json:"findingId"`
	Status    string `json:"status"` // FIXED, PARTIALLY_FIXED, NOT_FIXED, UNVERIFIED
	Details   string `json:"details"`
}

// SecuritySettings configures the security analysis.
type SecuritySettings struct {
	// ScanDependencies enables dependency vulnerability scanning.
	ScanDependencies bool `json:"scanDependencies"`
	// ScanSecrets enables secrets detection.
	ScanSecrets bool `json:"scanSecrets"`
	// ScanInjection enables injection vulnerability detection.
	ScanInjection bool `json:"scanInjection"`
	// ScanXSS enables XSS detection.
	ScanXSS bool `json:"scanXSS"`
	// MaxFindings is the maximum number of findings to return.
	MaxFindings int `json:"maxFindings"`
}

// DefaultSecuritySettings returns sensible defaults for security analysis.
func DefaultSecuritySettings() SecuritySettings {
	return SecuritySettings{
		ScanDependencies: true,
		ScanSecrets:      true,
		ScanInjection:    true,
		ScanXSS:          true,
		MaxFindings:      100,
	}
}

// Entity is a named thing (person, organization, location, ...) extracted from
// the input. Kind is one of the EntityKind constants.
type Entity struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// Entity kinds emitted by the claims extractor.
const (
	EntityOrganization = "organization"
	EntityLocation     = "location"
	EntityPerson       = "person"
	EntityDate         = "date"
)

// Interpretation is one plausible reading of an input. An analysis always
// returns 2-3 interpretations so a single meaning is never assumed. Confidence
// is 0-100 and reasoning explains why that reading is plausible.
// SupportingEvidenceCount is the number of scored checks consistent with this
// reading, so the user can see how much evidence backs each interpretation.
type Interpretation struct {
	Title                   string `json:"title"`
	Explanation             string `json:"explanation"`
	Confidence              int    `json:"confidence"`
	Reasoning               string `json:"reasoning"`
	SupportingEvidenceCount int    `json:"supportingEvidenceCount"`
}

// WarningSignal is a structured misinformation indicator. Severity is one of
// "high", "medium", or "low".
type WarningSignal struct {
	Label       string `json:"label"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// Recommendation is an actionable next step for the user, with a short reason.
type Recommendation struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ReasoningStep is one stage of the analysis timeline. The timeline exposes
// how the assessment was reached — claim detected, evidence gathered, conflicts
// identified, risk signals, reasoning, final assessment — so the user reviews
// an investigation rather than an unexplained score. Details is a concise,
// user-facing list; it never contains chain-of-thought or internal reasoning.
type ReasoningStep struct {
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Details []string `json:"details"`
}

// EvidenceItem is one scored check, bucketed into a supporting or
// contradicting section. Note carries a plain-English explanation of the check.
type EvidenceItem struct {
	Label  string `json:"label"`
	Result string `json:"result"`
	Points int    `json:"points"`
	Note   string `json:"note,omitempty"`
}

// SourceEvidence is one piece of evidence attributed to a named source. All
// fields are grounded in observable findings; PublicationDate is empty when the
// date is unknown and is never fabricated.
type SourceEvidence struct {
	Title           string `json:"title"`
	Source          string `json:"source"`
	Credibility     string `json:"credibility"`
	PublicationDate string `json:"publicationDate,omitempty"`
	Summary         string `json:"summary"`
}

// SourceGroup groups SourceEvidence items by the kind of source they came from
// (official, independent journalism, community, academic, ...).
type SourceGroup struct {
	Category string           `json:"category"`
	Items    []SourceEvidence `json:"items"`
}

// Contradiction describes a disagreement between sources or between the claim
// and a source. ConfidenceInContradiction is 0-100 and reflects how strongly
// the evidence supports that the sources genuinely disagree.
type Contradiction struct {
	SourceA                   string `json:"sourceA"`
	ClaimA                    string `json:"claimA"`
	SourceB                   string `json:"sourceB"`
	ClaimB                    string `json:"claimB"`
	WhyTheyDisagree           string `json:"whyTheyDisagree"`
	ConfidenceInContradiction int    `json:"confidenceInContradiction"`
}

// MissingInfo is one fact that is absent from the submission. WhyItMatters
// explains the consequence in plain English.
type MissingInfo struct {
	Item         string `json:"item"`
	WhyItMatters string `json:"whyItMatters"`
}

// ConfidenceMetric is one user-friendly, 0-100 component of the confidence
// breakdown. Notes explain what each metric reflects so no hidden scoring
// algorithm is exposed.
type ConfidenceMetric struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	Note  string `json:"note"`
}

// ConfidenceBreakdown is the user-facing decomposition of the overall
// confidence. Overall is 0-100; Metrics are the components that explain it.
type ConfidenceBreakdown struct {
	Overall int                `json:"overall"`
	Metrics []ConfidenceMetric `json:"metrics"`
}

// SuggestedReading recommends material the user should consult. URLs are never
// fabricated; when a specific resource is unknown the section carries an
// honest statement instead.
type SuggestedReading struct {
	Title      string `json:"title"`
	Publisher  string `json:"publisher"`
	WhyItHelps string `json:"whyItHelps"`
}

// ChangeEvent is one dated step in how the story evolved over time. Dates are
// only ever the ones that were actually observed; nothing is invented.
type ChangeEvent struct {
	Date  string `json:"date"`
	Event string `json:"event"`
}

// Result is the complete explainable analysis for one verified input.
//
// The Input/Type/Status/TrustScore/Summary fields mirror the legacy /verify
// response so older clients keep working; every field after them is the
// explainable-AI extension.
type Result struct {
	Input      string `json:"input"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	TrustScore int    `json:"trustScore"`
	Summary    string `json:"summary"`

	// Verdict is the coarse High/Medium/Low band of TrustScore.
	Verdict Verdict `json:"verdict"`

	// KeyClaim is the single sentence under review. For free-form text it is
	// the normalized input sentence; for structured identifiers it is the
	// trustworthiness claim being tested.
	KeyClaim string `json:"keyClaim"`

	// Entities and Keywords are extracted from the input by the claims module.
	Entities []Entity `json:"entities"`
	Keywords []string `json:"keywords"`

	// EvidenceFor / EvidenceAgainst split the scored checks by direction so
	// the user sees what supports and what contradicts the claim.
	EvidenceFor     []EvidenceItem `json:"evidenceFor"`
	EvidenceAgainst []EvidenceItem `json:"evidenceAgainst"`

	// MissingEvidence lists checks that could not be run or for which no
	// evidence was available. These are explicit statements, never fabricated.
	MissingEvidence []string `json:"missingEvidence"`

	// UnknownInformation lists facts that are genuinely unknown (author,
	// publication date, provenance, ...). Also explicit and never invented.
	UnknownInformation []string `json:"unknownInformation"`

	// Interpretations are 2-3 plausible readings of the input.
	Interpretations []Interpretation `json:"interpretations"`

	// WarningSignals are the structured misinformation indicators detected.
	WarningSignals []WarningSignal `json:"warningSignals"`

	// Confidence is 0-100 and reflects how confident the analysis is in its
	// own assessment (not the trust score itself).
	Confidence int `json:"confidence"`

	// Reasoning is the ordered bullet explanation of why the score is what it
	// is. Positive bullets are prefixed with "+", negative with "-".
	Reasoning []string `json:"reasoning"`

	// Timeline is the step-by-step reasoning timeline shown at the top of the
	// analysis. Each step has a one-line summary and expandable details.
	Timeline []ReasoningStep `json:"timeline"`

	// Recommendations are the next steps the user should take.
	Recommendations []Recommendation `json:"recommendations"`

	// --- Phase 12: multi-perspective fact analysis ---

	// SupportingEvidence groups supporting evidence by source category.
	SupportingEvidence []SourceGroup `json:"supportingEvidence"`

	// ContradictingEvidence lists disagreements found between sources.
	ContradictingEvidence []Contradiction `json:"contradictingEvidence"`

	// MissingInformation is the "What is missing?" section.
	MissingInformation []MissingInfo `json:"missingInformation"`

	// ConfidenceBreakdown decomposes the overall confidence into user-facing
	// metrics.
	ConfidenceBreakdown ConfidenceBreakdown `json:"confidenceBreakdown"`

	// AISummary is a short, user-facing paragraph (≤120 words) summarizing the
	// assessment. It is AI-generated and labeled as such; it never invents
	// facts beyond what the structured sections report.
	AISummary string `json:"aiSummary"`

	// SuggestedReading recommends material to consult. URLs are never
	// fabricated; when none can be identified the section says so.
	SuggestedReading []SuggestedReading `json:"suggestedReading"`

	// SuggestedReadingNote is an honest statement shown when no specific
	// reading could be identified (never a fabricated list).
	SuggestedReadingNote string `json:"suggestedReadingNote,omitempty"`

	// WhatChanged is the dated evolution of the story. Dates are only ever
	// observed ones; nothing is invented.
	WhatChanged []ChangeEvent `json:"whatChanged"`

	// WhatChangedNote is an honest statement shown when no timeline could be
	// reconstructed (never a fabricated timeline).
	WhatChangedNote string `json:"whatChangedNote,omitempty"`

	// --- Phase 13: intelligent claim extraction ---

	// Claims lists the individual factual claims extracted from the input.
	// Each claim is independently verified with its own evidence, verdict,
	// and confidence. For inputs that produce only a single claim, Claims
	// contains exactly one element and KeyClaim/EvidenceFor/EvidenceAgainst
	// remain populated for backward compatibility.
	Claims []Claim `json:"claims"`

	// ClaimCount is the total number of extracted claims.
	ClaimCount int `json:"claimCount"`

	// VerifiedClaims is the number of claims with status "verified".
	VerifiedClaims int `json:"verifiedClaims"`

	// PartialClaims is the number of claims with status "partially_verified".
	PartialClaims int `json:"partialClaims"`

	// UnverifiedClaims is the number of claims with status "unverified" or
	// "no_reliable_evidence".
	UnverifiedClaims int `json:"unverifiedClaims"`

	// --- Phase: Evidence Depth & Analysis Modes ---

	// AnalysisMode is the mode used for this analysis.
	AnalysisMode AnalysisMode `json:"analysisMode"`

	// EvidenceLedger is the structured record of all evidence.
	EvidenceLedger EvidenceLedger `json:"evidenceLedger"`

	// ScoreExplanation breaks down how the trust score was derived.
	ScoreExplanation ScoreExplanation `json:"scoreExplanation"`

	// SourceIntelligence lists metadata about all sources used.
	SourceIntelligence []SourceIntelligence `json:"sourceIntelligence"`

	// --- Security Intelligence Engine ---

	// SecurityReport contains the results of a security analysis when
	// the mode is SECURITY_REVIEW.
	SecurityReport *SecurityReport `json:"securityReport,omitempty"`
}
