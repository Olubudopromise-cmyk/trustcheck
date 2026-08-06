# TrustCheck

> **Can I trust this?** — One place to sanity-check domains, emails, IPs, phone numbers, and businesses before you trust them.

TrustCheck verifies a target, scores its trustworthiness, and explains the verdict with a transparent list of evidence. It combines a Go + Gin verification API with a responsive Next.js dashboard that supports single and batch verification, exportable reports, local verification history, and an analytics view computed entirely on-device.

## Live Demo

|            | URL                                        |
| ---------- | ------------------------------------------ |
| Frontend   | https://trustcheck-web.up.railway.app      |
| API        | https://trustcheck-api.up.railway.app      |
| Swagger UI | https://trustcheck-api.up.railway.app/docs |

> **Note:** these URLs are provisioned when the project is first deployed to Railway (see [Deployment](#deployment)). Both services are served over HTTPS by Railway, so there are no mixed-content warnings. Until then, run locally with [Running the backend](#running-the-backend) + [Running the frontend](#running-the-frontend).

## Features

- **Multi-type verification** — domains, URLs, emails, IPv4/IPv6, phone numbers, and company names.
- **Transparent scoring** — every result carries a `trustScore` (0–100), a status, and ordered `evidence` explaining how the score was reached.
- **Batch verification** — paste up to 100 inputs at once; inputs are verified concurrently with live progress, then sortable and filterable in a results table.
- **Local history** — past verifications persist in `localStorage` (newest first, capped at 20) and can be reopened with one click.
- **Analytics dashboard** — status/type distribution, trust-score buckets, average score per type, recent streaks, and top verified types, all computed client-side.
- **Export** — download a JSON report per verification or for an entire batch; print-to-PDF report for single results.
- **Dark mode** — class-based theme toggle persisted across visits.
- **Accessible** — keyboard-navigable menus, sortable table headers, live-region progress announcements, and visible focus indicators.

## Architecture

```
┌───────────────────────────────┐         ┌───────────────────────────────┐
│         Web (Next.js 15)      │   HTTP  │          API (Go + Gin)       │
│      apps/web · :3000         │ ─────▶  │       apps/api · :8080        │
│                               │  POST   │                               │
│  SearchForm / BatchInput      │ /verify │  classify ─▶ verify ─▶ score  │
│  ResultCard / BatchResults    │         │                               │
│  History  ·  Analytics        │ ◀────── │  { status, trustScore,        │
│  Export JSON / PDF            │   JSON  │    summary, evidence[] }      │
└───────────────────────────────┘         └───────────────────────────────┘
```

Verification flow inside the API:

```
input ─▶ classifier (domain | url | email | ipv4 | ipv6 | company | phone | unknown)
      ─▶ verifier  (DNS, HTTP(S), TLS, SMTP/email rules, WHOIS/company, phone format, …)
      ─▶ scoring.Builder ─▶ status + trustScore + ordered evidence
      ─▶ JSON response
```

## Tech stack

| Layer    | Technology                                     |
| -------- | ---------------------------------------------- |
| Frontend | Next.js 15, React 19, TypeScript, Tailwind CSS |
| Backend  | Go, Gin, net/http                              |
| Tooling  | pnpm workspaces, Turbo, ESLint, Prettier       |
| Storage  | Browser `localStorage` (history only)          |

## Folder structure

```
trustcheck/
├─ apps/
│  ├─ api/                      # Go + Gin backend
│  │  ├─ main.go                # server, CORS, /health, /verify
│  │  └─ internal/
│  │     ├─ classifier/         # input type detection
│  │     ├─ verifier/           # per-type verification engines
│  │     └─ scoring/            # trust score & evidence builder
│  └─ web/                      # Next.js 15 frontend
│     └─ src/
│        ├─ app/                # pages, layout, manifest, metadata, error pages
│        ├─ components/         # UI components
│        ├─ hooks/              # history + batch verification
│        ├─ utils/              # analytics, report export, history, time helpers
│        └─ types.ts            # shared TypeScript types
├─ packages/                    # scaffolded shared workspaces
├─ docs/                        # API, architecture, PRD, user-flow docs
└─ infrastructure/              # deployment scaffolding
```

## Installation

Requires **Node.js ≥ 20**, **pnpm ≥ 9.15**, and **Go ≥ 1.22**.

```bash
git clone https://github.com/pamierin/trustcheck.git
cd trustcheck
pnpm install
```

## Running the frontend

```bash
# Development (http://localhost:3000)
pnpm --filter @trustcheck/web dev

# Production
pnpm --filter @trustcheck/web build
pnpm --filter @trustcheck/web start
```

## Running the backend

```bash
cd apps/api
go run .          # listens on :8080
```

## Docker

Run the whole stack (frontend + backend) with a single command:

```bash
docker compose up --build
```

- **Frontend:** http://localhost:3000
- **Backend:** http://localhost:8080

Both images use multi-stage production builds (the Go API compiles a static binary into a minimal Alpine runtime; the Next.js app builds the production bundle and runs `pnpm start`). Because `NEXT_PUBLIC_API_URL` is inlined at build time and the frontend fetches from the _browser_, the web image is built with `NEXT_PUBLIC_API_URL=http://localhost:8080` (the API's published host port) so browser requests reach the API.

> For a production deployment behind a reverse proxy, the cleanest setup is to have Nginx (or Caddy) terminate TLS and proxy `/verify` to the API service — then the frontend only ever calls relative `/verify` and no container service name leaks into the browser.

## Environment variables

| Variable               | Scope | Default                 | Purpose                              |
| ---------------------- | ----- | ----------------------- | ------------------------------------ |
| `NEXT_PUBLIC_API_URL`  | web   | `http://localhost:8080` | Backend URL the frontend calls       |
| `NEXT_PUBLIC_SITE_URL` | web   | `http://localhost:3000` | Canonical site URL for SEO/OpenGraph |
| `ALLOWED_ORIGIN`       | api   | `http://localhost:3000` | CORS origin allowed to call the API  |

Copy `apps/web/.env.example` to `apps/web/.env.local` (or set the root `.env`) and adjust as needed.

## Deployment

TrustCheck is deployed on **Railway** as two services in one project, built from the existing `Dockerfile`s. Both get HTTPS automatically from Railway (valid certificates on `*.up.railway.app`), so the frontend → API calls are HTTPS-to-HTTPS with no mixed-content warnings.

| Service            | Root Directory  | Config                                           | Port                       |
| ------------------ | --------------- | ------------------------------------------------ | -------------------------- |
| Backend (Go API)   | `apps/api`      | [`apps/api/railway.toml`](apps/api/railway.toml) | `$PORT` (Railway-injected) |
| Frontend (Next.js) | `/` (repo root) | [`railway.toml`](railway.toml)                   | 3000                       |

### Environment variables

| Variable              | Service | Purpose                                                                                                                            |
| --------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| `NEXT_PUBLIC_API_URL` | web     | Public API URL (no localhost). Inlined into the client bundle at build time; Railway passes it to the Dockerfile's matching `ARG`. |
| `ALLOWED_ORIGIN`      | api     | Public frontend URL, for CORS (e.g. `https://trustcheck-web.up.railway.app`).                                                      |

### Deploy steps

1. `railway login` (or set a `RAILWAY_TOKEN`).
2. `railway init` to create a project, then `railway link` to select a service for each directory.
3. From the repo root (frontend service): `railway up` — Root Directory `/`, config [`railway.toml`](railway.toml).
4. From `apps/api` (backend service): `railway up` — Root Directory `apps/api`, config [`apps/api/railway.toml`](apps/api/railway.toml).
5. Set the variables above on each service (dashboard or `railway variables`).
6. Confirm `https://<api-url>/health`, `https://<api-url>/docs`, and the frontend URL respond; then update the [Live Demo](#live-demo) table.

Alternatively, connect the GitHub repository and set each service's Root Directory to the values above — the `railway.toml` files are picked up automatically.

## API

### Endpoint

`POST /verify` — accepts a JSON body `{ "input": "<target>" }`.

### Example request

```bash
curl -X POST http://localhost:8080/verify \
  -H 'Content-Type: application/json' \
  -d '{"input":"google.com"}'
```

### Example response

```json
{
  "input": "google.com",
  "type": "domain",
  "status": "verified",
  "trustScore": 60,
  "summary": "Domain resolves, HTTPS available, certificate valid.",
  "evidence": [
    { "label": "DNS Resolves", "result": "pass", "points": 0 },
    { "label": "HTTPS Available", "result": "pass", "points": 20 },
    { "label": "TLS Certificate Present", "result": "pass", "points": 20 },
    { "label": "HTTP Status OK", "result": "pass", "points": 20 }
  ]
}
```

`type` is one of `domain`, `url`, `email`, `ipv4`, `ipv6`, `company`, `phone`, `unknown`. `status` is one of `verified`, `warning`, `invalid`, `private`, `local`, `unreachable`, `suggestion`, `unknown`, `not_implemented`. Evidence `result` is one of `pass`, `warning`, `fail`, `info`.

There is also a health check: `GET /health` → `{ "status": "ok", "service": "trustcheck-api" }`.

## API Documentation

The API is described by an OpenAPI 3.1 specification at [`apps/api/openapi.yaml`](apps/api/openapi.yaml). An interactive Swagger UI is served by the backend itself (no CDN dependencies — the UI is bundled into the binary) at:

```
http://localhost:8080/docs
```

The raw specification is also served at `http://localhost:8080/openapi.yaml`, so you can pipe it into any OpenAPI-compatible tool, e.g. `curl http://localhost:8080/openapi.yaml | npx swagger-cli validate`.

## Screenshots

| Single verification      | Batch verification       | Analytics dashboard      |
| ------------------------ | ------------------------ | ------------------------ |
| _Screenshot coming soon_ | _Screenshot coming soon_ | _Screenshot coming soon_ |

## Testing

```bash
# Frontend lint + production build
pnpm --filter @trustcheck/web lint
pnpm --filter @trustcheck/web build

# Backend
cd apps/api
go build ./...
go vet ./...
go test ./...
```

## Continuous Integration

Every push and pull request to `main` runs the full quality gate on GitHub Actions (see [`.github/workflows/ci.yml`](.github/workflows/ci.yml)):

- `lint` — frontend and shared packages (ESLint via Turbo)
- production `build` — Next.js production bundle
- `build` — Go backend compiles
- `vet` — Go static analysis
- `test` — Go unit tests

The workflow uses pinned toolchains (Node 22, Go 1.24, pnpm 9.15) and caches the pnpm store, Go modules, and the Go build cache so installs are reproducible and runs stay fast. The job fails if any step fails.

## Structured Logging

Every API request produces **exactly one** structured JSON log line on stdout. Each request is tagged with a request ID that is also returned to the client in the `X-Request-ID` response header; if the client sends its own `X-Request-ID`, that value is reused. The raw input value is never logged to avoid exposing potentially sensitive data.

```json
{
  "timestamp": "2026-08-07T12:30:15Z",
  "level": "INFO",
  "msg": "request completed",
  "requestId": "c3ef4c8f1f2a3b4c5d6e7f809a1b2c3d4",
  "method": "POST",
  "path": "/verify",
  "status": 200,
  "latencyMs": 18,
  "clientIP": "127.0.0.1",
  "userAgent": "curl/8.8.0",
  "inputType": "domain",
  "verificationStatus": "verified",
  "trustScore": 60
}
```

`/verify` requests additionally include `inputType`, `verificationStatus`, and `trustScore` (from the verification result, never the raw input). Requests that complete with an HTTP status of 400 or higher are logged at `ERROR` level with an additional `error` field describing the failure.

## Rate Limiting

`POST /verify` is protected by a lightweight in-memory token-bucket rate limiter (Go standard library only — no Redis, no external packages), keyed per client IP:

- **Rate:** 60 requests per minute (refill of 1 token per second)
- **Burst:** 20 (a client may send up to 20 requests immediately, then must wait for tokens to refill)

`/health`, `/docs`, and `/openapi.yaml` are never rate limited. When the limit is exceeded, the API responds with HTTP `429 Too Many Requests`:

```text
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 1

{"error":"rate limit exceeded"}
```

The `Retry-After` header is the number of seconds until the client may send another request. Idle clients are removed periodically by a background cleanup goroutine, so memory usage stays bounded. 429 responses appear in the structured request logs like any other response.

Why it exists: `/verify` performs live DNS, HTTPS, and registry lookups, so it is the endpoint most worth protecting from accidental bursts, scraping, and abuse — without adding external infrastructure.

## Roadmap

See [ROADMAP.md](ROADMAP.md) for the planned roadmap. Highlights include AI-generated trust explanations, community scam reports, address lookup, crypto wallet checks, and a public API.

## License

[MIT](LICENSE) © 2026 Olubudo Promise
