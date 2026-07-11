package bridge

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spatial-memory/spatial-memory/internal/config"
)

func TestConfigureServerlessDatabasePreservesDedicatedRole(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "aws-1-ap-southeast-1.pooler.supabase.com",
			Port:     5432,
			User:     "spatial_memory_app.project-ref",
			MinConns: 4,
			MaxConns: 20,
		},
	}

	configureServerlessDatabase(cfg)

	if cfg.Database.User != "spatial_memory_app.project-ref" {
		t.Fatalf("dedicated database role was rewritten: %q", cfg.Database.User)
	}
	if cfg.Database.Host != "aws-1-ap-southeast-1.pooler.supabase.com" || cfg.Database.Port != 5432 {
		t.Fatalf("configured Supavisor endpoint was rewritten: %s:%d", cfg.Database.Host, cfg.Database.Port)
	}
	if cfg.Database.MinConns != 0 || cfg.Database.MaxConns != 2 {
		t.Fatalf("unexpected serverless pool limits: min=%d max=%d", cfg.Database.MinConns, cfg.Database.MaxConns)
	}
}

func TestDegradedHealthDoesNotExposeInitializationDetails(t *testing.T) {
	setupDegraded()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	GinEngine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	body := recorder.Body.String()
	if strings.Contains(body, "database") || strings.Contains(body, "postgres") || strings.Contains(body, "pooler") {
		t.Fatalf("health response exposed initialization details: %s", body)
	}
	if !strings.Contains(body, "service temporarily unavailable") {
		t.Fatalf("health response did not include the public-safe error: %s", body)
	}
}
