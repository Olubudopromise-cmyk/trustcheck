# TrustCheck — Codebase Index

> Auto-generated August 8, 2026

---

## Project Overview

TrustCheck is a fact-verification and security-analysis platform. Users submit URLs, text claims, or source code and receive an evidence-driven trust assessment with a numeric score, verdict, supporting/contradicting evidence, and actionable recommendations.

**Stack**: Go 1.23 (backend) · Next.js App Router (frontend) · pnpm monorepo · Netlify deployment

---

## Go Backend (`apps/api/`)

### Entry Points

| File              | Purpose                                           |
| ----------------- | ------------------------------------------------- |
| `cmd/api/main.go` | Local development server (port 8080, Gin)         |
| `function/api.go` | Netlify Function entry point (AWS Lambda adapter) |

### Internal Packages

| Package           | Key Files                                                                                           | Purpose                                                                                                      |
| ----------------- | --------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `server`          | `server.go`, `server_test.go`                                                                       | Gin HTTP router, route registration, middleware chain                                                        |
| `model`           | `model.go`, `model_test.go`                                                                         | Shared types: `Verdict`, `EvidenceItem`, `ScoreBreakdown`, `VerifyResponse`, `Claim`, `SecurityReport`, etc. |
| `analysis`        | `pipeline.go`, `analysis.go`, `timeline.go`                                                         | Main analysis orchestrator, reasoning timeline builder                                                       |
| `research`        | `provider.go`, `engine.go`, `module.go`, `duckduckgo.go`, `wikipedia.go`, `multi_provider.go`       | SearchProvider interface, DuckDuckGo HTML scraping, Wikipedia API, multi-provider dedup + classification     |
| `scoring`         | `scoring.go`, `classify.go`, `evidence.go`                                                          | Trust score calculation from evidence, evidence classification                                               |
| `claims`          | `claims.go`, `claims_test.go`                                                                       | Claim extraction from text articles                                                                          |
| `security`        | `analyzer.go`, `dependency_analyzer.go`                                                             | Source code security scanning, Go/Node.js dependency vulnerability detection                                 |
| `verifier`        | `dispatcher.go`, `url.go`, `email.go`, `ip.go`, `phone.go`, `domain.go`, `company.go`, `unknown.go` | Type-specific verification (DNS, HTTP, TLS, email, IP reputation, etc.)                                      |
| `classifier`      | `classifier.go`, `classifier_test.go`                                                               | Input type classification (URL, email, IP, phone, domain, text)                                              |
| `interpretations` | `interpretations.go`                                                                                | Multi-perspective analysis                                                                                   |
| `perspectives`    | `perspectives.go`                                                                                   | Alternative viewpoints                                                                                       |
| `reasoning`       | `reasoning.go`                                                                                      | Reasoning chain generation                                                                                   |
| `recommendations` | `recommendations.go`                                                                                | Actionable recommendation generation                                                                         |
| `warnings`        | `warnings.go`                                                                                       | Risk signal detection                                                                                        |
| `logging`         | `logger.go`, `middleware.go`                                                                        | Request logging middleware                                                                                   |
| `ratelimit`       | `limiter.go`, `middleware.go`                                                                       | Rate limiting middleware                                                                                     |
| `spec`            | `spec.go`, `openapi.yaml`                                                                           | OpenAPI specification                                                                                        |
| `docs`            | `docs.go`                                                                                           | Swagger UI serving                                                                                           |

### API Endpoints (server.go)

```
GET  /health           → Health check
GET  /openapi.json     → OpenAPI spec
GET  /docs             → Swagger UI
POST /verify           → Main verification (body: {input, mode})
POST /security         → Security analysis (body: {code, language, ...})
```

### Analysis Modes

| Mode                  | Search Depth | Source Count               | Contradiction Search |
| --------------------- | ------------ | -------------------------- | -------------------- |
| `quick`               | Shallow      | Fewer                      | Basic                |
| `deep_research`       | Deep         | More                       | Active               |
| `government_official` | Deep         | Gov/institutional priority | Active               |

### Request Flow

```
POST /verify
  → classifier.Classify(input)          // Detect input type
  → verifier.Dispatch(input)            // Type-specific verification
  → research.Engine.Research(claim)     // Evidence retrieval (DuckDuckGo + Wikipedia)
  → scoring.Score(evidence)             // Trust score calculation
  → analysis.Pipeline.Analyze(input)    // Full analysis with timeline
  → JSON response
```

---

## Frontend (`apps/web/`)

### App Structure (Next.js App Router)

```
src/app/
  layout.tsx       → Root layout (HTML, fonts, metadata)
  page.tsx         → Main page (single-page app)
  globals.css      → Tailwind CSS + custom styles
  error.tsx        → Error boundary
  not-found.tsx    → 404 page
  manifest.ts      → PWA manifest
```

### Components (`src/components/`)

| Component                   | Purpose                                         |
| --------------------------- | ----------------------------------------------- |
| `SearchForm.tsx`            | Main input form (URL, text, mode selector)      |
| `ResultCard.tsx`            | Verification result display container           |
| `TrustScore.tsx`            | Circular score visualization                    |
| `StatusBadge.tsx`           | Verdict status badge (Verified/Warning/Invalid) |
| `EvidenceList.tsx`          | Supporting evidence display                     |
| `SupportingEvidence.tsx`    | Supporting source list                          |
| `ContradictingEvidence.tsx` | Contradicting source list                       |
| `ReasoningTimeline.tsx`     | Step-by-step reasoning display                  |
| `ReasoningList.tsx`         | Reasoning chain list                            |
| `MainClaimSection.tsx`      | Primary claim display                           |
| `InterpretationsList.tsx`   | Multi-perspective analysis                      |
| `RecommendationsList.tsx`   | Actionable recommendations                      |
| `MissingInformation.tsx`    | Missing evidence indicators                     |
| `WarningSignals.tsx`        | Risk signal display                             |
| `ConfidenceBreakdown.tsx`   | Score component breakdown                       |
| `AISummary.tsx`             | AI-generated summary                            |
| `WhatChanged.tsx`           | Changes from previous verification              |
| `SuggestedReading.tsx`      | Related reading links                           |
| `BatchInput.tsx`            | Multi-input verification                        |
| `BatchResults.tsx`          | Batch results display                           |
| `AnalyticsDashboard.tsx`    | Usage analytics                                 |
| `ExportMenu.tsx`            | PDF/JSON export                                 |
| `HistoryList.tsx`           | Verification history                            |
| `ThemeToggle.tsx`           | Dark/light mode toggle                          |
| `LoadingSpinner.tsx`        | Loading state                                   |
| `ErrorCard.tsx`             | Error display                                   |
| `EmptyState.tsx`            | Empty state placeholder                         |
| `CollapsibleSection.tsx`    | Expandable/collapsible section                  |
| `TypeIcon.tsx`              | Input type icon                                 |
| `ExampleChips.tsx`          | Example input suggestions                       |

### Hooks

| Hook                        | Purpose                                     |
| --------------------------- | ------------------------------------------- |
| `useVerificationHistory.ts` | Manage verification history in localStorage |
| `useBatchVerification.ts`   | Handle multi-input verification             |

### Utils

| File              | Purpose                                 |
| ----------------- | --------------------------------------- |
| `api.ts`          | API client (`verify()`, `API_BASE_URL`) |
| `report.ts`       | PDF/JSON report generation              |
| `analytics.ts`    | Client-side analytics                   |
| `history.ts`      | Local history management                |
| `relativeTime.ts` | Time formatting ("2 hours ago")         |

### TypeScript Types (`types.ts`)

Key types: `VerifyResponse`, `AnalysisMode`, `ApiError`, `EvidenceItem`, `Verdict`, `SecurityReport`, etc.

---

## Configuration

### Root

| File                       | Purpose                            |
| -------------------------- | ---------------------------------- |
| `package.json`             | Workspace root scripts             |
| `pnpm-workspace.yaml`      | Defines `apps/*` as workspaces     |
| `turbo.json`               | Turborepo pipeline config          |
| `.github/workflows/ci.yml` | CI pipeline (Go test, build, lint) |
| `netlify.toml`             | Netlify deployment config          |

### Go Module (`apps/api/go.mod`)

```
module github.com/pamierin/trustcheck/apps/api
go 1.23.0
```

Key dependencies:

- `github.com/gin-gonic/gin` — HTTP router
- `github.com/aws/aws-lambda-go` — Lambda runtime
- `github.com/awslabs/aws-lambda-go-api-proxy` — Gin ↔ Lambda adapter
- `github.com/swaggest/swgui` — Swagger UI

### Frontend (`apps/web/package.json`)

Key dependencies:

- `next` — React framework (App Router)
- `react` — UI library
- `tailwindcss` — CSS framework

---

## Testing

### Go Tests

```bash
cd apps/api && go test ./...        # Run all
cd apps/api && go test ./internal/research/...  # Research package
cd apps/api && go test ./internal/security/...  # Security package
```

### Frontend

```bash
cd apps/web && pnpm build           # Production build
cd apps/web && pnpm lint            # Lint check
cd apps/web && npx tsc --noEmit     # Type check
```

### Full Regression

```bash
cd apps/api && go test ./... && go vet ./...
cd apps/web && pnpm lint && pnpm build
```

---

## Deployment

### Netlify

- **Build**: pnpm install → Next.js build → Go function compilation
- **Publish**: `apps/web/.next`
- **Functions**: `apps/api/function/` (Go, Lambda-compatible)
- **Plugin**: `@netlify/plugin-nextjs`
- **Redirects**: `/api/*` → `/.netlify/functions/api` (status 200)

### Known Issue

The `@netlify/plugin-nextjs` intercepts all routes including `/.netlify/functions/*`, preventing the Go function from being served. See `HANDOFF.md` for details.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────┐
│                    Frontend                       │
│  Next.js App Router (apps/web)                   │
│  SearchForm → ResultCard → TrustScore             │
│              → EvidenceList → ReasoningTimeline    │
│              → ExportMenu → HistoryList            │
└───────────────────┬─────────────────────────────┘
                    │ POST /api/verify
                    ▼
┌─────────────────────────────────────────────────┐
│                  Netlify                          │
│  @netlify/plugin-nextjs (static)                 │
│  Go Function: function/api.go (Lambda adapter)   │
└───────────────────┬─────────────────────────────┘
                    │ Gin router
                    ▼
┌─────────────────────────────────────────────────┐
│               Go Backend (apps/api)              │
│                                                   │
│  classifier → verifier (URL/email/IP/phone/text) │
│            → research (DuckDuckGo + Wikipedia)   │
│            → scoring (evidence → score)           │
│            → analysis (pipeline + timeline)       │
│            → claims (multi-claim extraction)      │
│            → security (code + dependency scan)    │
│            → perspectives (multi-viewpoint)       │
└─────────────────────────────────────────────────┘
```
