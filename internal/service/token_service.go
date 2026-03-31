package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/spatial-memory/spatial-memory/internal/config"
	"github.com/spatial-memory/spatial-memory/internal/model"
	domainerr "github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/repository"
)

type TokenService interface {
	GenerateTokenPair(ctx context.Context, user *model.User) (*model.TokenPair, error)
	ValidateAccessToken(tokenString string) (*model.Claims, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*model.TokenPair, error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
	RevokeAllUserTokens(ctx context.Context, userID int64) error
}

type tokenService struct {
	cfg      config.JWTConfig
	authRepo repository.AuthRepository
	userRepo repository.UserRepository
}

func NewTokenService(cfg config.JWTConfig, authRepo repository.AuthRepository, userRepo repository.UserRepository) TokenService {
	return &tokenService{cfg: cfg, authRepo: authRepo, userRepo: userRepo}
}

func (s *tokenService) GenerateTokenPair(ctx context.Context, user *model.User) (*model.TokenPair, error) {
	now := time.Now()

	// Access token
	claims := &model.Claims{
		UserID:  user.ID,
		IsAdmin: user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "spatial-memory",
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Refresh token (random 32 bytes, base64url-encoded)
	rawRefresh := make([]byte, 32)
	if _, err := rand.Read(rawRefresh); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshTokenStr := base64.URLEncoding.EncodeToString(rawRefresh)

	// Store hash in DB
	hash := hashToken(refreshTokenStr)
	record := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: now.Add(s.cfg.RefreshExpiration),
	}
	if err := s.authRepo.CreateRefreshToken(ctx, record); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int64(s.cfg.AccessExpiration.Seconds()),
	}, nil
}

func (s *tokenService) ValidateAccessToken(tokenString string) (*model.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil {
		return nil, domainerr.Wrap(domainerr.ErrUnauthorized, "invalid token")
	}

	claims, ok := token.Claims.(*model.Claims)
	if !ok || !token.Valid {
		return nil, domainerr.Wrap(domainerr.ErrUnauthorized, "invalid token claims")
	}

	return claims, nil
}

func (s *tokenService) RefreshTokens(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	hash := hashToken(refreshToken)

	record, err := s.authRepo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		return nil, domainerr.Wrap(domainerr.ErrUnauthorized, "invalid refresh token")
	}

	// Check revoked
	if record.RevokedAt != nil {
		// Potential token reuse attack — revoke all tokens for this user
		_ = s.authRepo.RevokeAllUserTokens(ctx, record.UserID)
		return nil, domainerr.Wrap(domainerr.ErrUnauthorized, "refresh token revoked")
	}

	// Check expired
	if time.Now().After(record.ExpiresAt) {
		return nil, domainerr.Wrap(domainerr.ErrUnauthorized, "refresh token expired")
	}

	// Rotate: revoke old token
	if err := s.authRepo.RevokeRefreshToken(ctx, record.ID); err != nil {
		return nil, fmt.Errorf("revoke old refresh token: %w", err)
	}

	// Get user for new token pair
	user, err := s.userRepo.GetByID(ctx, record.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user for refresh: %w", err)
	}

	return s.GenerateTokenPair(ctx, user)
}

func (s *tokenService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	hash := hashToken(refreshToken)
	record, err := s.authRepo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		return nil // Already revoked or doesn't exist — idempotent
	}
	return s.authRepo.RevokeRefreshToken(ctx, record.ID)
}

func (s *tokenService) RevokeAllUserTokens(ctx context.Context, userID int64) error {
	return s.authRepo.RevokeAllUserTokens(ctx, userID)
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
