package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	dbStatus := "connected"
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			dbStatus = "disconnected"
		}
	} else {
		dbStatus = "not configured"
	}

	redisStatus := "connected"
	if h.redis != nil {
		if err := h.redis.Ping(ctx).Err(); err != nil {
			redisStatus = "disconnected"
		}
	} else {
		redisStatus = "not configured"
	}

	status := http.StatusOK
	if dbStatus != "connected" || redisStatus != "connected" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status": map[bool]string{true: "ok", false: "degraded"}[status == http.StatusOK],
		"db":     dbStatus,
		"redis":  redisStatus,
	})
}
