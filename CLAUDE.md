# CLAUDE.md — Spatial Memory Network

## What This Is

Backend API for a Spatial Memory Network — an AR app that lets users pin multimedia memories (photos, videos, voice, text) to real-world geographic locations. Others must physically visit the location to discover and view those memories. "空间版小红书 / Pinterest for the physical world."

This repo contains the Go backend API. Mobile app (Flutter + ARCore/ARKit) is a separate repo (future).

## Tech Stack

- **Language**: Go 1.22+
- **HTTP Framework**: Gin
- **Database**: PostgreSQL 16 + PostGIS 3.4
- **Cache**: Redis 7
- **Object Storage**: Cloudflare R2 (S3-compatible)
- **AI Moderation**: GLM-4V (ZhipuAI)
- **Auth**: JWT (golang-jwt) + SMS + WeChat OAuth
- **Migrations**: golang-migrate
- **Logging**: zerolog
- **Config**: Viper

## Quick Start

```bash
make docker-up      # Start PostgreSQL + Redis
make migrate-up     # Run migrations
make dev            # Start API with hot-reload (air)
```

## Project Structure

```
spatial-memory/
├── cmd/server/main.go           # Entry point, DI wiring
├── internal/
│   ├── config/                  # Viper config
│   ├── database/                # DB + Redis + migrations
│   ├── model/                   # Domain models
│   ├── repository/              # Database access (pgx, raw SQL)
│   ├── service/                 # Business logic
│   ├── handler/                 # HTTP handlers (Gin)
│   ├── middleware/              # Auth, rate-limit, logging
│   └── pkg/                    # Shared utilities (R2, SMS, WeChat, GLM, response helpers)
├── migrations/                  # SQL migration files
├── tests/integration/           # E2E API tests (testcontainers)
├── docs/
│   ├── specs/                   # Product design spec
│   └── plans/                   # Implementation plans
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── go.mod
```

## Architecture

Clean layered architecture: **handler → service → repository**

- **No ORM** — PostGIS spatial queries need hand-written SQL via pgx
- **Two-phase upload** — clients upload directly to R2 via pre-signed URLs, never through API
- **Redis GEO cache** — hot-zone spatial queries cached with GEOADD/GEOSEARCH
- **Background moderation** — public memories queue for GLM-4V AI review

## Key Design Decisions

- **Discovery > Creation**: spatial queries are the most performance-critical path
- **Progressive visibility**: private → circle (authorized friends) → public
- **"Must be present"**: proximity verification is core differentiator, never compromise
- **Content moderation**: legally required in China for public UGC content

## Coding Conventions

- **Naming**: Go standard (PascalCase exports, camelCase private, snake_case files)
- **Errors**: Domain error types in `internal/pkg/errors/`, wrap with context
- **Testing**: Unit tests with testify mocks, integration tests with testcontainers
- **SQL**: Raw SQL with pgx, parameterized queries only (no string interpolation)
- **Commits**: Conventional commits (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `infra:`)

## Implementation Plan

See `docs/plans/2026-03-31-backend-api-plan.md` for the full implementation plan with 7 chunks and detailed tasks.

## Design Spec

See `docs/specs/2026-03-31-spatial-memory-network-design.md` for the complete product design document.
