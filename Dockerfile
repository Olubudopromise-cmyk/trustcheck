# syntax=docker/dockerfile:1
# Frontend — Next.js 15 web app.
# Multi-stage production build. NEXT_PUBLIC_* values are inlined at build time,
# so NEXT_PUBLIC_API_URL must be passed as a build argument (docker-compose does
# this automatically). The runtime image installs only production dependencies
# and runs the production server (`pnpm start`), never `pnpm dev`.

FROM node:22-alpine AS builder
WORKDIR /repo

RUN corepack enable

ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

COPY . .

RUN pnpm install --frozen-lockfile --ignore-scripts
RUN pnpm --filter @trustcheck/web build

FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
RUN corepack enable

ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

COPY --from=builder /repo/package.json /repo/pnpm-workspace.yaml /repo/pnpm-lock.yaml ./
COPY --from=builder /repo/apps/web/package.json ./apps/web/package.json
RUN pnpm install --prod --frozen-lockfile --filter @trustcheck/web --ignore-scripts

COPY --from=builder /repo/apps/web/.next ./apps/web/.next
COPY --from=builder /repo/apps/web/next.config.ts ./apps/web/next.config.ts
COPY --from=builder /repo/apps/web/next-env.d.ts ./apps/web/next-env.d.ts

WORKDIR /app/apps/web

EXPOSE 3000

CMD ["pnpm", "start"]
