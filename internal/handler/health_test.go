package handler

import (
	"net/http"
	"testing"
)

func TestHealthHTTPStatus(t *testing.T) {
	tests := []struct {
		name          string
		dbStatus      string
		redisStatus   string
		redisRequired bool
		want          int
	}{
		{
			name:          "all required dependencies connected",
			dbStatus:      "connected",
			redisStatus:   "connected",
			redisRequired: true,
			want:          http.StatusOK,
		},
		{
			name:          "optional redis unavailable",
			dbStatus:      "connected",
			redisStatus:   "not configured",
			redisRequired: false,
			want:          http.StatusOK,
		},
		{
			name:          "required redis unavailable",
			dbStatus:      "connected",
			redisStatus:   "disconnected",
			redisRequired: true,
			want:          http.StatusServiceUnavailable,
		},
		{
			name:          "database unavailable",
			dbStatus:      "disconnected",
			redisStatus:   "connected",
			redisRequired: false,
			want:          http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := healthHTTPStatus(tt.dbStatus, tt.redisStatus, tt.redisRequired); got != tt.want {
				t.Fatalf("healthHTTPStatus() = %d, want %d", got, tt.want)
			}
		})
	}
}
