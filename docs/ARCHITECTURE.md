# TrustCheck - System Architecture

## Version

0.1

---

# High-Level Architecture

```text
                User
                 │
                 ▼
         Next.js Web App
                 │
                 ▼
             API Gateway
                 │
 ┌───────────────┼────────────────┐
 │               │                │
 ▼               ▼                ▼
Verifier     AI Service      Auth Service
 │               │                │
 └───────┬───────┴────────────┬───┘
         ▼                    ▼
     PostgreSQL             Redis
         │
         ▼
 External APIs
```

---

# Components

## Frontend

Technology:

- Next.js
- TypeScript
- Tailwind CSS

Responsibilities:

- Search
- Display verification results
- User authentication
- Dashboard

---

## API

Technology:

- Go
- Gin

Responsibilities:

- Handle requests
- Validate input
- Call services
- Return responses

---

## Verifier Service

Responsibilities:

- Website checks
- Phone checks
- Email checks
- Business checks
- IP checks

---

## AI Service

Responsibilities:

- Explain results
- Summarize risks
- Recommend actions

---

## Database

PostgreSQL stores:

- Users
- Verification history
- Reports
- API keys

---

## Cache

Redis stores:

- Cached lookups
- Sessions
- Rate limiting

---

## External Providers

Examples:

- WHOIS
- DNS
- Google Safe Browsing
- VirusTotal
- AbuseIPDB

---

# Design Principles

- Modular
- Secure
- Scalable
- API-first
- AI-assisted
- Testable