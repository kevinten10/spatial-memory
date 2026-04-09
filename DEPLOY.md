# Deployment Guide

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
  -e GLM_API_KEY="your-glm-api-key" \
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
      - GLM_API_KEY=${GLM_API_KEY}
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
| `GLM_API_KEY` | ZhipuAI GLM-4V API key |

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
   go run cmd/migrate/main.go up
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
