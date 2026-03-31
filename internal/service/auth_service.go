package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/spatial-memory/spatial-memory/internal/model"
	domainerr "github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/pkg/sms"
	"github.com/spatial-memory/spatial-memory/internal/pkg/wechat"
	"github.com/spatial-memory/spatial-memory/internal/repository"
)

const (
	smsCodeLength     = 6
	smsCodeExpiry     = 5 * time.Minute
	smsCooldown       = 60 * time.Second
	smsMaxPerDay      = 5
	smsCooldownKeyFmt = "sms:cooldown:%s"
)

type AuthService interface {
	SendSMSCode(ctx context.Context, phone string) error
	VerifySMSCode(ctx context.Context, phone, code string) (*model.TokenPair, error)
	WeChatLogin(ctx context.Context, wxCode string) (*model.TokenPair, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*model.TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
}

type authService struct {
	userRepo     repository.UserRepository
	authRepo     repository.AuthRepository
	tokenService TokenService
	smsClient    sms.Client
	wechatClient wechat.Client
	redis        *redis.Client
}

func NewAuthService(
	userRepo repository.UserRepository,
	authRepo repository.AuthRepository,
	tokenService TokenService,
	smsClient sms.Client,
	wechatClient wechat.Client,
	redisClient *redis.Client,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		authRepo:     authRepo,
		tokenService: tokenService,
		smsClient:    smsClient,
		wechatClient: wechatClient,
		redis:        redisClient,
	}
}

func (s *authService) SendSMSCode(ctx context.Context, phone string) error {
	// Rate limit: 1 per 60s per phone (Redis)
	cooldownKey := fmt.Sprintf(smsCooldownKeyFmt, phone)
	if s.redis.Exists(ctx, cooldownKey).Val() > 0 {
		return domainerr.Wrap(domainerr.ErrRateLimit, "please wait before requesting another code")
	}

	// Rate limit: max 5 per day per phone (DB)
	count, err := s.authRepo.CountSMSCodesToday(ctx, phone)
	if err != nil {
		return fmt.Errorf("count sms codes: %w", err)
	}
	if count >= smsMaxPerDay {
		return domainerr.Wrap(domainerr.ErrRateLimit, "daily SMS limit reached")
	}

	// Generate 6-digit code
	code, err := generateCode(smsCodeLength)
	if err != nil {
		return fmt.Errorf("generate code: %w", err)
	}

	// Store in DB
	smsCode := &model.SMSCode{
		Phone:     phone,
		Code:      code,
		ExpiresAt: time.Now().Add(smsCodeExpiry),
	}
	if err := s.authRepo.CreateSMSCode(ctx, smsCode); err != nil {
		return fmt.Errorf("store sms code: %w", err)
	}

	// Send via SMS provider
	if err := s.smsClient.SendCode(ctx, phone, code); err != nil {
		log.Error().Err(err).Str("phone", phone).Msg("failed to send SMS")
		return fmt.Errorf("send sms: %w", err)
	}

	// Set cooldown
	s.redis.Set(ctx, cooldownKey, "1", smsCooldown)

	return nil
}

func (s *authService) VerifySMSCode(ctx context.Context, phone, code string) (*model.TokenPair, error) {
	// Get latest unused code for this phone
	smsCode, err := s.authRepo.GetLatestSMSCode(ctx, phone)
	if err != nil {
		return nil, domainerr.Wrap(domainerr.ErrValidation, "invalid verification code")
	}

	// Check expiry
	if time.Now().After(smsCode.ExpiresAt) {
		return nil, domainerr.Wrap(domainerr.ErrValidation, "verification code expired")
	}

	// Check code matches
	if smsCode.Code != code {
		return nil, domainerr.Wrap(domainerr.ErrValidation, "invalid verification code")
	}

	// Mark used
	if err := s.authRepo.MarkSMSCodeUsed(ctx, smsCode.ID); err != nil {
		return nil, fmt.Errorf("mark code used: %w", err)
	}

	// Find or create user
	user, err := s.findOrCreateUserByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}

	return s.tokenService.GenerateTokenPair(ctx, user)
}

func (s *authService) WeChatLogin(ctx context.Context, wxCode string) (*model.TokenPair, error) {
	info, err := s.wechatClient.ExchangeCode(ctx, wxCode)
	if err != nil {
		return nil, domainerr.Wrap(domainerr.ErrValidation, "invalid wechat code")
	}

	user, err := s.findOrCreateUserByWeChat(ctx, info)
	if err != nil {
		return nil, err
	}

	return s.tokenService.GenerateTokenPair(ctx, user)
}

func (s *authService) RefreshTokens(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	return s.tokenService.RefreshTokens(ctx, refreshToken)
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	return s.tokenService.RevokeRefreshToken(ctx, refreshToken)
}

// --- Helpers ---

func (s *authService) findOrCreateUserByPhone(ctx context.Context, phone string) (*model.User, error) {
	user, err := s.userRepo.GetByPhone(ctx, phone)
	if err == nil {
		return user, nil
	}

	// Create new user
	user = &model.User{
		Phone:    &phone,
		Nickname: "用户" + phone[len(phone)-4:], // Last 4 digits as default nickname
		Status:   model.UserStatusActive,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (s *authService) findOrCreateUserByWeChat(ctx context.Context, info *wechat.UserInfo) (*model.User, error) {
	user, err := s.userRepo.GetByWeChatOpenID(ctx, info.OpenID)
	if err == nil {
		return user, nil
	}

	user = &model.User{
		WeChatOpenID: &info.OpenID,
		Nickname:     info.Nickname,
		AvatarURL:    info.Avatar,
		Status:       model.UserStatusActive,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func generateCode(length int) (string, error) {
	code := ""
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code += fmt.Sprintf("%d", n.Int64())
	}
	return code, nil
}
