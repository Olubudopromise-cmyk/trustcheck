# TrustCheck

> **Can I trust this?** — One place to sanity-check domains, emails, IPs, phone numbers, and businesses before you trust them.

TrustCheck verifies a target, scores its trustworthiness, and explains the verdict with a transparent list of evidence. It combines a Go + Gin verification API with a responsive Next.js dashboard that supports single and batch verification, exportable reports, local verification history, and an analytics view computed entirely on-device.

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

## Environment variables

| Variable               | Scope | Default                 | Purpose                              |
| ---------------------- | ----- | ----------------------- | ------------------------------------ |
| `NEXT_PUBLIC_API_URL`  | web   | `http://localhost:8080` | Backend URL the frontend calls       |
| `NEXT_PUBLIC_SITE_URL` | web   | `http://localhost:3000` | Canonical site URL for SEO/OpenGraph |
| `ALLOWED_ORIGIN`       | api   | `http://localhost:3000` | CORS origin allowed to call the API  |

Copy `apps/web/.env.example` to `apps/web/.env.local` (or set the root `.env`) and adjust as needed.

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

## Roadmap

See [ROADMAP.md](ROADMAP.md) for the planned roadmap. Highlights include AI-generated trust explanations, community scam reports, address lookup, crypto wallet checks, and a public API.

## License

[MIT](LICENSE) © 2026 Olubudo Promise
