# TrustCheck — Handoff Document

> **Created**: August 8, 2026
> **Author**: Buffy (AI agent)
> **Status**: Netlify deployment is broken. Local development works perfectly.

---

## TL;DR

TrustCheck's Go backend works perfectly locally (localhost:8080), all tests pass, the frontend builds and typechecks. But the **Netlify production deployment is non-functional** — the Go serverless function is never deployed. All requests to `/api/*` and `/.netlify/functions/*` return the Next.js frontend HTML instead of API responses.

**The `@netlify/plugin-nextjs` intercepts ALL routes including `/.netlify/functions/*`**, preventing the Go function from ever being reached. This is the single blocking issue.

---

## 1. Codebase Index

### Project Structure (pnpm monorepo)

```
trustcheck/
├── apps/
│   ├── api/                          # Go backend
│   │   ├── cmd/api/main.go           # Local dev server entry point
│   │   ├── function/api.go           # Netlify Function entry point (Lambda adapter)
│   │   ├── go.mod                    # Go module: github.com/pamierin/trustcheck/apps/api
│   │   ├── go.sum
│   │   └── internal/
│   │       ├── analysis/             # Evidence-driven analysis pipeline
│   │       │   ├── pipeline.go       # Main analysis orchestrator
│   │       │   ├── timeline.go       # Reasoning timeline builder
│   │       │   └── analysis.go       # Core analysis logic
│   │       ├── claims/               # Claim extraction from text
│   │       │   └── claims.go
│   │       ├── classifier/           # Input type classification (URL/email/IP/text)
│   │       │   └── classifier.go
│   │       ├── docs/                 # Swagger/OpenAPI docs
│   │       │   └── docs.go
│   │       ├── interpretations/      # Multi-perspective analysis
│   │       │   └── interpretations.go
│   │       ├── logging/              # Request logging middleware
│   │       │   ├── logger.go
│   │       │   └── middleware.go
│   │       ├── model/                # Shared data models
│   │       │   └── model.go          # Verdict, EvidenceItem, ScoreBreakdown, etc.
│   │       ├── perspectives/         # Alternative viewpoints
│   │       │   └── perspectives.go
│   │       ├── ratelimit/            # Rate limiting middleware
│   │       │   ├── limiter.go
│   │       │   └── middleware.go
│   │       ├── reasoning/            # Reasoning chain generation
│   │       │   └── reasoning.go
│   │       ├── recommendations/      # Actionable recommendations
│   │       │   └── recommendations.go
│   │       ├── research/             # *** NEW: Evidence retrieval layer ***
│   │       │   ├── provider.go       # SearchProvider interface
│   │       │   ├── duckduckgo.go     # DuckDuckGo HTML scraping provider
│   │       │   ├── wikipedia.go      # Wikipedia API provider
│   │       │   ├── multi_provider.go # Composite provider (dedup + classify)
│   │       │   ├── engine.go         # Research engine (query generation, orchestration)
│   │       │   ├── engine_test.go
│   │       │   └── module.go         # HTTP module for /research endpoint
│   │       ├── scoring/              # Trust score calculation from evidence
│   │       │   ├── scoring.go        # Score calculation
│   │       │   ├── classify.go       # Evidence classification
│   │       │   └── evidence.go       # Evidence summary building
│   │       ├── security/             # *** NEW: Security intelligence engine ***
│   │       │   ├── analyzer.go       # Code security scanner
│   │       │   ├── analyzer_test.go
│   │       │   ├── dependency_analyzer.go  # Go/Node dependency scanning
│   │       │   ├── dependency_analyzer_test.go
│   │       │   └── endpoint_test.go
│   │       ├── server/               # HTTP router (Gin)
│   │       │   └── server.go         # Routes: GET /health, POST /verify, POST /security
│   │       ├── spec/                 # OpenAPI spec
│   │       │   ├── spec.go
│   │       │   └── openapi.yaml
│   │       ├── verifier/             # Type-specific verifiers (URL, email, IP, phone, company, domain)
│   │       │   ├── dispatcher.go     # Routes to correct verifier
│   │       │   ├── url.go            # URL safety checks (DNS, HTTP, TLS)
│   │       │   ├── email.go          # Email verification
│   │       │   ├── ip.go             # IP reputation
│   │       │   ├── phone.go          # Phone validation
│   │       │   ├── domain.go         # Domain checks
│   │       │   ├── company.go        # Company verification
│   │       │   └── unknown.go        # Unknown type handling
│   │       └── warnings/             # Risk signal detection
│   │           └── warnings.go
│   │
│   └── web/                          # Next.js frontend
│       ├── src/
│       │   ├── app/
│       │   │   ├── page.tsx          # Main page (single-page app)
│       │   │   ├── layout.tsx        # Root layout
│       │   │   ├── error.tsx         # Error boundary
│       │   │   ├── not-found.tsx     # 404 page
│       │   │   ├── globals.css       # Global styles (Tailwind)
│       │   │   └── manifest.ts       # PWA manifest
│       │   ├── components/           # 30+ React components
│       │   │   ├── SearchForm.tsx    # Main input form
│       │   │   ├── ResultCard.tsx    # Verification result display
│       │   │   ├── TrustScore.tsx    # Score visualization
│       │   │   ├── EvidenceList.tsx  # Supporting/contradicting evidence
│       │   │   ├── ReasoningTimeline.tsx  # Step-by-step reasoning
│       │   │   ├── BatchInput.tsx    # Multi-input verification
│       │   │   ├── BatchResults.tsx  # Batch results display
│       │   │   ├── AnalyticsDashboard.tsx  # Analytics
│       │   │   ├── ExportMenu.tsx    # PDF/JSON export
│       │   │   ├── HistoryList.tsx   # Verification history
│       │   │   ├── ThemeToggle.tsx   # Dark/light mode
│       │   │   └── ... (20+ more)
│       │   ├── hooks/
│       │   │   ├── useVerificationHistory.ts
│       │   │   └── useBatchVerification.ts
│       │   ├── utils/
│       │   │   ├── api.ts            # API client (API_BASE_URL = '/api' in prod)
│       │   │   ├── report.ts         # Report generation
│       │   │   ├── analytics.ts      # Client analytics
│       │   │   ├── history.ts        # Local history management
│       │   │   └── relativeTime.ts   # Time formatting
│       │   └── types.ts              # TypeScript types (VerifyResponse, etc.)
│       ├── tailwind.config.ts
│       ├── next.config.ts
│       └── package.json
│
├── netlify.toml                      # Netlify deployment config
├── package.json                      # Root workspace package
├── pnpm-workspace.yaml               # pnpm workspace config
├── pnpm-lock.yaml
├── turbo.json                        # Turborepo config
└── .github/workflows/ci.yml          # CI pipeline
```

### Key API Routes (server.go)

| Method | Path            | Description                                          |
| ------ | --------------- | ---------------------------------------------------- |
| GET    | `/health`       | Health check                                         |
| GET    | `/openapi.json` | OpenAPI spec                                         |
| GET    | `/docs`         | Swagger UI                                           |
| POST   | `/verify`       | Main verification endpoint (accepts `input`, `mode`) |
| POST   | `/security`     | Security analysis endpoint                           |

### Frontend API Client (api.ts)

- **Production**: `API_BASE_URL = '/api'` (same-origin, expects Go function at `/api/*`)
- **Development**: `API_BASE_URL = 'http://localhost:8080'`
- **Override**: `NEXT_PUBLIC_API_URL` env var

---

## 2. The Blocking Problem: Netlify Go Function Not Deploying

### What Happens

| Path                                 | Expected                  | Actual                      |
| ------------------------------------ | ------------------------- | --------------------------- |
| `GET /api/health`                    | Go function JSON response | Next.js HTML (SPA fallback) |
| `POST /api/verify`                   | Go function JSON response | "Page not found" HTML       |
| `GET /.netlify/functions/api/health` | Go function JSON response | Next.js HTML                |
| `GET /.netlify/functions/`           | Function listing          | "Page not found" HTML       |

**The Go function is NEVER being deployed by Netlify.** All responses are HTML from the Next.js SPA.

### Root Cause

The `@netlify/plugin-nextjs` plugin intercepts ALL requests, including `/.netlify/functions/*`. Even though `netlify.toml` defines redirects before the plugin's redirects, the plugin still wins. This is a **known interaction issue** between `@netlify/plugin-nextjs` and Netlify Functions.

### What Was Already Tried (All Failed)

1. **`/trustcheck-api` prefix** (commit `4398aea`) — Still intercepted by Next.js plugin
2. **Flattening function** from `function/api/main.go` to `function/api.go` (commit `663c155`) — Compiles locally, but Netlify still doesn't deploy it
3. **Adding `go.mod` inside function directory** — Failed because `internal` packages can't be imported cross-module
4. **Pre-compiling Go binary in build command** (commit `56f41df`) — Netlify doesn't use pre-compiled binaries for Go functions
5. **Various redirect configurations** — All intercepted by the Next.js plugin

### What's Correct

- **`go.mod`** at `apps/api/go.mod` — single Go module for everything
- **Function** at `apps/api/function/api.go` — single file, compiles locally with `go build -o /dev/null ./function/api.go`
- **Function imports** `internal/server` for the Gin router + Lambda adapter
- **`netlify.toml`** config: `functions.directory = "apps/api/function"`, `GO_VERSION = "1.23.3"`
- **Build command**: `pnpm install --frozen-lockfile --ignore-scripts && pnpm --filter @trustcheck/web build && cd apps/api && go build -o function/api ./function/api.go`

---

## 3. The Netlify Configuration Problem

### netlify.toml (current)

```toml
[build]
  command = "pnpm install --frozen-lockfile --ignore-scripts && pnpm --filter @trustcheck/web build && cd apps/api && go build -o function/api ./function/api.go"
  publish = "apps/web/.next"

[build.environment]
  NETLIFY_USE_PNPM = "true"
  NODE_VERSION = "22"
  GO_VERSION = "1.23.3"

[functions]
  directory = "apps/api/function"

[[plugins]]
  package = "@netlify/plugin-nextjs"

[[redirects]]
  from = "/api/*"
  to = "/.netlify/functions/api"
  status = 200
[[redirects]]
  from = "/api"
  to = "/.netlify/functions/api"
  status = 200
```

### The Conflict

The `@netlify/plugin-nextjs` plugin generates its own set of redirects during build. These plugin-generated redirects override the manually-defined ones in `netlify.toml`. Specifically, the Next.js plugin:

1. Serves all pages through the Next.js runtime
2. Catches `/.netlify/functions/*` paths with its own catch-all
3. Returns `text/html` for any path not matching a Next.js page

### Possible Solutions to Investigate

1. **Exclude `/.netlify/functions/*` from the Next.js plugin** — Check if the plugin has a config option to skip certain paths
2. **Use `[[redirects]]` with `force = true`** — This may bypass the plugin's redirects
3. **Move the function outside the Next.js plugin's scope** — Different functions directory location
4. **Use Netlify Edge Functions instead** — May not conflict with the Next.js plugin
5. **Use a Next.js API route as a proxy** — Route `/api/*` through Next.js to a separate Go server
6. **Deploy the Go API separately** — e.g., on Railway, Fly.io, or AWS Lambda, and have the frontend call an external URL
7. **Check Netlify support/docs for known workaround** — This may be a documented issue

---

## 4. What Works Perfectly (Don't Touch)

### Local Development

```bash
# Start Go API server
cd apps/api && go run ./cmd/api/main.go
# Server running at localhost:8080

# Start Next.js dev server
cd apps/web && pnpm dev
# Frontend at localhost:3000
```

- All API endpoints work at `localhost:8080`
- Evidence gathering works (DuckDuckGo + Wikipedia)
- Trust scores are calculated from real evidence
- All 3 analysis modes work (Quick, Deep Research, Government)
- Security analysis works
- Multi-claim extraction works

### Go Tests

```bash
cd apps/api && go test ./...    # ALL PASS
cd apps/api && go vet ./...     # PASSES
cd apps/api && go build ./...   # BUILDS OK
```

### Frontend Build

```bash
cd apps/web && pnpm build       # BUILDS OK
cd apps/web && pnpm lint        # PASSES
cd apps/web && npx tsc --noEmit # TYPECHECKS OK
```

### Existing Packages

| Package     | Status                | Description                                                                                       |
| ----------- | --------------------- | ------------------------------------------------------------------------------------------------- |
| `research/` | ✅ Complete           | SearchProvider interface, DuckDuckGo + Wikipedia providers, multi-provider dedup, research engine |
| `security/` | ✅ Complete           | Code analyzer, dependency analyzer (Go + Node.js)                                                 |
| `scoring/`  | ✅ Complete           | Evidence-driven score calculation, classification                                                 |
| `analysis/` | ✅ Complete           | Pipeline, timeline, multi-mode analysis                                                           |
| `claims/`   | ✅ Complete           | Claim extraction from text                                                                        |
| `model/`    | ⚠️ Has staged changes | Initial commit has duplicate Claim/StatusFromVerdict (needs cleanup)                              |

---

## 5. What Needs to Be Done

### Priority 1: Fix Netlify Deployment (BLOCKING)

The Go function must actually deploy. Investigate and fix one of:

- `@netlify/plugin-nextjs` conflict resolution
- Edge Functions as alternative
- Next.js API route proxy
- External Go server deployment

**Verification**: After fix, these must return JSON (not HTML):

```bash
curl https://trust-check.netlify.app/api/health
curl -X POST https://trust-check.netlify.app/api/verify -H 'Content-Type: application/json' -d '{"input":"google.com"}'
```

### Priority 2: Clean Up model.go

There are duplicate `Claim` struct definitions and a duplicate `StatusFromVerdict` function in the initial commit's staged changes. These need to be resolved before any new work.

### Priority 3: Wire Research Package into Pipeline

The `research/` package exists but may not be fully integrated into the analysis pipeline for text claims. Verify that:

- Text claims go through the research engine
- Evidence is gathered from DuckDuckGo + Wikipedia
- Results include real evidence (not just verifier checks)

### Priority 4: Wire Security Package into Pipeline

The `security/` package exists but needs to be wired into the `POST /security` endpoint properly.

---

## 6. File Reference

### Critical Files

| File                                     | Purpose                                                          |
| ---------------------------------------- | ---------------------------------------------------------------- |
| `netlify.toml`                           | Netlify deployment config (build, functions, redirects, plugins) |
| `apps/api/function/api.go`               | Netlify Function entry point (Lambda adapter)                    |
| `apps/api/cmd/api/main.go`               | Local dev server entry point                                     |
| `apps/api/internal/server/server.go`     | Gin HTTP router with all routes                                  |
| `apps/api/internal/analysis/pipeline.go` | Main analysis pipeline                                           |
| `apps/api/internal/research/engine.go`   | Evidence retrieval engine                                        |
| `apps/api/internal/research/provider.go` | SearchProvider interface                                         |
| `apps/api/internal/scoring/scoring.go`   | Trust score calculation                                          |
| `apps/api/internal/model/model.go`       | Shared data models                                               |
| `apps/web/src/utils/api.ts`              | Frontend API client                                              |
| `apps/web/src/app/page.tsx`              | Main page component                                              |
| `apps/web/src/types.ts`                  | TypeScript types                                                 |

### Git History (Recent)

```
56f41df fix: compile Go function during build and use /api redirect
663c155 fix: flatten function to single file so Netlify can compile Go function
4398aea fix: use /trustcheck-api prefix to bypass Next.js plugin interception
bdfce8f chore: remove debug test file
5859ef2 revert: restore original netlify.toml and remove speculative changes
1339a0a feat: evidence-driven verification with multi-provider search architecture
ab16186 feat: security intelligence engine, dependency scanning, analysis modes, and UI components
```

---

## 7. Environment & Tooling

- **Go**: 1.23.0 (go.mod) / 1.23.3 (Netlify GO_VERSION)
- **Node**: 22 (Netlify NODE_VERSION)
- **Package Manager**: pnpm (monorepo with workspaces)
- **Framework**: Next.js (App Router) + Gin (Go)
- **Deployment**: Netlify (static + functions)
- **CI**: GitHub Actions (`.github/workflows/ci.yml`)

---

## 8. Honest Assessment

### What's Good

- The Go backend is well-architected with clean separation of concerns
- The research package has a proper `SearchProvider` interface
- The security package handles dependency scanning
- All tests pass
- The frontend is polished with 30+ components

### What's Broken

- Netlify deployment is completely non-functional for the API
- The model.go has duplicate definitions that need cleanup

### What's Missing

- Research engine may not be fully wired into the analysis pipeline
- Security endpoint may not be fully wired up
- No actual Netlify Function testing (only local `go test`)
- No integration tests for the deployed function
