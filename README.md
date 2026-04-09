# Spatial Memory Network API

Backend API for a Spatial Memory Network - an AR app that lets users pin multimedia memories (photos, videos, voice, text) to real-world geographic locations. Others must physically visit the location to discover and view those memories.

> "空间版小红书 / Pinterest for the physical world"

## Features

- **Spatial Memory Creation**: Pin multimedia memories to real-world GPS coordinates
- **Proximity-Based Discovery**: Find memories near your current location using PostGIS spatial queries
- **Progressive Visibility**: Private → Circle (friends) → Public sharing options
- **Two-Phase Upload**: Direct-to-R2 uploads via pre-signed URLs (never proxy through API)
- **AI Moderation**: GLM-4V powered content moderation for public UGC
- **Social Circles**: Create friend circles for selective memory sharing
- **Real-time Cache**: Redis GEO commands for hot-zone spatial caching
- **Multi-Auth**: SMS and WeChat OAuth login support

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
- **API Docs**: Swagger/OpenAPI (swaggo)

## Quick Start

### Prerequisites

- Go 1.22+
- Docker and Docker Compose
- Make

### Setup

1. **Clone and start dependencies:**
   ```bash
   git clone https://github.com/spatial-memory/spatial-memory.git
   cd spatial-memory
   make docker-up
   ```

2. **Run database migrations:**
   ```bash
   make migrate-up
   ```

3. **Start the development server:**
   ```bash
   make dev
   ```

4. **Verify the API is running:**
   ```bash
   curl http://localhost:8080/health
   ```

### Environment Variables

Create a `.env` file based on `.env.example`:

```bash
# Server
SPATIAL_SERVER_PORT=8080
SPATIAL_SERVER_MODE=debug

# Database
SPATIAL_DATABASE_HOST=localhost
SPATIAL_DATABASE_PORT=5432
SPATIAL_DATABASE_USER=spatial
SPATIAL_DATABASE_PASSWORD=spatial
SPATIAL_DATABASE_NAME=spatial_memory
SPATIAL_DATABASE_SSLMODE=disable

# Redis
SPATIAL_REDIS_HOST=localhost
SPATIAL_REDIS_PORT=6379

# JWT
SPATIAL_JWT_SECRET=your-secret-key-here
SPATIAL_JWT_ACCESS_TTL=2h
SPATIAL_JWT_REFRESH_TTL=720h

# R2 Storage
SPATIAL_R2_ACCOUNT_ID=your-account-id
SPATIAL_R2_ACCESS_KEY_ID=your-access-key
SPATIAL_R2_ACCESS_KEY_SECRET=your-secret-key
SPATIAL_R2_BUCKET_NAME=spatial-memory
SPATIAL_R2_PUBLIC_URL=https://your-bucket.r2.cloudflarestorage.com

# SMS (Twilio or similar)
SPATIAL_SMS_PROVIDER=twilio
SPATIAL_SMS_ACCOUNT_SID=your-account-sid
SPATIAL_SMS_AUTH_TOKEN=your-auth-token
SPATIAL_SMS_FROM_NUMBER=your-phone-number

# WeChat OAuth
SPATIAL_WECHAT_APP_ID=your-app-id
SPATIAL_WECHAT_APP_SECRET=your-app-secret

# GLM-4V Moderation
SPATIAL_GLM_API_KEY=your-glm-api-key
```

## API Documentation

Once the server is running, access the Swagger UI at:

```
http://localhost:8080/swagger/index.html
```

### Key Endpoints

#### Authentication
- `POST /api/v1/auth/sms/send` - Send SMS verification code
- `POST /api/v1/auth/sms/verify` - Verify SMS code and login
- `POST /api/v1/auth/wechat` - WeChat OAuth login
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - Logout and revoke token

#### Users
- `GET /api/v1/users/me` - Get current user profile
- `PUT /api/v1/users/me` - Update user profile
- `GET /api/v1/users/:id` - Get public user profile

#### Memories
- `POST /api/v1/memories` - Create a new memory
- `GET /api/v1/memories/mine` - List user's memories
- `GET /api/v1/memories/nearby` - Find memories near location
- `GET /api/v1/memories/:id` - Get memory by ID
- `PUT /api/v1/memories/:id` - Update memory
- `DELETE /api/v1/memories/:id` - Delete memory

#### Interactions
- `POST /api/v1/memories/:id/like` - Like a memory
- `DELETE /api/v1/memories/:id/like` - Unlike a memory
- `POST /api/v1/memories/:id/bookmark` - Bookmark a memory
- `DELETE /api/v1/memories/:id/bookmark` - Remove bookmark
- `POST /api/v1/memories/:id/report` - Report a memory

#### Uploads
- `POST /api/v1/uploads/request` - Request pre-signed upload URL
- `POST /api/v1/uploads/confirm` - Confirm upload completion

#### Circles
- `POST /api/v1/circles` - Create a friend circle
- `GET /api/v1/circles/mine` - List my circles
- `GET /api/v1/circles/joined` - List joined circles
- `GET /api/v1/circles/:id` - Get circle details
- `PUT /api/v1/circles/:id` - Update circle
- `DELETE /api/v1/circles/:id` - Delete circle
- `POST /api/v1/circles/:id/members` - Add member
- `DELETE /api/v1/circles/:id/members/:user_id` - Remove member

#### Permissions
- `POST /api/v1/memories/:id/grant/circle` - Grant circle access
- `POST /api/v1/memories/:id/grant/user` - Grant user access
- `POST /api/v1/memories/:id/revoke` - Revoke access
- `POST /api/v1/memories/:id/share` - Generate share token

#### Admin (Moderation)
- `GET /api/v1/admin/moderation/queue` - List moderation queue
- `GET /api/v1/admin/moderation/stats` - Get moderation stats
- `GET /api/v1/admin/moderation/:id` - Get moderation item
- `PUT /api/v1/admin/moderation/:id/review` - Review item

## Project Structure

```
spatial-memory/
├── cmd/server/main.go           # Entry point, DI wiring
├── internal/
│   ├── config/                  # Viper configuration
│   ├── database/                # DB + Redis + migrations
│   ├── handler/                 # HTTP handlers (Gin)
│   ├── middleware/              # Auth, rate-limit, logging
│   ├── model/                   # Domain models
│   ├── pkg/                     # Shared utilities
│   │   ├── errors/              # Domain error types
│   │   ├── response/            # API response helpers
│   │   ├── sms/                 # SMS provider client
│   │   ├── storage/             # R2/S3 client
│   │   ├── wechat/              # WeChat OAuth client
│   │   └── moderation/          # GLM-4V moderation client
│   ├── repository/              # Database access (pgx)
│   ├── router/                  # Route definitions
│   └── service/                 # Business logic
├── migrations/                  # SQL migration files
├── tests/integration/           # E2E API tests
├── docs/swagger/                # Generated Swagger docs
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── go.mod
```

## Architecture

Clean layered architecture: **handler → service → repository**

- **No ORM**: PostGIS spatial queries use hand-written SQL via pgx
- **Two-phase upload**: Clients upload directly to R2 via pre-signed URLs
- **Redis GEO cache**: Hot-zone spatial queries cached with GEOADD/GEOSEARCH
- **Background moderation**: Public memories queue for GLM-4V AI review

## Development

### Available Make Commands

```bash
make dev              # Start with hot-reload (air)
make build            # Build binary
make test             # Run unit tests
make test-integration # Run integration tests
make lint             # Run golangci-lint
make swagger          # Generate Swagger docs
make migrate-up       # Run migrations up
make migrate-down     # Run migrations down
make migrate-create   # Create new migration
make docker-up        # Start Docker services
make docker-down      # Stop Docker services
```

### Running Tests

```bash
# Unit tests
make test

# Integration tests (requires Docker)
make test-integration

# With coverage
make test-coverage
```

### Creating Migrations

```bash
make migrate-create
# Enter migration name when prompted
```

### Generating Swagger Docs

```bash
make swagger
```

## Deployment

### Docker

Build and run with Docker:

```bash
# Build image
docker build -t spatial-memory:latest .

# Run container
docker run -p 8080:8080 --env-file .env spatial-memory:latest
```

### Production Checklist

- [ ] Set strong JWT secret
- [ ] Configure production database
- [ ] Set up R2/cloud storage credentials
- [ ] Configure SMS provider
- [ ] Set up WeChat OAuth credentials
- [ ] Configure GLM-4V API key
- [ ] Enable HTTPS/TLS
- [ ] Set up monitoring and logging
- [ ] Configure rate limiting
- [ ] Run database migrations

## CI/CD

This project uses GitHub Actions for CI/CD. The workflow includes:

1. **Lint**: golangci-lint for code quality
2. **Test**: Unit and integration tests with PostgreSQL and Redis services
3. **Build**: Binary and Docker image builds
4. **Docker**: Multi-stage Docker build verification

See `.github/workflows/ci.yml` for details.

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please follow conventional commits specification for commit messages.
