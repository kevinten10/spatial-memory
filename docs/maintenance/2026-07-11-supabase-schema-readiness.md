# Spatial Memory Supabase Schema Readiness

## Target

- Supabase project: `kevinten10` (`lvazmokpqrywaysgxspg`)
- Application schema: `spatial_memory`
- Production host: `https://spatial-memory-zeta.vercel.app`
- Data access: backend PostgreSQL connection only; do not add `spatial_memory` to exposed Data API schemas.

## Implemented

- Added `SPATIAL_DATABASE_SCHEMA`, defaulting to `spatial_memory`.
- Applied a fixed connection search path: `spatial_memory,public,extensions`.
- Added schema-name validation before constructing a connection.
- Added a migration-specific DSN with the unique metadata table `public.spatial_memory_schema_migrations`.
- Updated the initial migration to create application objects in `spatial_memory`.
- Preserved the shared PostGIS extension during rollback.
- Added `go run ./cmd/migrate up|down [steps]` as the supported migration command.
- Added unit tests for URL encoding, schema isolation, unsafe schema rejection, migration history isolation, and rollback safety.

## Validation

- `go test ./...`: passed.
- `gitleaks dir . --no-banner --redact`: passed.
- Fresh `postgis/postgis:16-3.4` container migration: passed.
- Repeated `up`: stayed at version 2 without reapplying migrations.
- Application tables: 10 in `spatial_memory`, 0 matching app tables in `public`.
- Migration metadata: `public.spatial_memory_schema_migrations` exists.
- Full `down 2`: removed `spatial_memory` while preserving PostGIS.

## Production Gate

1. In Vercel, replace the stale database host/user/password with the canonical project's connection values and add `SPATIAL_DATABASE_SCHEMA=spatial_memory`.
2. Confirm PostGIS is enabled in the Supabase project.
3. From a trusted environment with the same private variables, run `go run ./cmd/migrate up`.
4. Redeploy the exact application commit and verify `/health` becomes 200 without returning connection details.
5. Keep the schema private; if it is ever exposed through the Data API, add explicit grants and RLS first.

No database password, API key, connection string, or environment value is recorded in this document.
