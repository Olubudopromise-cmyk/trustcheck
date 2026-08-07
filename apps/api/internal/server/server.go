// Package server wires the TrustCheck HTTP router: request logging, CORS,
// health check, OpenAPI/Swagger UI, and the /verify endpoint. Both the local
// development server (cmd/api) and the Netlify Function (function/api) build
// their router from this package so the deployed API behaves identically to
// the local one.
package server

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pamierin/trustcheck/apps/api/internal/analysis"
	"github.com/pamierin/trustcheck/apps/api/internal/classifier"
	"github.com/pamierin/trustcheck/apps/api/internal/docs"
	"github.com/pamierin/trustcheck/apps/api/internal/logging"
	"github.com/pamierin/trustcheck/apps/api/internal/model"
	"github.com/pamierin/trustcheck/apps/api/internal/ratelimit"
	"github.com/pamierin/trustcheck/apps/api/internal/scoring"
	"github.com/pamierin/trustcheck/apps/api/internal/spec"
	"github.com/pamierin/trustcheck/apps/api/internal/verifier"
	"github.com/swaggest/swgui/v5emb"
)

type verifyRequest struct {
	Input string `json:"input"`
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
	analyzer := analysis.New()

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
			Status:                vr.Status,
			TrustScore:            vr.TrustScore,
			Summary:               vr.Summary,
			Evidence:              vr.Evidence,
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
		})
	})

	return r
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
