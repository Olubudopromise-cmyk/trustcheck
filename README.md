# TrustCheck

> **Can I trust this?** — One place to sanity-check domains, emails, IPs, phone numbers, and businesses before you trust them.

TrustCheck verifies a target, scores its trustworthiness, and explains the verdict with a transparent list of evidence. It combines a Go + Gin verification API with a responsive Next.js dashboard that supports single and batch verification, exportable reports, local verification history, and an analytics view computed entirely on-device.

## Live Demo

|            | URL                                                   |
| ---------- | ----------------------------------------------------- |
| Frontend   | `https://<site>.netlify.app` (set after first deploy) |
| API        | `https://<site>.netlify.app/api`                      |
| Swagger UI | `https://<site>.netlify.app/api/docs`                 |

> The frontend and API are served from the **same origin** on Netlify: the frontend calls the Go API through the `/api/*` rewrite, so there are no CORS or mixed-content concerns. Until deployed, run locally with [Running the backend](#running-the-backend) + [Running the frontend](#running-the-frontend).

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

TrustCheck is a pnpm/Turbo monorepo with two deployables, both hosted on **Netlify** as a single site:

```
┌───────────────────────────────────────────────────────────────┐
│                     Netlify site (same origin)                │
│                                                               │
│  Browser ──▶ Next.js web app ──▶ /api/* ──▶ Go API Function  │
│              apps/web (static)      rewrite    apps/api       │
│                                                               │
│  <site>.netlify.app                 /api/verify, /api/health  │
│                                     /api/docs (Swagger UI)    │
└───────────────────────────────────────────────────────────────┘
```

- The **web app** (`apps/web`) is a Next.js 15 app built to `.next` and served by Netlify's Next.js runtime.
- The **API** (`apps/api`) is a Go + Gin server compiled by Netlify as a **Function** mounted at `/api`. Both entry points share the exact same router (`internal/server`), so local behavior matches production.
- The frontend calls the API with a relative `/api/verify` path in production (see [`apps/web/src/utils/api.ts`](apps/web/src/utils/api.ts)); locally it falls back to `http://localhost:8080`.

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
| Hosting  | Netlify (site + Functions + Next.js runtime)   |
| Tooling  | pnpm workspaces, Turbo, ESLint, Prettier       |
| Storage  | Browser `localStorage` (history only)          |

## Folder structure

```
trustcheck/
├─ apps/
│  ├─ api/                      # Go + Gin backend (Go module)
│  │  ├─ cmd/api/               # local dev server entry point (go run ./cmd/api)
│  │  ├─ function/api/          # Netlify Function entry point (serverless)
│  │  └─ internal/
│  │     ├─ server/             # shared HTTP router (used by both entry points)
│  │     ├─ spec/               # embedded OpenAPI 3.1 specification
│  │     ├─ classifier/         # input type detection
│  │     ├─ verifier/           # per-type verification engines
│  │     ├─ scoring/            # trust score & evidence builder
│  │     ├─ logging/            # structured JSON request logging
│  │     └─ ratelimit/          # in-memory token-bucket rate limiter
│  └─ web/                      # Next.js 15 frontend
│     └─ src/
│        ├─ app/                # pages, layout, manifest, metadata, error pages
│        ├─ components/         # UI components
│        ├─ hooks/              # history + batch verification
│        ├─ utils/              # api client, analytics, report export, history, time helpers
│        └─ types.ts            # shared TypeScript types
├─ docs/                        # API, architecture, PRD, user-flow docs
└─ netlify.toml                 # Netlify build, functions, and redirect config
```

## Installation

Requires **Node.js ≥ 20**, **pnpm ≥ 9.15**, and **Go ≥ 1.23**.

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
go run ./cmd/api          # listens on :8080 (or $PORT)
```

The API responds on `/health`, `/verify`, `/docs`, and `/openapi.yaml`.

## Environment variables

All environment variables are read from the environment (or a `.env` file, which is git-ignored). Copy `.env.example` to `.env` and adjust as needed.

| Variable               | Scope | Default                                     | Purpose                                                                                                                       |
| ---------------------- | ----- | ------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `NEXT_PUBLIC_API_URL`  | web   | `''` (prod) / `http://localhost:8080` (dev) | Backend URL the browser calls. Leave **unset** in production so the Netlify Function is reached at `/api` on the same origin. |
| `NEXT_PUBLIC_SITE_URL` | web   | `http://localhost:3000`                     | Canonical site URL for SEO/OpenGraph metadata                                                                                 |
| `ALLOWED_ORIGIN`       | api   | `http://localhost:3000`                     | CORS origin allowed to call the API (local development only)                                                                  |

The API reads `PORT` (default `8080`) and `ALLOWED_ORIGIN` at runtime; it has no other secrets. Nothing sensitive is ever committed — see [.env.example](.env.example) and `apps/web/.env.example`.

## Deployment (Netlify)

TrustCheck deploys as a **single Netlify site** that serves both the Next.js frontend and the Go API. The site is configured entirely in [`netlify.toml`](netlify.toml): the build runs from the repository root (pnpm monorepo), publishes `apps/web/.next` through Netlify's Next.js runtime, and compiles the Go API from `apps/api/function` as a Function.

### Automatic deploys

Connect the GitHub repository to Netlify once, and deploys are automatic:

- **Production** — every push to `main` builds and deploys.
- **Preview** — every pull request gets its own deploy preview URL (with its own API Function).

No deployment scripts or CI deploy steps are required.

### Manual steps (one-time)

1. **Create the site** — Netlify → _Add new site_ → _Import an existing project_ → pick this repository.
2. **Verify build settings** — Netlify reads `netlify.toml` automatically. Confirm:
   - Build command: `pnpm install --frozen-lockfile --ignore-scripts && pnpm --filter @trustcheck/web build`
   - Publish directory: `apps/web/.next`
   - Functions directory: `apps/api/function`
3. **Set environment variables** in _Site configuration → Environment variables_:
   - `NEXT_PUBLIC_SITE_URL` → your production URL (e.g. `https://trustcheck.netlify.app`).
   - Leave `NEXT_PUBLIC_API_URL` **unset** so the frontend uses the same-origin `/api`.
4. **Deploy** — push to `main` (or trigger a deploy). After the first successful deploy:
   - The API is live at `https://<site>/api/health`.
   - Swagger UI is live at `https://<site>/api/docs`.
   - Update the [Live Demo](#live-demo) table with your URLs.

> **Note on deploy previews:** preview URLs work out of the box — each preview deploys its own API Function, so `NEXT_PUBLIC_API_URL` must remain unset (the same-origin `/api` resolves against the preview URL).

### How the API is served

Netlify compiles Go Functions from the `apps/api/function` directory using the Lambda Go runtime. The `api` function reuses the router from `internal/server` and is exposed at `/api/*` via the rewrite in `netlify.toml`:

```toml
[[redirects]]
  from = "/api/*"
  to = "/.netlify/functions/api"
  status = 200
```

Because Netlify 200-rewrites preserve the original request path, the function receives `event.path = /api/verify` and routes it exactly like the local server — including `/api/health`, `/api/docs`, and `/api/openapi.yaml`.

## API

### Endpoint

`POST /verify` — accepts a JSON body `{ "input": "<target>" }`.

Locally: `http://localhost:8080/verify` · On Netlify: `https://<site>/api/verify`

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

The API is described by an OpenAPI 3.1 specification at [`apps/api/internal/spec/openapi.yaml`](apps/api/internal/spec/openapi.yaml). An interactive Swagger UI is served by the backend itself (no CDN dependencies — the UI is bundled into the binary):

```
http://localhost:8080/docs          # local development
https://<site>.netlify.app/api/docs # production
```

The raw specification is served alongside it (`/openapi.yaml` locally, `/api/openapi.yaml` on Netlify), so you can pipe it into any OpenAPI-compatible tool, e.g. `curl http://localhost:8080/openapi.yaml | npx swagger-cli validate`.

> **Note:** the OpenAPI document's `servers` block points at the local development server. On Netlify the live endpoints live under `/api` — the paths in the spec are the logical API routes; the `/api` prefix is an infrastructure detail.

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

- `lint` — frontend (ESLint via Turbo)
- production `build` — Next.js production bundle
- `build` — Go backend compiles (including the Netlify Function entry point)
- `vet` — Go static analysis
- `test` — Go unit tests

The workflow uses pinned toolchains (Node 22, Go 1.23.3, pnpm 9.15) and caches the pnpm store, Go modules, and the Go build cache so installs are reproducible and runs stay fast. The job fails if any step fails.

## Structured Logging

Every API request produces **exactly one** structured JSON log line on stdout. Each request is tagged with a request ID that is also returned to the client in the `X-Request-ID` response header; if the client sends its own `X-Request-ID`, that value is reused. The raw input value is never logged to avoid exposing potentially sensitive data.

```json
{
  "timestamp": "2026-08-07T12:30:15Z",
  "level": "INFO",
  "msg": "request completed",
  "requestId": "c3ef4c8f1f2a3b4c5d6e7f809a1b2c3d4",
  "method": "POST",
  "path": "/api/verify",
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

On Netlify, the function's stdout appears in **Function logs** under _Site configuration → Logs → Functions_ (AWS CloudWatch behind the scenes), so you can correlate `X-Request-ID` values from the browser with server logs.

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

> **Serverless note:** on Netlify the API runs as a serverless Function, so the in-memory limiter is **per warm instance** rather than globally shared. It still protects individual instances from bursts; a truly global limit would require external storage.

Why it exists: `/verify` performs live DNS, HTTPS, and registry lookups, so it is the endpoint most worth protecting from accidental bursts, scraping, and abuse — without adding external infrastructure.

## Roadmap

See [ROADMAP.md](ROADMAP.md) for the planned roadmap. Highlights include AI-generated trust explanations, community scam reports, address lookup, crypto wallet checks, and a public API.

## License

[MIT](LICENSE) © 2026 Olubudo Promise
