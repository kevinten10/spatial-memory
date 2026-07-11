package main

import (
	"strings"
	"testing"
)

func TestRunRejectsMissingActionBeforeLoadingConfig(t *testing.T) {
	err := run(nil)
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("expected usage error, got %v", err)
	}
}

func TestRunRejectsUnknownAction(t *testing.T) {
	t.Setenv("SPATIAL_DATABASE_SCHEMA", "spatial_memory")
	err := run([]string{"sideways"})
	if err == nil || !strings.Contains(err.Error(), "unknown migration action") {
		t.Fatalf("expected unknown action error, got %v", err)
	}
}
