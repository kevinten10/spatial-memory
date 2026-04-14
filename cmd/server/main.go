// @title Spatial Memory Network API
// @version 1.0
// @description Backend API for Spatial Memory Network - an AR app that lets users pin multimedia memories to real-world geographic locations.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/spatial-memory/spatial-memory/issues
// @contact.email support@spatial-memory.app

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/spatial-memory/spatial-memory/internal/config"
	"github.com/spatial-memory/spatial-memory/internal/database"
	"github.com/spatial-memory/spatial-memory/internal/handler"
	"github.com/spatial-memory/spatial-memory/internal/pkg/moderation"
	"github.com/spatial-memory/spatial-memory/internal/pkg/sms"
	"github.com/spatial-memory/spatial-memory/internal/pkg/storage"
	"github.com/spatial-memory/spatial-memory/internal/pkg/wechat"
	"github.com/spatial-memory/spatial-memory/internal/repository"
	"github.com/spatial-memory/spatial-memory/internal/router"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().Timestamp().Caller().Logger()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	ctx := context.Background()

	// --- Infrastructure ---
	dbPool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to PostgreSQL")
	}
	defer dbPool.Close()

	if err := database.RunMigrations(cfg.Database.DSN()); err != nil {
		log.Warn().Err(err).Msg("migrations skipped (may already be applied)")
	}

	var redisClient *redis.Client
	if cfg.Redis.Enabled {
		redisClient, err = database.NewRedisClient(ctx, cfg.Redis)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect to Redis")
		}
		defer redisClient.Close()
	} else {
		log.Warn().Msg("Redis is disabled, running without cache")
		// Create a disconnected client so nil checks aren't needed everywhere
		redisClient = redis.NewClient(&redis.Options{Addr: "localhost:0"})
	}

	// --- Repositories ---
	userRepo := repository.NewUserRepository(dbPool)
	authRepo := repository.NewAuthRepository(dbPool)
	memoryRepo := repository.NewMemoryRepository(dbPool)
	permRepo := repository.NewPermissionRepository(dbPool)
	circleRepo := repository.NewCircleRepository(dbPool)
	var spatialCache repository.SpatialCache
	if cfg.Redis.Enabled {
		spatialCache = repository.NewSpatialCache(redisClient)
	} else {
		spatialCache = repository.NewNoOpSpatialCache()
	}
	interactionRepo := repository.NewInteractionRepository(dbPool)
	moderationRepo := repository.NewModerationRepository(dbPool)
	mediaRepo := repository.NewMediaRepository(dbPool)

	// --- External Clients ---
	smsClient := sms.NewClient(cfg.SMS)
	wechatClient := wechat.NewClient(cfg.WeChat)
	r2Client, err := storage.NewClient(cfg.R2)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create R2 storage client")
	}
	glmClient := moderation.NewClient(cfg.GLM)

	// --- Services ---
	tokenService := service.NewTokenService(cfg.JWT, authRepo, userRepo)
	authService := service.NewAuthService(userRepo, authRepo, tokenService, smsClient, wechatClient, redisClient)
	memoryService := service.NewMemoryService(memoryRepo, spatialCache, permRepo)
	interactionService := service.NewInteractionService(interactionRepo, memoryRepo, moderationRepo)
	uploadService := service.NewUploadService(r2Client, memoryRepo, mediaRepo, cfg.R2.PublicURL)
	circleService := service.NewCircleService(circleRepo, userRepo)
	permissionService := service.NewPermissionService(permRepo, memoryRepo, circleRepo)
	moderationService := service.NewModerationService(moderationRepo, memoryRepo, glmClient)

	// --- Handlers ---
	healthHandler := handler.NewHealthHandler(dbPool, redisClient)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userRepo)
	memoryHandler := handler.NewMemoryHandler(memoryService)
	interactionHandler := handler.NewInteractionHandler(interactionService)
	uploadHandler := handler.NewUploadHandler(uploadService)
	circleHandler := handler.NewCircleHandler(circleService)
	permissionHandler := handler.NewPermissionHandler(permissionService)
	moderationHandler := handler.NewModerationHandler(moderationService)

	// --- Router ---
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	r.Use(gin.Recovery())

	router.Setup(r, router.Config{
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

	// --- Server ---
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info().Int("port", cfg.Server.Port).Msg("starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	// Start moderation worker
	moderationService.StartWorker(5 * time.Minute)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server forced shutdown")
	}

	// Stop moderation worker
	moderationService.StopWorker()

	log.Info().Msg("server stopped")
}
