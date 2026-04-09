package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PermissionRepository handles memory access permissions.
type PermissionRepository interface {
	// GrantCircleAccess grants access to a circle
	GrantCircleAccess(ctx context.Context, memoryID, circleID int64) error
	// GrantUserAccess grants access to a specific user
	GrantUserAccess(ctx context.Context, memoryID, userID int64) error
	// GrantTokenAccess creates a share token access
	GrantTokenAccess(ctx context.Context, memoryID int64, tokenHash string, expiresAt *time.Time) error
	// RevokeAccess removes a specific access grant
	RevokeAccess(ctx context.Context, memoryID int64, circleID, userID *int64, tokenHash *string) error
	// RevokeAllAccess removes all access grants for a memory
	RevokeAllAccess(ctx context.Context, memoryID int64) error
	// CanAccess checks if a user can access a memory
	CanAccess(ctx context.Context, memoryID, userID int64, isOwner bool, visibility int) (bool, error)
	// CanAccessByToken checks if a token grants access
	CanAccessByToken(ctx context.Context, memoryID int64, tokenHash string) (bool, error)
	// ListGrantedCircles returns circle IDs that have access
	ListGrantedCircles(ctx context.Context, memoryID int64) ([]int64, error)
	// ListGrantedUsers returns user IDs that have direct access
	ListGrantedUsers(ctx context.Context, memoryID int64) ([]int64, error)
}

type pgxPermissionRepo struct {
	pool *pgxpool.Pool
}

// NewPermissionRepository creates a permission repository.
func NewPermissionRepository(pool *pgxpool.Pool) PermissionRepository {
	return &pgxPermissionRepo{pool: pool}
}

func (r *pgxPermissionRepo) GrantCircleAccess(ctx context.Context, memoryID, circleID int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO memory_permissions (memory_id, circle_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		memoryID, circleID,
	)
	if err != nil {
		return fmt.Errorf("grant circle access: %w", err)
	}
	return nil
}

func (r *pgxPermissionRepo) GrantUserAccess(ctx context.Context, memoryID, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO memory_permissions (memory_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		memoryID, userID,
	)
	if err != nil {
		return fmt.Errorf("grant user access: %w", err)
	}
	return nil
}

func (r *pgxPermissionRepo) GrantTokenAccess(ctx context.Context, memoryID int64, tokenHash string, expiresAt *time.Time) error {
	query := `INSERT INTO memory_permissions (memory_id, token_hash, expires_at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`
	_, err := r.pool.Exec(ctx, query, memoryID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("grant token access: %w", err)
	}
	return nil
}

func (r *pgxPermissionRepo) RevokeAccess(ctx context.Context, memoryID int64, circleID, userID *int64, tokenHash *string) error {
	var query string
	var args []interface{}

	switch {
	case circleID != nil:
		query = `DELETE FROM memory_permissions WHERE memory_id = $1 AND circle_id = $2`
		args = []interface{}{memoryID, *circleID}
	case userID != nil:
		query = `DELETE FROM memory_permissions WHERE memory_id = $1 AND user_id = $2`
		args = []interface{}{memoryID, *userID}
	case tokenHash != nil:
		query = `DELETE FROM memory_permissions WHERE memory_id = $1 AND token_hash = $2`
		args = []interface{}{memoryID, *tokenHash}
	default:
		return fmt.Errorf("must specify circle_id, user_id, or token_hash")
	}

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("revoke access: %w", err)
	}
	return nil
}

func (r *pgxPermissionRepo) RevokeAllAccess(ctx context.Context, memoryID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM memory_permissions WHERE memory_id = $1`,
		memoryID,
	)
	if err != nil {
		return fmt.Errorf("revoke all access: %w", err)
	}
	return nil
}

func (r *pgxPermissionRepo) CanAccess(ctx context.Context, memoryID, userID int64, isOwner bool, visibility int) (bool, error) {
	// Owner always has access
	if isOwner {
		return true, nil
	}

	// Public memories are accessible to all
	if visibility == 2 { // VisibilityPublic
		return true, nil
	}

	// Check for direct access or circle membership
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			-- Direct user grant
			SELECT 1 FROM memory_permissions
			WHERE memory_id = $1 AND user_id = $2
			UNION
			-- Circle membership grant
			SELECT 1 FROM memory_permissions mp
			JOIN circle_members cm ON mp.circle_id = cm.circle_id
			WHERE mp.memory_id = $1 AND cm.user_id = $2
		) grants
	`, memoryID, userID).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("check access: %w", err)
	}

	return count > 0, nil
}

func (r *pgxPermissionRepo) CanAccessByToken(ctx context.Context, memoryID int64, tokenHash string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM memory_permissions
		 WHERE memory_id = $1 AND token_hash = $2 AND (expires_at IS NULL OR expires_at > NOW())`,
		memoryID, tokenHash,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check token access: %w", err)
	}
	return count > 0, nil
}

func (r *pgxPermissionRepo) ListGrantedCircles(ctx context.Context, memoryID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT circle_id FROM memory_permissions WHERE memory_id = $1 AND circle_id IS NOT NULL`,
		memoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("list granted circles: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan circle id: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func (r *pgxPermissionRepo) ListGrantedUsers(ctx context.Context, memoryID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id FROM memory_permissions WHERE memory_id = $1 AND user_id IS NOT NULL`,
		memoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("list granted users: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan user id: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// pgconn is imported to handle Postgres-specific errors (e.g., pgconn.PgError for constraint violations)
var _ = &pgconn.PgError{}
