package model

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenPair holds the access + refresh tokens returned after login.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds until access token expires
}

// Claims are the JWT claims embedded in access tokens.
type Claims struct {
	UserID  int64 `json:"user_id"`
	IsAdmin bool  `json:"is_admin,omitempty"`
	jwt.RegisteredClaims
}

// SMSCode represents a verification code sent to a phone number.
type SMSCode struct {
	ID        int64     `json:"id"`
	Phone     string    `json:"phone"`
	Code      string    `json:"-"`
	Used      bool      `json:"-"`
	ExpiresAt time.Time `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// RefreshToken is the stored (hashed) refresh token record.
type RefreshToken struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"-"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"-"`
}

// --- Request / Response DTOs ---

type SendSMSRequest struct {
	Phone string `json:"phone" binding:"required,e164"`
}

type VerifySMSRequest struct {
	Phone string `json:"phone" binding:"required,e164"`
	Code  string `json:"code" binding:"required,len=6"`
}

type WeChatLoginRequest struct {
	Code string `json:"code" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
