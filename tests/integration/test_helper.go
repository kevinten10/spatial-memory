//go:build integration

package integration

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/spatial-memory/spatial-memory/internal/config"
	"github.com/spatial-memory/spatial-memory/internal/handler"
	"github.com/spatial-memory/spatial-memory/internal/pkg/sms"
	"github.com/spatial-memory/spatial-memory/internal/pkg/storage"
	"github.com/spatial-memory/spatial-memory/internal/pkg/wechat"
	"github.com/spatial-memory/spatial-memory/internal/repository"
	"github.com/spatial-memory/spatial-memory/internal/router"
	"github.com/spatial-memory/spatial-memory/internal/service"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestSuite struct {
	DB          *pgxpool.Pool
	Redis       *redis.Client
	Router      *gin.Engine
	TokenService service.TokenService
}

var (
	suite *TestSuite
	ctx   = context.Background()
)

func TestMain(m *testing.M) {
	code, err := runTests(m)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(code)
}

func runTests(m *testing.M) (int, error) {
	// Start PostgreSQL container
	postgresReq := testcontainers.ContainerRequest{
		Image:        "postgis/postgis:16-3.4",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "spatial",
			"POSTGRES_PASSWORD": "spatial",
			"POSTGRES_DB":       "spatial_memory_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}
	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: postgresReq,
		Started:          true,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to start postgres container: %w", err)
	}
	defer postgresC.Terminate(ctx)

	postgresHost, _ := postgresC.Host(ctx)
	postgresPort, _ := postgresC.MappedPort(ctx, "5432")

	// Start Redis container
	redisReq := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(30 * time.Second),
	}
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: redisReq,
		Started:          true,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to start redis container: %w", err)
	}
	defer redisC.Terminate(ctx)

	redisHost, _ := redisC.Host(ctx)
	redisPort, _ := redisC.MappedPort(ctx, "6379")

	// Setup database connection
	dsn := fmt.Sprintf("postgres://spatial:spatial@%s:%s/spatial_memory_test?sslmode=disable", postgresHost, postgresPort.Port())
	dbPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Run migrations
	if err := runMigrations(dsn); err != nil {
		return 0, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Setup Redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return 0, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// Setup test suite
	suite = setupTestSuite(dbPool, redisClient)

	return m.Run(), nil
}

func runMigrations(dsn string) error {
	// For simplicity, we'll run migrations directly using SQL
	// In production, use golang-migrate
	return nil
}

func setupTestSuite(db *pgxpool.Pool, redisClient *redis.Client) *TestSuite {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Mode: "test",
			Port: 8080,
		},
		JWT: config.JWTConfig{
			Secret:           "test-secret-key",
			AccessTokenTTL:   2 * time.Hour,
			RefreshTokenTTL:  30 * 24 * time.Hour,
		},
	}

	// Repositories
	userRepo := repository.NewUserRepository(db)
	authRepo := repository.NewAuthRepository(db)
	memoryRepo := repository.NewMemoryRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	circleRepo := repository.NewCircleRepository(db)
	spatialCache := repository.NewSpatialCache(redisClient)
	interactionRepo := repository.NewInteractionRepository(db)
	moderationRepo := repository.NewModerationRepository(db)
	mediaRepo := repository.NewMediaRepository(db)

	// External clients (mocked)
	smsClient := sms.NewClient(cfg.SMS)
	wechatClient := wechat.NewClient(cfg.WeChat)
	r2Client, _ := storage.NewClient(cfg.R2)

	// Services
	tokenService := service.NewTokenService(cfg.JWT, authRepo, userRepo)
	authService := service.NewAuthService(userRepo, authRepo, tokenService, smsClient, wechatClient, redisClient)
	memoryService := service.NewMemoryService(memoryRepo, spatialCache, permRepo)
	interactionService := service.NewInteractionService(interactionRepo, memoryRepo, moderationRepo)
	uploadService := service.NewUploadService(r2Client, memoryRepo, mediaRepo, cfg.R2.PublicURL)
	circleService := service.NewCircleService(circleRepo, userRepo)

	// Handlers
	healthHandler := handler.NewHealthHandler(db, redisClient)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userRepo)
	memoryHandler := handler.NewMemoryHandler(memoryService)
	interactionHandler := handler.NewInteractionHandler(interactionService)
	uploadHandler := handler.NewUploadHandler(uploadService)
	circleHandler := handler.NewCircleHandler(circleService)

	// Router
	gin.SetMode(gin.TestMode)
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
		},
	})

	return &TestSuite{
		DB:          db,
		Redis:       redisClient,
		Router:      r,
		TokenService: tokenService,
	}
}

func (s *TestSuite) CreateTestUser(t *testing.T, phone string) (int64, string) {
	// Create user directly in database
	var userID int64
	err := s.DB.QueryRow(ctx, `
		INSERT INTO users (phone, nickname, status, created_at, updated_at)
		VALUES ($1, $2, 1, NOW(), NOW())
		RETURNING id
	`, phone, "Test User").Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Generate tokens
	tokens, err := s.TokenService.GenerateTokenPair(ctx, userID, false)
	if err != nil {
		t.Fatalf("failed to generate tokens: %v", err)
	}

	return userID, tokens.AccessToken
}

func (s *TestSuite) CleanupTestData(t *testing.T) {
	// Clean up test data
	tables := []string{
		"moderation_queue",
		"memory_interactions",
		"memory_permissions",
		"memory_media",
		"memories",
		"circle_members",
		"friend_circles",
		"refresh_tokens",
		"sms_codes",
		"users",
	}

	for _, table := range tables {
		_, err := s.DB.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("failed to clean up table %s: %v", table, err)
		}
	}

	// Clear Redis
	s.Redis.FlushAll(ctx)
}
