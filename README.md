# TrustCheck

AI-powered platform for verifying websites, phone numbers, emails, businesses, domains, IPs, and more.

## Vision

Help anyone answer one question:

> Can I trust this?

## Monorepo structure

- apps/web: Next.js 15 frontend
- apps/api: Go + Gin backend
- packages/ui: shared UI components
- packages/types: shared TypeScript types
- packages/config: shared configuration values
- packages/utils: shared utilities

## Planned Features

- Website verification
- Domain lookup
- Phone number validation
- Email verification
- Business verification
- Address lookup
- IP intelligence
- Crypto wallet checks
- AI-generated trust explanations
- Community scam reports
- Public API

## Development

1. Install dependencies with pnpm install
2. Start the web app with pnpm --filter @trustcheck/web dev
3. Start the API with go run ./apps/api
4. Copy .env.example to .env and adjust values as needed

## Status

🚧 Under active development
