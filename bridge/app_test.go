package bridge

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
