# Spatial Memory Network

Backend API for a Spatial Memory Network — an AR app that lets users pin multimedia memories (photos, videos, voice, text) to real-world geographic locations. Others must physically visit the location to discover and view those memories.

"空间版小红书 / Pinterest for the physical world"

## Features

- **Spatial Memories**: Pin memories to geographic coordinates with PostGIS
- **Proximity Discovery**: Find nearby memories using PostGIS spatial queries and Redis GEO cache
- **Progressive Visibility**: Private → Circle (friend groups) → Public
- **Multimedia Upload**: Direct-to-R2 two-phase upload with pre-signed URLs
- **AI Moderation**: GLM-4V AI-powered content moderation with manual review
- **Social Circles**: Create friend circles for sharing memories
- **Share Tokens**: Generate secure, shareable tokens for memory access

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

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Make

### Development

```bash
# Start PostgreSQL + Redis
make docker-up

# Run migrations
make migrate-up

# Start API with hot-reload
make dev
```

The API will be available at `http://localhost:8080`

### Build & Run

```bash
# Build binary
make build

# Run migrations and start server
./spatial-memory
```

## API Documentation

Swagger UI is available at `http://localhost:8080/swagger/index.html`

### Key Endpoints

#### Auth
- `POST /api/v1/auth/sms/send` - Send SMS verification code
- `POST /api/v1/auth/sms/verify` - Verify SMS code and login
- `POST /api/v1/auth/wechat` - WeChat OAuth login
- `POST /api/v1/auth/refresh` - Refresh access token

#### Memories
- `POST /api/v1/memories` - Create a memory
- `GET /api/v1/memories/mine` - List my memories
- `GET /api/v1/memories/nearby` - Find nearby memories (spatial query)
- `GET /api/v1/memories/:id` - Get memory details
- `PUT /api/v1/memories/:id` - Update memory
- `DELETE /api/v1/memories/:id` - Delete memory

#### Circles
- `POST /api/v1/circles` - Create a friend circle
- `GET /api/v1/circles/mine` - List my circles
- `GET /api/v1/circles/:id` - Get circle details
- `POST /api/v1/circles/:id/members` - Add member to circle
- `DELETE /api/v1/circles/:id/members/:user_id` - Remove member

#### Permissions
- `POST /api/v1/memories/:id/grant/circle` - Grant circle access
- `POST /api/v1/memories/:id/grant/user` - Grant user access
- `POST /api/v1/memories/:id/revoke` - Revoke access
- `POST /api/v1/memories/:id/share` - Generate share token

#### Uploads
- `POST /api/v1/uploads/request` - Get pre-signed upload URL
- `POST /api/v1/uploads/confirm` - Confirm upload completion

#### Admin
- `GET /api/v1/admin/moderation/queue` - Get moderation queue
- `PUT /api/v1/admin/moderation/:id/review` - Manual review

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
│   └── pkg/                    # Shared utilities
├── migrations/                  # SQL migration files
├── docs/                        # Documentation
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── go.mod
```

## Architecture

Clean layered architecture: **handler → service → repository**

- **No ORM** — PostGIS spatial queries need hand-written SQL via pgx
- **Two-phase upload** — clients upload directly to R2 via pre-signed URLs
- **Redis GEO cache** — hot-zone spatial queries cached with GEOADD/GEOSEARCH
- **Background moderation** — public memories queue for GLM-4V AI review

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:postgres@localhost:5432/spatial_memory?sslmode=disable` |
| `REDIS_URL` | Redis connection string | `redis://localhost:6379/0` |
| `JWT_SECRET` | JWT signing secret | required |
| `R2_ACCOUNT_ID` | Cloudflare R2 account ID | required |
| `R2_ACCESS_KEY_ID` | R2 access key | required |
| `R2_SECRET_ACCESS_KEY` | R2 secret key | required |
| `R2_BUCKET` | R2 bucket name | required |
| `GLM_API_KEY` | ZhipuAI GLM-4V API key | required |
| `SMS_ACCESS_KEY` | SMS service access key | optional |
| `WECHAT_APP_ID` | WeChat app ID | optional |
| `WECHAT_SECRET` | WeChat app secret | optional |

## Deployment

### Docker

```bash
# Build image
docker build -t spatial-memory .

# Run container
docker run -p 8080:8080 \
  -e DATABASE_URL=$DATABASE_URL \
  -e REDIS_URL=$REDIS_URL \
  -e JWT_SECRET=$JWT_SECRET \
  spatial-memory
```

### Production Checklist

- [ ] Set strong JWT_SECRET
- [ ] Configure R2 credentials
- [ ] Configure GLM-4V API key
- [ ] Enable SSL/TLS
- [ ] Set up monitoring (metrics, logs)
- [ ] Configure rate limiting

## License

MIT License - see LICENSE for details
