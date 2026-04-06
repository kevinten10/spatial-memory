package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InteractionRepository handles memory interactions (likes, bookmarks, reports).
type InteractionRepository interface {
	ToggleLike(ctx context.Context, memoryID, userID int64) (liked bool, err error)
	HasLiked(ctx context.Context, memoryID, userID int64) (bool, error)

	CreateBookmark(ctx context.Context, memoryID, userID int64) error
	DeleteBookmark(ctx context.Context, memoryID, userID int64) error
	HasBookmarked(ctx context.Context, memoryID, userID int64) (bool, error)
	ListBookmarks(ctx context.Context, userID int64, page, pageSize int) ([]int64, int64, error)

	CreateReport(ctx context.Context, memoryID, userID int64, reason string) error
	CountReports(ctx context.Context, memoryID int64) (int, error)
}

type pgxInteractionRepo struct {
	pool *pgxpool.Pool
}

func NewInteractionRepository(pool *pgxpool.Pool) InteractionRepository {
	return &pgxInteractionRepo{pool: pool}
}

// ToggleLike toggles the like status. Returns true if now liked, false if unliked.
func (r *pgxInteractionRepo) ToggleLike(ctx context.Context, memoryID, userID int64) (bool, error) {
	// Try to insert - if conflict (already liked), delete instead
	_, err := r.pool.Exec(ctx,
		`INSERT INTO memory_interactions (memory_id, user_id, interaction_type)
		 VALUES ($1, $2, 0)
		 ON CONFLICT (memory_id, user_id, interaction_type) DO NOTHING`,
		memoryID, userID,
	)
	if err != nil {
		return false, fmt.Errorf("toggle like insert: %w", err)
	}

	// Check if row was inserted by trying to get row count
	var count int
	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM memory_interactions
		 WHERE memory_id = $1 AND user_id = $2 AND interaction_type = 0`,
		memoryID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check like status: %w", err)
	}

	if count == 0 {
		// Was already liked, so unlike
		_, err = r.pool.Exec(ctx,
			`DELETE FROM memory_interactions
			 WHERE memory_id = $1 AND user_id = $2 AND interaction_type = 0`,
			memoryID, userID,
		)
		if err != nil {
			return false, fmt.Errorf("unlike: %w", err)
		}
		// Decrement like count
		_, _ = r.pool.Exec(ctx,
			`UPDATE memories SET like_count = GREATEST(like_count - 1, 0) WHERE id = $1`,
			memoryID,
		)
		return false, nil
	}

	// Increment like count
	_, _ = r.pool.Exec(ctx,
		`UPDATE memories SET like_count = like_count + 1 WHERE id = $1`,
		memoryID,
	)
	return true, nil
}

func (r *pgxInteractionRepo) HasLiked(ctx context.Context, memoryID, userID int64) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM memory_interactions
		 WHERE memory_id = $1 AND user_id = $2 AND interaction_type = 0`,
		memoryID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check liked: %w", err)
	}
	return count > 0, nil
}

func (r *pgxInteractionRepo) CreateBookmark(ctx context.Context, memoryID, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO memory_interactions (memory_id, user_id, interaction_type)
		 VALUES ($1, $2, 1)
		 ON CONFLICT (memory_id, user_id, interaction_type) DO NOTHING`,
		memoryID, userID,
	)
	if err != nil {
		return fmt.Errorf("create bookmark: %w", err)
	}
	// Increment bookmark count
	_, _ = r.pool.Exec(ctx,
		`UPDATE memories SET bookmark_count = bookmark_count + 1 WHERE id = $1`,
		memoryID,
	)
	return nil
}

func (r *pgxInteractionRepo) DeleteBookmark(ctx context.Context, memoryID, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM memory_interactions
		 WHERE memory_id = $1 AND user_id = $2 AND interaction_type = 1`,
		memoryID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete bookmark: %w", err)
	}
	// Decrement bookmark count
	_, _ = r.pool.Exec(ctx,
		`UPDATE memories SET bookmark_count = GREATEST(bookmark_count - 1, 0) WHERE id = $1`,
		memoryID,
	)
	return nil
}

func (r *pgxInteractionRepo) HasBookmarked(ctx context.Context, memoryID, userID int64) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM memory_interactions
		 WHERE memory_id = $1 AND user_id = $2 AND interaction_type = 1`,
		memoryID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check bookmarked: %w", err)
	}
	return count > 0, nil
}

func (r *pgxInteractionRepo) ListBookmarks(ctx context.Context, userID int64, page, pageSize int) ([]int64, int64, error) {
	offset := (page - 1) * pageSize

	var total int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM memory_interactions
		 WHERE user_id = $1 AND interaction_type = 1`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count bookmarks: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT memory_id FROM memory_interactions
		 WHERE user_id = $1 AND interaction_type = 1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list bookmarks: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("scan bookmark: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, total, rows.Err()
}

func (r *pgxInteractionRepo) CreateReport(ctx context.Context, memoryID, userID int64, reason string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO memory_interactions (memory_id, user_id, interaction_type, report_reason)
		 VALUES ($1, $2, 2, $3)
		 ON CONFLICT (memory_id, user_id, interaction_type) DO NOTHING`,
		memoryID, userID, reason,
	)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	return nil
}

func (r *pgxInteractionRepo) CountReports(ctx context.Context, memoryID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM memory_interactions
		 WHERE memory_id = $1 AND interaction_type = 2`,
		memoryID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count reports: %w", err)
	}
	return count, nil
}
