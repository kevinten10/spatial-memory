package config

import (
	"net/url"
	"testing"
)

func testDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:     "db.example.test",
		Port:     5432,
		User:     "postgres.project-ref",
		Password: "password with symbols !@#",
		DBName:   "postgres",
		SSLMode:  "require",
		Schema:   "spatial_memory",
	}
}

func TestDatabaseConfigDSNUsesDedicatedSchema(t *testing.T) {
	cfg := testDatabaseConfig()
	parsed, err := url.Parse(cfg.DSN())
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}

	if got := parsed.Query().Get("search_path"); got != "spatial_memory,public,extensions" {
		t.Fatalf("unexpected search_path: %q", got)
	}
	if got := parsed.Query().Get("default_query_exec_mode"); got != "simple_protocol" {
		t.Fatalf("unexpected query mode: %q", got)
	}
	password, ok := parsed.User.Password()
	if !ok || password != cfg.Password {
		t.Fatal("database password was not URL encoded and decoded correctly")
	}
}

func TestDatabaseConfigMigrationDSNIsolatesHistory(t *testing.T) {
	cfg := testDatabaseConfig()
	parsed, err := url.Parse(cfg.MigrationDSN())
	if err != nil {
		t.Fatalf("parse migration DSN: %v", err)
	}

	query := parsed.Query()
	if got := query.Get("search_path"); got != "public" {
		t.Fatalf("migration metadata must use public search_path, got %q", got)
	}
	if got := query.Get("x-migrations-table"); got != "spatial_memory_schema_migrations" {
		t.Fatalf("unexpected migration table: %q", got)
	}
	if query.Has("default_query_exec_mode") {
		t.Fatal("pgx-only query mode leaked into the lib/pq migration DSN")
	}
}

func TestDatabaseConfigRejectsUnsafeSchema(t *testing.T) {
	cfg := testDatabaseConfig()
	cfg.Schema = "spatial_memory; drop schema public"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unsafe schema name to be rejected")
	}
}

func TestDatabaseConfigRejectsSchemaThatDoesNotMatchMigrations(t *testing.T) {
	cfg := testDatabaseConfig()
	cfg.Schema = "another_safe_schema"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected schema not represented by migrations to be rejected")
	}
}

func TestDatabaseConfigDefaultsSchema(t *testing.T) {
	cfg := testDatabaseConfig()
	cfg.Schema = ""
	if got := cfg.SchemaName(); got != "spatial_memory" {
		t.Fatalf("unexpected default schema: %q", got)
	}
}
