package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/spatial-memory/spatial-memory/internal/handler"
	"github.com/spatial-memory/spatial-memory/internal/middleware"
	"github.com/spatial-memory/spatial-memory/internal/service"

	_ "github.com/spatial-memory/spatial-memory/docs/swagger"
)

type Handlers struct {
	Health      *handler.HealthHandler
	Auth        *handler.AuthHandler
	User        *handler.UserHandler
	Memory      *handler.MemoryHandler
	Interaction *handler.InteractionHandler
	Upload      *handler.UploadHandler
	Circle      *handler.CircleHandler
	Permission  *handler.PermissionHandler
	Moderation  *handler.ModerationHandler
}

type Config struct {
	TokenService service.TokenService
	RedisClient  *redis.Client
	Handlers     Handlers
}

func Setup(r *gin.Engine, cfg Config) {
	// Health check (no auth)
	r.GET("/health", cfg.Handlers.Health.Health)

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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

		// Interactions
		protected.POST("/memories/:id/like", cfg.Handlers.Interaction.Like)
		protected.DELETE("/memories/:id/like", cfg.Handlers.Interaction.Unlike)
		protected.POST("/memories/:id/bookmark", cfg.Handlers.Interaction.Bookmark)
		protected.DELETE("/memories/:id/bookmark", cfg.Handlers.Interaction.Unbookmark)
		protected.POST("/memories/:id/report", cfg.Handlers.Interaction.Report)

		// Uploads
		protected.POST("/uploads/request", cfg.Handlers.Upload.RequestUpload)
		protected.POST("/uploads/confirm", cfg.Handlers.Upload.ConfirmUpload)

		// Circles
		protected.POST("/circles", cfg.Handlers.Circle.Create)
		protected.GET("/circles/mine", cfg.Handlers.Circle.ListMine)
		protected.GET("/circles/joined", cfg.Handlers.Circle.ListJoined)
		protected.GET("/circles/:id", cfg.Handlers.Circle.Get)
		protected.PUT("/circles/:id", cfg.Handlers.Circle.Update)
		protected.DELETE("/circles/:id", cfg.Handlers.Circle.Delete)
		protected.POST("/circles/:id/members", cfg.Handlers.Circle.AddMember)
		protected.GET("/circles/:id/members", cfg.Handlers.Circle.ListMembers)
		protected.DELETE("/circles/:id/members/:user_id", cfg.Handlers.Circle.RemoveMember)

		// Permissions
		protected.POST("/memories/:id/grant/circle", cfg.Handlers.Permission.GrantCircleAccess)
		protected.POST("/memories/:id/grant/user", cfg.Handlers.Permission.GrantUserAccess)
		protected.POST("/memories/:id/revoke", cfg.Handlers.Permission.RevokeAccess)
		protected.POST("/memories/:id/share", cfg.Handlers.Permission.GenerateShareToken)
	}

	// Admin routes (protected + admin only)
	admin := api.Group("/admin")
	admin.Use(middleware.Auth(cfg.TokenService))
	admin.Use(middleware.AdminOnly())
	admin.Use(middleware.RateLimit(cfg.RedisClient, "admin", 100, 1*time.Minute))
	{
		// Moderation
		admin.GET("/moderation/queue", cfg.Handlers.Moderation.ListQueue)
		admin.GET("/moderation/stats", cfg.Handlers.Moderation.GetStats)
		admin.GET("/moderation/:id", cfg.Handlers.Moderation.GetItem)
		admin.PUT("/moderation/:id/review", cfg.Handlers.Moderation.ManualReview)
	}
}
