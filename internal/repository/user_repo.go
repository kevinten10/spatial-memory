package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainerr "github.com/spatial-memory/spatial-memory/internal/pkg/errors"

	"github.com/spatial-memory/spatial-memory/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id int64) (*model.User, error)
	GetByPhone(ctx context.Context, phone string) (*model.User, error)
	GetByWeChatOpenID(ctx context.Context, openID string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
}

type pgxUserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &pgxUserRepo{pool: pool}
}

func (r *pgxUserRepo) Create(ctx context.Context, user *model.User) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (phone, wechat_open_id, nickname, avatar_url, bio, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		user.Phone, user.WeChatOpenID, user.Nickname, user.AvatarURL, user.Bio, user.Status,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *pgxUserRepo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, phone, wechat_open_id, nickname, avatar_url, bio, status, is_admin, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(
		&user.ID, &user.Phone, &user.WeChatOpenID,
		&user.Nickname, &user.AvatarURL, &user.Bio,
		&user.Status, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.Wrap(domainerr.ErrNotFound, "user not found")
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

func (r *pgxUserRepo) GetByPhone(ctx context.Context, phone string) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, phone, wechat_open_id, nickname, avatar_url, bio, status, is_admin, created_at, updated_at
		 FROM users WHERE phone = $1`, phone,
	).Scan(
		&user.ID, &user.Phone, &user.WeChatOpenID,
		&user.Nickname, &user.AvatarURL, &user.Bio,
		&user.Status, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.Wrap(domainerr.ErrNotFound, "user not found")
		}
		return nil, fmt.Errorf("get user by phone: %w", err)
	}
	return user, nil
}

func (r *pgxUserRepo) GetByWeChatOpenID(ctx context.Context, openID string) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, phone, wechat_open_id, nickname, avatar_url, bio, status, is_admin, created_at, updated_at
		 FROM users WHERE wechat_open_id = $1`, openID,
	).Scan(
		&user.ID, &user.Phone, &user.WeChatOpenID,
		&user.Nickname, &user.AvatarURL, &user.Bio,
		&user.Status, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.Wrap(domainerr.ErrNotFound, "user not found")
		}
		return nil, fmt.Errorf("get user by wechat open id: %w", err)
	}
	return user, nil
}

func (r *pgxUserRepo) Update(ctx context.Context, user *model.User) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE users SET nickname = $1, avatar_url = $2, bio = $3, status = $4
		 WHERE id = $5`,
		user.Nickname, user.AvatarURL, user.Bio, user.Status, user.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domainerr.Wrap(domainerr.ErrNotFound, "user not found")
	}
	return nil
}
