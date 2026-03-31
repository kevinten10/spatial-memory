package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/spatial-memory/spatial-memory/internal/pkg/response"
)

// RateLimit returns middleware that enforces a per-key rate limit using Redis.
// keyPrefix determines the rate limit bucket (e.g., "auth" or "api").
// limit is max requests per window, window is the time window duration.
func RateLimit(redisClient *redis.Client, keyPrefix string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build key: ip for unauthenticated, user_id for authenticated
		var identifier string
		if userID, exists := c.Get(ContextKeyUserID); exists {
			identifier = fmt.Sprintf("user:%d", userID.(int64))
		} else {
			identifier = "ip:" + c.ClientIP()
		}

		key := fmt.Sprintf("rl:%s:%s", keyPrefix, identifier)

		ctx := c.Request.Context()

		// INCR + EXPIRE in a pipeline
		pipe := redisClient.Pipeline()
		incrCmd := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, window)
		_, err := pipe.Exec(ctx)
		if err != nil {
			// If Redis is down, allow the request (fail-open)
			c.Next()
			return
		}

		count := incrCmd.Val()
		if count > int64(limit) {
			remaining := redisClient.TTL(ctx, key).Val()
			c.Header("Retry-After", fmt.Sprintf("%d", int(remaining.Seconds())))
			response.Error(c, http.StatusTooManyRequests, 42900, "rate limit exceeded")
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", int64(limit)-count))
		c.Next()
	}
}
