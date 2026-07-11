package database

import (
	"strings"
	"testing"

	"github.com/spatial-memory/spatial-memory/migrations"
)

func TestInitialMigrationUsesDedicatedSchema(t *testing.T) {
	up, err := migrations.FS.ReadFile("000001_initial_schema.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}

	sql := string(up)
	for _, required := range []string{
		"CREATE SCHEMA IF NOT EXISTS spatial_memory",
		"SET search_path TO spatial_memory, public, extensions",
		"CREATE TABLE users",
		"CREATE TABLE memories",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("up migration missing %q", required)
		}
	}
}

func TestEveryMigrationSelectsDedicatedSchema(t *testing.T) {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := migrations.FS.ReadFile(entry.Name())
		if err != nil {
			t.Fatalf("read migration %s: %v", entry.Name(), err)
		}
		if !strings.Contains(string(content), "SET search_path TO spatial_memory, public, extensions") {
			t.Errorf("migration %s does not select the dedicated schema", entry.Name())
		}
	}
}

func TestRollbackPreservesSharedPostGIS(t *testing.T) {
	down, err := migrations.FS.ReadFile("000001_initial_schema.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	sql := string(down)
	searchPathIndex := strings.Index(sql, "SET search_path TO spatial_memory, public, extensions")
	firstDropIndex := strings.Index(sql, "DROP TABLE")
	if searchPathIndex < 0 || firstDropIndex < 0 || searchPathIndex > firstDropIndex {
		t.Fatal("rollback must select the application schema before dropping objects")
	}
	if strings.Contains(strings.ToUpper(sql), "DROP EXTENSION") {
		t.Fatal("app rollback must not drop shared PostGIS")
	}
	if !strings.Contains(sql, "DROP SCHEMA IF EXISTS spatial_memory") {
		t.Fatal("rollback must remove the dedicated application schema")
	}
}
