# Spatial Memory Network — Backend API Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the backend API and database layer for a Spatial Memory Network — an AR app that lets users pin memories to physical locations and discover others' memories by visiting those locations.

**Architecture:** Go REST API with PostGIS for spatial queries, Cloudflare R2 for media storage, Redis for spatial caching. Clean layered architecture: handler → service → repository. JWT auth with SMS + WeChat login. GLM-4V AI moderation for public content.

**Tech Stack:** Go 1.22+, Gin, pgx v5, PostGIS, Redis, Cloudflare R2 (S3-compatible), golang-jwt, golang-migrate, zerolog, Viper, Swagger/swaggo

**Spec:** `docs/superpowers/specs/2026-03-31-spatial-memory-network-design.md`

**New repo:** Create at `~/projects/spatial-memory/` (independent from Game Studios and TripMeta)

---

## Context

This is Phase 1 of a startup MVP for the Spatial Memory Network ("空间小红书"). The full product involves a phone AR app + future AR glasses upgrade, but this plan covers only the backend infrastructure that all clients will depend on. The mobile app and AR layer will be separate plans.

Key product decisions driving this backend:
- Discovery is the core experience (spatial queries are the most performance-critical path)
- Progressive disclosure visibility model (private → circle → public)
- "Must be present" proximity verification is core differentiator
- Public memories require content moderation (legally mandated in China)
- Two-phase media upload via pre-signed URLs (never proxy through API server)

---

## Technology Choices

| Layer | Choice | Why |
|-------|--------|-----|
| HTTP | `gin-gonic/gin` | Largest ecosystem, excellent performance, built-in validation |
| PostgreSQL | `jackc/pgx` v5 + raw SQL | Fastest driver, native PostGIS type support. No ORM — spatial queries need hand-written SQL |
| Migrations | `golang-migrate/migrate` v4 | Industry standard, embeddable |
| Redis | `redis/go-redis` v9 | Full GEO command support (GEOADD, GEOSEARCH) |
| JWT | `golang-jwt/jwt` v5 | Standard, supports refresh token rotation |
| Logging | `rs/zerolog` | Zero-allocation JSON logger |
| Config | `spf13/viper` | Env-based config with `.env` fallback |
| S3/R2 | `aws/aws-sdk-go-v2` | R2 is S3-compatible |
| Rate Limit | `ulule/limiter` v3 | Redis-backed sliding window |
| API Docs | `swaggo/swag` | Generates OpenAPI from annotations |
| Testing | `stretchr/testify` + `testcontainers-go` | Assertions + real DB/Redis in Docker for integration tests |

---

## Project Structure

```
spatial-memory/
├── cmd/server/main.go           # Entry point, DI wiring, graceful shutdown
├── internal/
│   ├── config/config.go         # Viper-based config struct
│   ├── database/
│   │   ├── postgres.go          # pgxpool setup
│   │   ├── redis.go             # go-redis setup
│   │   └── migrate.go           # Migration runner
│   ├── model/                   # Domain models (User, Memory, Circle, etc.)
│   ├── repository/              # Database access (one file per entity)
│   ├── service/                 # Business logic (one file per domain)
│   ├── handler/                 # HTTP handlers (one file per route group)
│   ├── middleware/               # Auth, rate-limit, logging, admin, CORS
│   └── pkg/
│       ├── response/            # Standardized API response helpers
│       ├── errors/              # Domain error types
│       ├── storage/             # R2/S3 client
│       ├── sms/                 # SMS provider client
│       ├── wechat/              # WeChat OAuth client
│       └── moderation/          # GLM-4V moderation client
├── migrations/                  # SQL migration files
├── tests/integration/           # End-to-end API tests
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── .github/workflows/ci.yml
└── go.mod
```

---

## Dependency Graph

```
Chunk 1 (Scaffolding)
  └── Chunk 2 (Auth)
       └── Chunk 3 (Memories + Spatial)
            ├── Chunk 4 (Uploads)     ← parallelizable
            ├── Chunk 5 (Social)      ← parallelizable
            └── Chunk 6 (Moderation)  ← parallelizable
                 └── Chunk 7 (Docs + CI)
```

---

## Chunk 1: Project Scaffolding + Docker + Database (Days 1-3)

### Task 1.1: Initialize Go project and directory structure

**Files:** Create all directories and skeleton files per project structure above.

- [ ] `mkdir -p ~/projects/spatial-memory && cd ~/projects/spatial-memory && go mod init github.com/spatial-memory/spatial-memory`
- [ ] Create `cmd/server/main.go` — minimal main that loads config, creates Gin router, registers `/health`, starts server with graceful shutdown
- [ ] Create `internal/config/config.go` — Viper config struct (Server, Database, Redis, R2, JWT, SMS, WeChat, GLM sections). Load from env vars with `SPATIAL_` prefix, `.env` fallback
- [ ] Create `internal/pkg/response/response.go` — `Success()`, `Error()`, `Paginated()` helpers
- [ ] Create `internal/pkg/errors/errors.go` — domain errors (ErrNotFound, ErrUnauthorized, ErrForbidden, ErrValidation)
- [ ] Create `internal/handler/health.go` — `GET /health` with DB + Redis connectivity checks
- [ ] Create `.gitignore`, `.env.example`
- [ ] Run: `go build ./cmd/server` — verify compiles
- [ ] Commit: `scaffold: initialize Go project structure`

### Task 1.2: Docker Compose for local development

**Files:** `docker-compose.yml`, `Dockerfile`, `Dockerfile.dev`, `.air.toml`

- [ ] Create `docker-compose.yml` with: `postgis/postgis:16-3.4` (port 5432), `redis:7-alpine` (port 6379), `api` service with hot-reload
- [ ] Create `Dockerfile` — multi-stage build: `golang:1.22-alpine` → `alpine:3.19` (~15MB image)
- [ ] Create `Dockerfile.dev` with `air` for hot-reload
- [ ] Run: `docker-compose up -d && curl localhost:8080/health` — verify services start
- [ ] Commit: `infra: add Docker Compose with PostGIS and Redis`

### Task 1.3: Database connection and migration framework

**Files:** `internal/database/postgres.go`, `redis.go`, `migrate.go`

- [ ] Implement `pgxpool.Pool` creation (min 2, max 20 conns, 30m lifetime)
- [ ] Implement Redis client with ping-on-startup
- [ ] Implement migration runner using `golang-migrate` with `embed.FS`
- [ ] Run: `make migrate-up` — verify succeeds
- [ ] Commit: `infra: add database pool, Redis client, migration framework`

### Task 1.4: Initial database schema (V001)

**Files:** `migrations/000001_initial_schema.up.sql`, `000001_initial_schema.down.sql`

- [ ] Create schema with all tables: `users`, `memories` (with `GEOGRAPHY(POINT,4326)`), `memory_media`, `friend_circles`, `circle_members`, `memory_permissions`, `memory_interactions`, `moderation_queue`, `sms_codes`, `refresh_tokens`
- [ ] Create spatial index: `CREATE INDEX idx_memories_location ON memories USING GIST (location)`
- [ ] Create composite partial index: `idx_memories_active_public WHERE status=1 AND visibility=2`
- [ ] Create `updated_at` trigger function
- [ ] Run: `make migrate-up && psql -c "\dt"` — verify all tables exist, `SELECT PostGIS_Version()` returns version
- [ ] Commit: `schema: add initial PostGIS spatial tables`

### Task 1.5: Makefile

**Files:** `Makefile`

- [ ] Add targets: `dev`, `build`, `test`, `test-integration`, `migrate-up`, `migrate-down`, `migrate-create`, `lint`, `swagger`, `docker-up`, `docker-down`, `seed`
- [ ] Run: `make build` — produces binary
- [ ] Commit: `dx: add Makefile`

---

## Chunk 2: User Auth Service (Days 4-7)

### Task 2.1: Domain models

**Files:** `internal/model/user.go`, `internal/model/auth.go`

- [ ] Define `User`, `UserStatus`, `TokenPair`, `SMSCode`, `Claims` structs
- [ ] Commit: `model: add User and Auth domain models`

### Task 2.2: User repository

**Files:** `internal/repository/user_repo.go`, `user_repo_test.go`

- [ ] Interface: `Create`, `GetByID`, `GetByPhone`, `GetByWeChatOpenID`, `Update`
- [ ] Implementation with pgx
- [ ] Integration tests with testcontainers (PostGIS image)
- [ ] Run: `make test-integration` — tests pass
- [ ] Commit: `repo: implement UserRepository with integration tests`

### Task 2.3: Auth repository

**Files:** `internal/repository/auth_repo.go`, `auth_repo_test.go`

- [ ] SMS code CRUD + refresh token CRUD
- [ ] Integration tests for code lifecycle (create, fetch, mark used, expired handling)
- [ ] Commit: `repo: implement AuthRepository`

### Task 2.4: JWT token service

**Files:** `internal/service/token_service.go`, `token_service_test.go`

- [ ] `GenerateTokenPair` (access 2h, refresh 30d), `ValidateAccessToken`, `RefreshTokens` (rotation)
- [ ] Store refresh token hash (SHA-256), not raw token
- [ ] Unit tests: generate/validate round-trip, expired rejection, refresh rotation
- [ ] Commit: `service: implement JWT with refresh token rotation`

### Task 2.5: Auth service (SMS + WeChat)

**Files:** `internal/service/auth_service.go`, `auth_service_test.go`, `internal/pkg/sms/client.go`, `internal/pkg/wechat/client.go`

- [ ] `SendSMSCode`: rate limit (1/60s, 5/day per phone via Redis), generate 6-digit, 5min expiry
- [ ] `VerifySMSCode`: validate → find-or-create user → generate tokens
- [ ] `WeChatLogin`: exchange code → fetch user info → find-or-create → tokens
- [ ] Unit tests with mocked repos and clients
- [ ] Commit: `service: implement SMS and WeChat auth`

### Task 2.6: Auth + rate-limit middleware

**Files:** `internal/middleware/auth.go`, `ratelimit.go`, tests

- [ ] Auth middleware: extract Bearer token, validate JWT, set userID in context
- [ ] Rate limit: Redis sliding window, 10 req/min/IP for auth, 100 req/min/user for API
- [ ] Unit tests
- [ ] Commit: `middleware: implement JWT auth and rate limiting`

### Task 2.7: Auth + user HTTP handlers

**Files:** `internal/handler/auth_handler.go`, `user_handler.go`, tests

- [ ] Auth endpoints: `POST /api/v1/auth/sms/send`, `/sms/verify`, `/wechat`, `/refresh`, `/logout`
- [ ] User endpoints: `GET/PUT /api/v1/users/me`, `GET /api/v1/users/:id`
- [ ] Phone validation: `^\+86\d{11}$`
- [ ] Handler tests with mocked services
- [ ] Commit: `handler: implement auth and user profile endpoints`

### Task 2.8: Router wiring and DI

**Files:** `internal/router/router.go`, update `cmd/server/main.go`

- [ ] Wire all dependencies: config → db → redis → repos → services → handlers → router
- [ ] Apply middleware per route group
- [ ] Run: `make dev` — all auth endpoints respond
- [ ] Commit: `router: wire up auth and user routes with DI`

---

## Chunk 3: Memory CRUD + Spatial Queries (Days 8-14)

### Task 3.1: Memory models

**Files:** `internal/model/memory.go`

- [ ] Define `Memory`, `GeoPoint`, `Visibility`, `MemoryStatus`, `MemoryMedia`, `MediaType`
- [ ] Commit: `model: add Memory domain models`

### Task 3.2: Memory repository with PostGIS

**Files:** `internal/repository/memory_repo.go`, `memory_repo_test.go`

- [ ] CRUD: `Create`, `GetByID`, `Update`, `SoftDelete`, `ListByUser`
- [ ] **Critical**: `FindNearby` using `ST_DWithin` + `ST_Distance` on `geography` type with visibility filtering
- [ ] Media CRUD: `CreateMedia`, `DeleteMedia`, `GetMediaByContentHash`
- [ ] Counter updates: `IncrementLikes`, `IncrementViews`
- [ ] Integration tests: seed at known coordinates, verify radius filtering, distance ordering, visibility rules
- [ ] Performance test: 10K seeded memories, verify query < 50ms
- [ ] Commit: `repo: implement MemoryRepository with PostGIS spatial queries`

### Task 3.3: Redis spatial cache

**Files:** `internal/repository/spatial_cache.go`, `spatial_cache_test.go`

- [ ] Use Redis GEO: `GEOADD`, `GEOSEARCH` for spatial index
- [ ] Cache memory IDs → batch-fetch from PostgreSQL
- [ ] 5-minute TTL, invalidate on create/update/delete
- [ ] Integration tests with Redis container
- [ ] Commit: `cache: implement Redis GEO spatial cache`

### Task 3.4: Memory service

**Files:** `internal/service/memory_service.go`, `memory_service_test.go`

- [ ] `Create`: validate → insert → if public, set pending_review + queue moderation → add to cache
- [ ] `FindNearby`: cache-first → fallback to DB → populate cache
- [ ] `GetByID`: check access permissions before returning
- [ ] View count: async increment via goroutine
- [ ] Unit tests with mocked repo and cache
- [ ] Commit: `service: implement MemoryService with caching`

### Task 3.5: Spatial clustering

**Files:** `internal/service/cluster_service.go`, `cluster_service_test.go`

- [ ] When radius > 500m and count > 50: use `ST_ClusterDBSCAN` to return clusters
- [ ] Cluster model: centroid, count, sample thumbnail
- [ ] Unit tests
- [ ] Commit: `service: implement spatial clustering for dense areas`

### Task 3.6: Memory HTTP handlers

**Files:** `internal/handler/memory_handler.go`, `memory_handler_test.go`

- [ ] `POST /api/v1/memories`, `GET /:id`, `PUT /:id`, `DELETE /:id`
- [ ] `GET /api/v1/memories/nearby?lat=&lng=&radius=&sort=&limit=&cluster=`
- [ ] `GET /api/v1/memories/mine?page=&page_size=`
- [ ] Validation: lat(-90,90), lng(-180,180), radius(10-50000), limit(1-100)
- [ ] Handler tests
- [ ] Commit: `handler: implement memory CRUD and nearby query endpoints`

### Task 3.7: Interactions (like/bookmark/report)

**Files:** `internal/repository/interaction_repo.go`, `internal/service/interaction_service.go`, `internal/handler/interaction_handler.go`, tests

- [ ] Like toggle with atomic count update
- [ ] Bookmark CRUD
- [ ] Report: auto-escalate to moderation at 3 distinct reports
- [ ] Endpoints: `POST/DELETE /memories/:id/like`, `/bookmark`, `POST /report`
- [ ] Tests
- [ ] Commit: `feature: implement likes, bookmarks, and reports`

---

## Chunk 4: Content Upload / R2 (Days 15-18)

### Task 4.1: R2 storage client

**Files:** `internal/pkg/storage/r2_client.go`, `r2_client_test.go`

- [ ] Interface: `GeneratePresignedUploadURL`, `GeneratePresignedDownloadURL`, `DeleteObject`, `HeadObject`
- [ ] aws-sdk-go-v2 with R2 endpoint config
- [ ] Key pattern: `memories/<user_id>/<memory_id>/<type>/<uuid>.<ext>`
- [ ] Unit tests with mocked S3 client
- [ ] Commit: `storage: implement R2 pre-signed URL client`

### Task 4.2: Upload service (two-phase)

**Files:** `internal/service/upload_service.go`, `upload_service_test.go`

- [ ] `RequestUpload`: validate ownership + MIME + size limits (photo 20MB, video 100MB, voice 10MB) + dedup check → generate pre-signed URL
- [ ] `ConfirmUpload`: HeadObject verify → create memory_media record
- [ ] Content hash dedup: if same SHA-256 exists, reuse URL
- [ ] Unit tests
- [ ] Commit: `service: implement two-phase upload with deduplication`

### Task 4.3: Upload HTTP handler

**Files:** `internal/handler/upload_handler.go`, `upload_handler_test.go`

- [ ] `POST /api/v1/upload/request` → returns pre-signed URL
- [ ] `POST /api/v1/upload/confirm` → finalizes media record
- [ ] Handler tests
- [ ] Commit: `handler: implement upload endpoints`

---

## Chunk 5: Social Layer (Days 19-24)

### Task 5.1: Circle repository

**Files:** `internal/model/circle.go`, `internal/repository/circle_repo.go`, `circle_repo_test.go`

- [ ] Circle CRUD + membership operations
- [ ] Integration tests
- [ ] Commit: `repo: implement FriendCircle repository`

### Task 5.2: Permission repository

**Files:** `internal/repository/permission_repo.go`, `permission_repo_test.go`

- [ ] `GrantCircle`, `GrantUser`, `GrantToken`, `Revoke`, `CanAccess`, `CanAccessByToken`
- [ ] `CanAccess` SQL covers: owner, public, circle member, direct user grant
- [ ] Tests for all access paths
- [ ] Commit: `repo: implement permission checks`

### Task 5.3: Circle service

**Files:** `internal/service/circle_service.go`, `circle_service_test.go`

- [ ] Limits: max 20 circles/user, max 100 members/circle
- [ ] Unit tests
- [ ] Commit: `service: implement circle management`

### Task 5.4: Permission service

**Files:** `internal/service/permission_service.go`, `permission_service_test.go`

- [ ] Generate shareable auth tokens (32 random bytes, store SHA-256, return base64url)
- [ ] Validate access before returning memory content
- [ ] Unit tests
- [ ] Commit: `service: implement shareable auth tokens`

### Task 5.5: Circle + permission handlers

**Files:** `internal/handler/circle_handler.go`, tests

- [ ] Circle CRUD + member management endpoints
- [ ] Permission grant/revoke + share-token generation
- [ ] Tests
- [ ] Commit: `handler: implement circle and permission endpoints`

### Task 5.6: Discovery engine

**Files:** `internal/service/discovery_service.go`, `discovery_service_test.go`

- [ ] Multi-sort: distance (default), recent, popular
- [ ] Pagination with total count
- [ ] Clustering integration
- [ ] Tests
- [ ] Commit: `service: implement discovery engine`

---

## Chunk 6: Moderation Pipeline (Days 25-28)

### Task 6.1: GLM-4V moderation client

**Files:** `internal/pkg/moderation/glm_client.go`, `glm_client_test.go`

- [ ] ZhipuAI REST API integration (`POST https://open.bigmodel.cn/api/paas/v4/chat/completions`)
- [ ] `ModerateImage(imageURL)`, `ModerateText(text)` → `ModerationResult{Safe, Confidence, Categories}`
- [ ] 30s timeout, 1 retry on 5xx
- [ ] Unit tests with mocked HTTP
- [ ] Commit: `pkg: implement GLM-4V moderation client`

### Task 6.2: Moderation repository

**Files:** `internal/repository/moderation_repo.go`, `moderation_repo_test.go`

- [ ] `Create`, `ListPending`, `ListEscalated`, `UpdateReview`
- [ ] Integration tests
- [ ] Commit: `repo: implement moderation queue`

### Task 6.3: Moderation service + background worker

**Files:** `internal/service/moderation_service.go`, `moderation_service_test.go`

- [ ] `ProcessQueue`: batch fetch → GLM-4V → auto-approve (>0.95 safe) / auto-reject (>0.95 unsafe) / escalate
- [ ] Background worker goroutine with configurable interval
- [ ] `ManualReview` for admin
- [ ] Unit tests
- [ ] Commit: `service: implement AI moderation pipeline`

### Task 6.4: Admin endpoints

**Files:** `internal/handler/moderation_handler.go`, `internal/middleware/admin.go`, migration `000002_add_admin_role.up.sql`

- [ ] Add `is_admin` column to users
- [ ] Admin middleware: check is_admin flag
- [ ] `GET /api/v1/admin/moderation/queue`, `/stats`, `PUT /:id/review`
- [ ] Tests
- [ ] Commit: `handler: implement admin moderation endpoints`

---

## Chunk 7: Docs + Testing + CI (Days 29-33)

### Task 7.1: Swagger/OpenAPI annotations

- [ ] Add swaggo annotations to all handlers
- [ ] Run `swag init`, register `/swagger/*any`
- [ ] Verify: `curl localhost:8080/swagger/index.html`
- [ ] Commit: `docs: add OpenAPI documentation`

### Task 7.2: Structured logging middleware

- [ ] Request/response logging with zerolog (method, path, status, latency, request_id)
- [ ] Commit: `middleware: add structured request logging`

### Task 7.3: CORS middleware

- [ ] Configurable allowed origins, methods, headers
- [ ] Commit: `middleware: add CORS`

### Task 7.4: Graceful shutdown

- [ ] Update main.go: catch SIGINT/SIGTERM, shutdown server, close DB pool, close Redis, stop moderation worker
- [ ] Commit: `infra: implement graceful shutdown`

### Task 7.5: End-to-end integration tests

**Files:** `tests/integration/` — `auth_test.go`, `memory_test.go`, `spatial_test.go`, `upload_test.go`, `permission_test.go`, `moderation_test.go`

- [ ] Full API flow tests against real PostGIS + Redis via testcontainers
- [ ] Key flows: register → create memory → nearby query → like → share token → access
- [ ] Commit: `test: add E2E integration tests`

### Task 7.6: GitHub Actions CI

**Files:** `.github/workflows/ci.yml`

- [ ] Jobs: lint (golangci-lint), test-unit, test-integration (with PostGIS + Redis services), build (binary + Docker image)
- [ ] Commit: `ci: add GitHub Actions pipeline`

### Task 7.7: README

**Files:** `README.md`

- [ ] Quick start: `make docker-up && make migrate-up && make dev`
- [ ] API docs link, env vars table, project structure, testing instructions
- [ ] Commit: `docs: add README`

---

## Verification

After completing all chunks:

1. **Start services**: `make docker-up && make migrate-up && make dev`
2. **Health check**: `curl localhost:8080/health` → `{"status":"ok","db":"connected","redis":"connected"}`
3. **Auth flow**: Send SMS → verify → get JWT → access protected endpoints
4. **Memory flow**: Create memory (with lat/lng) → query nearby → verify it appears
5. **Spatial accuracy**: Create memories at known coordinates, verify `ST_DWithin` radius filtering
6. **Upload flow**: Request presigned URL → upload to R2 → confirm → verify media attached
7. **Permission flow**: Create private memory → share with circle → verify circle member can access, non-member cannot
8. **Moderation flow**: Create public memory → verify it enters moderation queue → process queue → verify auto-approve/reject
9. **Tests**: `make test && make test-integration` — all pass
10. **Swagger**: Visit `localhost:8080/swagger/index.html` — full API documentation
11. **CI**: Push to GitHub → verify CI pipeline passes (lint + test + build)

---

## Critical Files

| File | Why Critical |
|------|-------------|
| `migrations/000001_initial_schema.up.sql` | PostGIS schema is the foundation. Spatial indexes determine query performance |
| `internal/repository/memory_repo.go` | Contains the core `ST_DWithin` + `ST_Distance` spatial query — the product's primary value proposition |
| `internal/service/auth_service.go` | SMS + WeChat login — critical path for Chinese market user onboarding |
| `internal/repository/spatial_cache.go` | Redis GEO cache for hot zones — determines performance under load |
| `internal/service/moderation_service.go` | GLM-4V AI pipeline — content moderation is legally required in China |
