# syntax=docker/dockerfile:1
# Backend — Go + Gin verification API.
# Multi-stage build: the builder compiles a static binary that is copied into
# a minimal Alpine runtime image. No source code ships in the final image.

FROM golang:1.24-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /trustcheck-api .

FROM alpine:3.21

RUN adduser -D -u 10001 appuser
USER appuser

COPY --from=builder /trustcheck-api /usr/local/bin/trustcheck-api

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/trustcheck-api"]
