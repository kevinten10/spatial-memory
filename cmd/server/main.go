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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/spatial-memory/spatial-memory/internal/config"
	"github.com/spatial-memory/spatial-memory/internal/database"
	"github.com/spatial-memory/spatial-memory/internal/handler"
	"github.com/spatial-memory/spatial-memory/internal/pkg/sms"
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
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	redisClient, err := database.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redis")
	}
	defer redisClient.Close()

	// --- Repositories ---
	userRepo := repository.NewUserRepository(dbPool)
	authRepo := repository.NewAuthRepository(dbPool)

	// --- External Clients ---
	smsClient := sms.NewClient(cfg.SMS)
	wechatClient := wechat.NewClient(cfg.WeChat)

	// --- Services ---
	tokenService := service.NewTokenService(cfg.JWT, authRepo, userRepo)
	authService := service.NewAuthService(userRepo, authRepo, tokenService, smsClient, wechatClient, redisClient)

	// --- Handlers ---
	healthHandler := handler.NewHealthHandler(dbPool, redisClient)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userRepo)

	// --- Router ---
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	r.Use(gin.Recovery())

	router.Setup(r, router.Config{
		TokenService: tokenService,
		RedisClient:  redisClient,
		Handlers: router.Handlers{
			Health: healthHandler,
			Auth:   authHandler,
			User:   userHandler,
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

	log.Info().Msg("server stopped")
}
