package handler

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/spatial-memory/spatial-memory/internal/config"
	"github.com/spatial-memory/spatial-memory/internal/database"
	"github.com/spatial-memory/spatial-memory/internal/handler"
	"github.com/spatial-memory/spatial-memory/internal/repository"
	"github.com/spatial-memory/spatial-memory/internal/router"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

var ginEngine *gin.Engine

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().Timestamp().Logger()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
		return
	}

	ctx := context.Background()

	dbPool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to PostgreSQL")
		// Set up a fallback router that returns 503
		gin.SetMode(gin.ReleaseMode)
		ginEngine = gin.New()
		ginEngine.Use(gin.Recovery())
		ginEngine.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "error": "database unavailable"})
		})
		ginEngine.NoRoute(func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "service temporarily unavailable"})
		})
		return
	}

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:0"})

	userRepo := repository.NewUserRepository(dbPool)
	authRepo := repository.NewAuthRepository(dbPool)
	memoryRepo := repository.NewMemoryRepository(dbPool)
	permRepo := repository.NewPermissionRepository(dbPool)
	circleRepo := repository.NewCircleRepository(dbPool)
	spatialCache := repository.NewNoOpSpatialCache()
	interactionRepo := repository.NewInteractionRepository(dbPool)
	moderationRepo := repository.NewModerationRepository(dbPool)
	mediaRepo := repository.NewMediaRepository(dbPool)

	tokenService := service.NewTokenService(cfg.JWT, authRepo, userRepo)
	authService := service.NewAuthService(userRepo, authRepo, tokenService, nil, nil, redisClient)
	memoryService := service.NewMemoryService(memoryRepo, spatialCache, permRepo)
	interactionService := service.NewInteractionService(interactionRepo, memoryRepo, moderationRepo)
	uploadService := service.NewUploadService(nil, memoryRepo, mediaRepo, cfg.R2.PublicURL)
	circleService := service.NewCircleService(circleRepo, userRepo)
	permissionService := service.NewPermissionService(permRepo, memoryRepo, circleRepo)
	moderationService := service.NewModerationService(moderationRepo, memoryRepo, nil)

	healthHandler := handler.NewHealthHandler(dbPool, redisClient)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userRepo)
	memoryHandler := handler.NewMemoryHandler(memoryService)
	interactionHandler := handler.NewInteractionHandler(interactionService)
	uploadHandler := handler.NewUploadHandler(uploadService)
	circleHandler := handler.NewCircleHandler(circleService)
	permissionHandler := handler.NewPermissionHandler(permissionService)
	moderationHandler := handler.NewModerationHandler(moderationService)

	gin.SetMode(gin.ReleaseMode)
	ginEngine = gin.New()
	ginEngine.Use(gin.Recovery())

	router.Setup(ginEngine, router.Config{
		TokenService: tokenService,
		RedisClient:  redisClient,
		Handlers: router.Handlers{
			Health:      healthHandler,
			Auth:        authHandler,
			User:        userHandler,
			Memory:      memoryHandler,
			Interaction: interactionHandler,
			Upload:      uploadHandler,
			Circle:      circleHandler,
			Permission:  permissionHandler,
			Moderation:  moderationHandler,
		},
	})

	log.Info().Msg("spatial memory API initialized")
}

// Handler is the Vercel serverless function entry point
func Handler(w http.ResponseWriter, r *http.Request) {
	ginEngine.ServeHTTP(w, r)
}
