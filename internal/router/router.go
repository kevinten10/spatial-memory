package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/spatial-memory/spatial-memory/internal/handler"
	"github.com/spatial-memory/spatial-memory/internal/middleware"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

type Handlers struct {
	Health *handler.HealthHandler
	Auth   *handler.AuthHandler
	User   *handler.UserHandler
	Memory *handler.MemoryHandler
}

type Config struct {
	TokenService service.TokenService
	RedisClient  *redis.Client
	Handlers     Handlers
}

func Setup(r *gin.Engine, cfg Config) {
	// Health check (no auth)
	r.GET("/health", cfg.Handlers.Health.Health)

	api := r.Group("/api/v1")

	// Auth routes (public, rate-limited)
	auth := api.Group("/auth")
	auth.Use(middleware.RateLimit(cfg.RedisClient, "auth", 10, 1*time.Minute))
	{
		auth.POST("/sms/send", cfg.Handlers.Auth.SendSMSCode)
		auth.POST("/sms/verify", cfg.Handlers.Auth.VerifySMSCode)
		auth.POST("/wechat", cfg.Handlers.Auth.WeChatLogin)
		auth.POST("/refresh", cfg.Handlers.Auth.RefreshTokens)
		auth.POST("/logout", cfg.Handlers.Auth.Logout)
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.Auth(cfg.TokenService))
	protected.Use(middleware.RateLimit(cfg.RedisClient, "api", 100, 1*time.Minute))
	{
		// User
		protected.GET("/users/me", cfg.Handlers.User.GetMe)
		protected.PUT("/users/me", cfg.Handlers.User.UpdateMe)
		protected.GET("/users/:id", cfg.Handlers.User.GetUser)

		// Memories
		protected.POST("/memories", cfg.Handlers.Memory.Create)
		protected.GET("/memories/mine", cfg.Handlers.Memory.ListMine)
		protected.GET("/memories/nearby", cfg.Handlers.Memory.Nearby)
		protected.GET("/memories/:id", cfg.Handlers.Memory.Get)
		protected.PUT("/memories/:id", cfg.Handlers.Memory.Update)
		protected.DELETE("/memories/:id", cfg.Handlers.Memory.Delete)
	}
}
