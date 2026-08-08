// Package server wires the TrustCheck HTTP router: request logging, CORS,
// health check, OpenAPI/Swagger UI, and the /verify endpoint. Both the local
// development server (cmd/api) and the Netlify Function (function/api) build
// their router from this package so the deployed API behaves identically to
// the local one.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pamierin/trustcheck/apps/api/internal/analysis"
	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/docs"
	"github.com/pamierin/trustcheck/apps/api/internal/logging"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
	"github.com/pamierin/trustcheck/apps/api/internal/ratelimit"
	"github.com/pamierin/trustcheck/apps/api/internal/scoring"
	"github.com/pamierin/trustcheck/apps/api/internal/security"
	"github.com/pamierin/trustcheck/apps/api/internal/spec"
	"github.com/pamierin/trustcheck/apps/api/internal/verifier"
	"github.com/swaggest/swgui/v5emb"
)

type verifyRequest struct {
	Input string           `json:"input"`
	Mode  model.AnalysisMode `json:"mode,omitempty"`
}

// securityRequest is the /security request body.
type securityRequest struct {
	Code     string            `json:"code"`
	Filename string            `json:"filename"`
	Language string            `json:"language,omitempty"`
	Files    map[string]string `json:"files,omitempty"` // Additional files for dependency scanning
}

// securityResponse is the /security response body.
type securityResponse struct {
	Report         *model.SecurityReport  `json:"report"`
	DependencyRisks []model.DependencyRisk `json:"dependencyRisks,omitempty"`
}

// verifyResponse is the /verify response body. The legacy fields
// (input, type, status, trustScore, summary) are preserved verbatim; the
// analysis model adds the explainable-AI sections without breaking clients
// that only read the legacy fields.
type verifyResponse struct {
	Input      string             `json:"input"`
	Type       string             `json:"type"`
	Status     string             `json:"status"`
	TrustScore int                `json:"trustScore"`
	Summary    string             `json:"summary"`
	Evidence   []scoring.Evidence `json:"evidence"`

	Verdict            model.Verdict          `json:"verdict"`
	KeyClaim           string                 `json:"keyClaim"`
	Entities           []model.Entity         `json:"entities"`
	Keywords           []string               `json:"keywords"`
	EvidenceFor        []model.EvidenceItem   `json:"evidenceFor"`
	EvidenceAgainst    []model.EvidenceItem   `json:"evidenceAgainst"`
	MissingEvidence    []string               `json:"missingEvidence"`
	UnknownInformation []string               `json:"unknownInformation"`
	Interpretations    []model.Interpretation `json:"interpretations"`
	WarningSignals     []model.WarningSignal  `json:"warningSignals"`
	Confidence         int                    `json:"confidence"`
	Reasoning          []string               `json:"reasoning"`
	Timeline           []model.ReasoningStep  `json:"timeline"`
	Recommendations    []model.Recommendation `json:"recommendations"`

	// Phase 12: multi-perspective fact analysis.
	SupportingEvidence    []model.SourceGroup       `json:"supportingEvidence"`
	ContradictingEvidence []model.Contradiction     `json:"contradictingEvidence"`
	MissingInformation    []model.MissingInfo       `json:"missingInformation"`
	ConfidenceBreakdown   model.ConfidenceBreakdown `json:"confidenceBreakdown"`
	AISummary             string                    `json:"aiSummary"`
	SuggestedReading      []model.SuggestedReading  `json:"suggestedReading"`
	SuggestedReadingNote  string                    `json:"suggestedReadingNote,omitempty"`
	WhatChanged           []model.ChangeEvent       `json:"whatChanged"`
	WhatChangedNote       string                    `json:"whatChangedNote,omitempty"`

	// Phase 13: intelligent claim extraction.
	Claims          []model.Claim `json:"claims"`
	ClaimCount      int           `json:"claimCount"`
	VerifiedClaims  int           `json:"verifiedClaims"`
	PartialClaims   int           `json:"partialClaims"`
	UnverifiedClaims int          `json:"unverifiedClaims"`

	// Evidence Depth & Analysis Modes.
	AnalysisMode      model.AnalysisMode          `json:"analysisMode"`
	EvidenceLedger    model.EvidenceLedger        `json:"evidenceLedger"`
	ScoreExplanation  model.ScoreExplanation      `json:"scoreExplanation"`
	SourceIntelligence []model.SourceIntelligence  `json:"sourceIntelligence"`
}

// NewRouter builds the fully configured TrustCheck API router.
//
// prefix is prepended to every route so the same router works in every
// environment: pass "" for the local server and "/api" for the Netlify
// Function (which is mounted behind the /api/* rewrite in netlify.toml). The
// prefix is also used for the OpenAPI spec and Swagger UI paths so those
// resolve at a stable absolute URL from the browser.
func NewRouter(prefix string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	logger := logging.New()

	// analyzer runs the explainable-AI pipeline. Future modules (live web
	// search, fact-check integrations, ...) are registered here.
	baseAnalyzer := analysis.New()

	limiter := ratelimit.New(ratelimit.Options{
		Rate:  1, // 60 requests per minute
		Burst: 20,
	})
	limiter.StartCleanup(context.Background(), time.Minute, 10*time.Minute)

	r := gin.New()
	r.Use(logging.RequestID())
	r.Use(logging.RequestLogger(logger))
	r.Use(logging.Recovery())
	r.Use(corsMiddleware())

	r.GET(prefix+"/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "trustcheck-api",
		})
	})

	r.GET(prefix+docs.OpenAPIPath, func(c *gin.Context) {
		c.Data(http.StatusOK, docs.YAMLContentType(), spec.YAML)
	})

	swaggerUI := v5emb.New(docs.SwaggerUITitle, prefix+docs.OpenAPIPath, prefix+docs.DocsPath)

	r.GET(prefix+docs.DocsPath, gin.WrapH(swaggerUI))
	r.GET(prefix+docs.DocsPath+"/*any", gin.WrapH(swaggerUI))

	r.POST(prefix+"/verify", ratelimit.RateLimit(limiter), func(c *gin.Context) {
		var req verifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: input is required"})
			return
		}
		if req.Input == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "input is required"})
			return
		}

		// Determine analysis mode from request, defaulting to quick.
		mode := req.Mode
		if mode == "" {
			mode = model.ModeQuick
		}
		settings := model.DefaultSettings(mode)
		analyzer := baseAnalyzer.WithSettings(settings)

		detected := classifier.Detect(req.Input)
		vr := verifier.Verify(detected, req.Input)
		result := analyzer.Analyze(context.Background(), req.Input, detected, vr)

		logging.AddRequestAttrs(c,
			"inputType", string(detected),
			"trustScore", vr.TrustScore,
			"verificationStatus", vr.Status,
		)

		c.JSON(http.StatusOK, verifyResponse{
			Input:                 req.Input,
			Type:                  string(detected),
			Status:                result.Status,
			TrustScore:            vr.TrustScore,
			Summary:               vr.Summary,
			Evidence:              scoring.FilterEnginePlaceholders(vr.Evidence),
			Verdict:               result.Verdict,
			KeyClaim:              result.KeyClaim,
			Entities:              result.Entities,
			Keywords:              result.Keywords,
			EvidenceFor:           result.EvidenceFor,
			EvidenceAgainst:       result.EvidenceAgainst,
			MissingEvidence:       result.MissingEvidence,
			UnknownInformation:    result.UnknownInformation,
			Interpretations:       result.Interpretations,
			WarningSignals:        result.WarningSignals,
			Confidence:            result.Confidence,
			Reasoning:             result.Reasoning,
			Timeline:              result.Timeline,
			Recommendations:       result.Recommendations,
			SupportingEvidence:    result.SupportingEvidence,
			ContradictingEvidence: result.ContradictingEvidence,
			MissingInformation:    result.MissingInformation,
			ConfidenceBreakdown:   result.ConfidenceBreakdown,
			AISummary:             result.AISummary,
			SuggestedReading:      result.SuggestedReading,
			SuggestedReadingNote:  result.SuggestedReadingNote,
			WhatChanged:           result.WhatChanged,
			WhatChangedNote:       result.WhatChangedNote,
			Claims:                result.Claims,
			ClaimCount:            result.ClaimCount,
			VerifiedClaims:        result.VerifiedClaims,
			PartialClaims:         result.PartialClaims,
			UnverifiedClaims:      result.UnverifiedClaims,
			AnalysisMode:          result.AnalysisMode,
			EvidenceLedger:        result.EvidenceLedger,
			ScoreExplanation:      result.ScoreExplanation,
			SourceIntelligence:    result.SourceIntelligence,
		})
	})

	// Security analysis endpoint
	r.POST(prefix+"/security", ratelimit.RateLimit(limiter), func(c *gin.Context) {
		var req securityRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: code is required"})
			return
		}
		if req.Code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
			return
		}
		if req.Filename == "" {
			req.Filename = "unknown"
		}
		if req.Language == "" {
			req.Language = detectLanguage(req.Filename)
		}

		securityAnalyzer := security.New()
		report := securityAnalyzer.Analyze(req.Filename, req.Code, req.Language)

		// Perform dependency scanning if files are provided
		var depRisks []model.DependencyRisk
		if len(req.Files) > 0 {
			depRisks = securityAnalyzer.AnalyzeDependencies(req.Files)
			report.DependencyRisks = depRisks

			// Add dependency findings to report
			for i, risk := range depRisks {
				if risk.IsAffected {
					findingID := len(report.Findings) + i + 1
					report.Findings = append(report.Findings, model.SecurityFinding{
						ID:             fmt.Sprintf("SEC-%04d", findingID),
						Title:          fmt.Sprintf("Vulnerable Dependency: %s", risk.Package),
						Severity:       risk.Severity,
						Confidence:     90,
						Category:       model.CategoryDependencyRisk,
						File:           risk.Package,
						Description:    fmt.Sprintf("Package %s@%s has known vulnerability %s.", risk.Package, risk.Version, risk.Vulnerability),
						SecurityImpact: risk.Description,
						Evidence: fmt.Sprintf("Package: %s\nVersion: %s\nVulnerability: %s\nAffected: %s",
							risk.Package, risk.Version, risk.Vulnerability, risk.AffectedVersions),
						Remediation: fmt.Sprintf("Upgrade to version %s or later.", risk.RecommendedUpgrade),
						References:  []string{fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", risk.Vulnerability)},
						Status:      model.FindingConfirmed,
						EvidenceType: "external_research",
					})				}
			}

			// Update severity counts
			for _, risk := range depRisks {
				if risk.IsAffected {
					switch risk.Severity {
					case model.SeverityCritical:
						report.CriticalCount++
					case model.SeverityHigh:
						report.HighCount++
					case model.SeverityMedium:
						report.MediumCount++
					case model.SeverityLow:
						report.LowCount++
					case model.SeverityInfo:
						report.InfoCount++
					}
				}
			}
		}

		logging.AddRequestAttrs(c,
			"filename", req.Filename,
			"language", req.Language,
			"securityScore", report.SecurityScore,
			"findings", len(report.Findings),
			"dependencyRisks", len(depRisks),
		)

		c.JSON(http.StatusOK, securityResponse{
			Report:          report,
			DependencyRisks: depRisks,
		})
	})

	return r
}

// detectLanguage detects the programming language from the filename extension.
func detectLanguage(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".go"):
		return "go"
	case strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".jsx"):
		return "javascript"
	case strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx"):
		return "typescript"
	case strings.HasSuffix(lower, ".py"):
		return "python"
	case strings.HasSuffix(lower, ".java"):
		return "java"
	case strings.HasSuffix(lower, ".rb"):
		return "ruby"
	case strings.HasSuffix(lower, ".php"):
		return "php"
	case strings.HasSuffix(lower, ".rs"):
		return "rust"
	case strings.HasSuffix(lower, ".cs"):
		return "csharp"
	case strings.HasSuffix(lower, ".cpp") || strings.HasSuffix(lower, ".c"):
		return "cpp"
	default:
		return "unknown"
	}
}

// corsMiddleware enables cross-origin requests from the allowed frontend
// origin only. It answers CORS preflight (OPTIONS) requests and injects the
// Access-Control-Allow-Origin header on actual responses. No third-party
// packages are required. On Netlify the frontend and API share an origin, so
// this is only exercised during local development.
func corsMiddleware() gin.HandlerFunc {
	allowedOrigins := map[string]bool{
		os.Getenv("ALLOWED_ORIGIN"): true,
	}
	if allowedOrigins[""] {
		allowedOrigins["http://localhost:3000"] = true
	}
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
