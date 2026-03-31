package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainerr "github.com/spatial-memory/spatial-memory/internal/pkg/errors"

	"github.com/spatial-memory/spatial-memory/internal/model"
)

type AuthRepository interface {
	// SMS codes
	CreateSMSCode(ctx context.Context, code *model.SMSCode) error
	GetLatestSMSCode(ctx context.Context, phone string) (*model.SMSCode, error)
	MarkSMSCodeUsed(ctx context.Context, id int64) error
	CountSMSCodesToday(ctx context.Context, phone string) (int, error)

	// Refresh tokens
	CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, hash string) (*model.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id int64) error
	RevokeAllUserTokens(ctx context.Context, userID int64) error
}

type pgxAuthRepo struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) AuthRepository {
	return &pgxAuthRepo{pool: pool}
}

// --- SMS Codes ---

func (r *pgxAuthRepo) CreateSMSCode(ctx context.Context, code *model.SMSCode) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO sms_codes (phone, code, expires_at)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		code.Phone, code.Code, code.ExpiresAt,
	).Scan(&code.ID, &code.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert sms code: %w", err)
	}
	return nil
}

func (r *pgxAuthRepo) GetLatestSMSCode(ctx context.Context, phone string) (*model.SMSCode, error) {
	code := &model.SMSCode{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, phone, code, used, expires_at, created_at
		 FROM sms_codes
		 WHERE phone = $1 AND used = FALSE
		 ORDER BY created_at DESC
		 LIMIT 1`, phone,
	).Scan(&code.ID, &code.Phone, &code.Code, &code.Used, &code.ExpiresAt, &code.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.Wrap(domainerr.ErrNotFound, "sms code not found")
		}
		return nil, fmt.Errorf("get latest sms code: %w", err)
	}
	return code, nil
}

func (r *pgxAuthRepo) MarkSMSCodeUsed(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE sms_codes SET used = TRUE WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark sms code used: %w", err)
	}
	return nil
}

func (r *pgxAuthRepo) CountSMSCodesToday(ctx context.Context, phone string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM sms_codes
		 WHERE phone = $1 AND created_at >= $2`,
		phone, time.Now().Truncate(24*time.Hour),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count sms codes today: %w", err)
	}
	return count, nil
}

// --- Refresh Tokens ---

func (r *pgxAuthRepo) CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		token.UserID, token.TokenHash, token.ExpiresAt,
	).Scan(&token.ID, &token.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (r *pgxAuthRepo) GetRefreshTokenByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	token := &model.RefreshToken{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		 FROM refresh_tokens
		 WHERE token_hash = $1`, hash,
	).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.CreatedAt, &token.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.Wrap(domainerr.ErrNotFound, "refresh token not found")
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return token, nil
}

func (r *pgxAuthRepo) RevokeRefreshToken(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *pgxAuthRepo) RevokeAllUserTokens(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW()
		 WHERE user_id = $1 AND revoked_at IS NULL`, userID,
	)
	if err != nil {
		return fmt.Errorf("revoke all user tokens: %w", err)
	}
	return nil
}
