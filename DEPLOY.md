# Deployment Guide

## Canonical Supabase + Vercel Deployment

Kevin's hosted demo uses the shared Supabase project `lvazmokpqrywaysgxspg` and the dedicated `spatial_memory` schema. Do not create another Supabase project and do not expose this backend-only schema through the Supabase Data API.

Configure these server-side Vercel variables without committing their values:

- `SPATIAL_DATABASE_HOST`
- `SPATIAL_DATABASE_PORT`
- `SPATIAL_DATABASE_USER`
- `SPATIAL_DATABASE_PASSWORD`
- `SPATIAL_DATABASE_DBNAME`
- `SPATIAL_DATABASE_SSLMODE`
- `SPATIAL_DATABASE_SCHEMA=spatial_memory`

Use a dedicated runtime login such as
`spatial_memory_app.<project-ref>` for `SPATIAL_DATABASE_USER`. Copy the current
Supavisor host and port from the project's **Connect** panel (or the linked
Supabase CLI metadata); pooler hosts can change when a project is restored or
moved. Do not put the shared `postgres` administrator credential in Vercel.

Provision the runtime role from a trusted Supabase SQL session. Replace the
password placeholder before execution and store the resulting value only in
the deployment platform:

```sql
CREATE ROLE spatial_memory_app WITH LOGIN PASSWORD '<strong-random-password>';
GRANT CONNECT ON DATABASE postgres TO spatial_memory_app;
GRANT USAGE ON SCHEMA public, extensions, spatial_memory TO spatial_memory_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA spatial_memory TO spatial_memory_app;
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA spatial_memory TO spatial_memory_app;
GRANT SELECT, INSERT, UPDATE, DELETE
  ON TABLE public.spatial_memory_schema_migrations TO spatial_memory_app;
```

The hosted deployment additionally makes this role the owner of the dedicated
`spatial_memory` schema and its objects, so future application migrations do not
need the shared database administrator password. Do not grant ownership of
`public`, `extensions`, or objects belonging to other applications.

Run the isolated migration from a trusted shell after setting those variables:

```bash
go run ./cmd/migrate up
```

The migration stores its history in `public.spatial_memory_schema_migrations`, creates application tables only in `spatial_memory`, and never drops shared PostGIS during rollback.

## Push to GitHub

```bash
# Create repository on GitHub (or use gh CLI)
gh repo create spatial-memory/spatial-memory --public --source=. --push

# Or manually:
git remote add origin https://github.com/yourusername/spatial-memory.git
git push -u origin main
```

## Docker Deployment

### Build and Run Locally

```bash
# Build Docker image
docker build -t spatial-memory:latest .

# Run with environment variables
docker run -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/spatial_memory?sslmode=require" \
  -e REDIS_URL="redis://host:6379/0" \
  -e JWT_SECRET="your-secret-key" \
  -e R2_ACCOUNT_ID="your-account-id" \
  -e R2_ACCESS_KEY_ID="your-access-key" \
  -e R2_SECRET_ACCESS_KEY="your-secret-key" \
  -e R2_BUCKET="your-bucket" \
  -e R2_PUBLIC_URL="https://your-bucket.r2.dev" \
  -e ARK_API_KEY="your-ark-api-key" \
  -e ARK_BASE_URL="https://ark.cn-beijing.volces.com/api/coding/v3" \
  -e ARK_CHAT_MODEL="doubao-seed-2-0-code-preview-260215" \
  -e ARK_VISION_MODEL="doubao-seed-2-0-code-preview-260215" \
  spatial-memory:latest
```

### Docker Compose (Production)

```yaml
version: '3.8'
services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
      - JWT_SECRET=${JWT_SECRET}
      - R2_ACCOUNT_ID=${R2_ACCOUNT_ID}
      - R2_ACCESS_KEY_ID=${R2_ACCESS_KEY_ID}
      - R2_SECRET_ACCESS_KEY=${R2_SECRET_ACCESS_KEY}
      - R2_BUCKET=${R2_BUCKET}
      - R2_PUBLIC_URL=${R2_PUBLIC_URL}
      - ARK_API_KEY=${ARK_API_KEY}
      - ARK_BASE_URL=${ARK_BASE_URL}
      - ARK_CHAT_MODEL=${ARK_CHAT_MODEL}
      - ARK_VISION_MODEL=${ARK_VISION_MODEL}
    restart: unless-stopped
```

## Cloud Deployment

### Railway

```bash
# Install Railway CLI and login
railway login

# Initialize project
railway init

# Deploy
railway up
```

### Fly.io

```bash
# Install flyctl and login
flyctl auth login

# Launch app
flyctl launch

# Set secrets
flyctl secrets set DATABASE_URL="..." JWT_SECRET="..." ...

# Deploy
flyctl deploy
```

## Environment Variables Required

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection string |
| `JWT_SECRET` | JWT signing secret (generate strong random) |
| `R2_ACCOUNT_ID` | Cloudflare R2 account ID |
| `R2_ACCESS_KEY_ID` | R2 access key |
| `R2_SECRET_ACCESS_KEY` | R2 secret key |
| `R2_BUCKET` | R2 bucket name |
| `R2_PUBLIC_URL` | Public URL for R2 bucket |
| `ARK_API_KEY` | Volcengine Ark API key |
| `ARK_BASE_URL` | Ark OpenAI-compatible base URL |
| `ARK_CHAT_MODEL` | Ark text moderation model |
| `ARK_VISION_MODEL` | Ark image moderation model |

### Optional
| Variable | Description |
|----------|-------------|
| `SMS_ACCESS_KEY` | SMS service access key |
| `WECHAT_APP_ID` | WeChat app ID |
| `WECHAT_SECRET` | WeChat app secret |

## Database Setup

1. Create PostgreSQL database with PostGIS extension
2. Run migrations:
   ```bash
   make migrate-up
   # or
   go run ./cmd/migrate up
   ```

## Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "database": "connected",
  "redis": "connected"
}
```
