package bridge

import (
	"context"
	"net/http"
	"os"
	"strings"
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

var GinEngine *gin.Engine
var initError string

func InitApp() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().Timestamp().Logger()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
		return
	}

	// Serverless: override pool settings for Supavisor transaction mode
	cfg.Database.MinConns = 0
	cfg.Database.MaxConns = 2

	// Auto-detect Supabase pooler: use port 6543 if host contains "pooler" and port is 0/5432
	if strings.Contains(cfg.Database.Host, "pooler") && cfg.Database.Port != 6543 {
		cfg.Database.Port = 6543
	}

	ctx := context.Background()

	dbPool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		initError = err.Error()
		log.Error().Err(err).Msg("failed to connect to PostgreSQL, starting in degraded mode")
		setupDegraded()
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
	GinEngine = gin.New()
	GinEngine.Use(gin.Recovery())

	router.Setup(GinEngine, router.Config{
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

func setupDegraded() {
	gin.SetMode(gin.ReleaseMode)
	GinEngine = gin.New()
	GinEngine.Use(gin.Recovery())
	GinEngine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "error": initError})
	})
	GinEngine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "service temporarily unavailable"})
	})
}
