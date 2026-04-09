package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainerr "github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/model"
)

// ModerationRepository handles content moderation queue.
type ModerationRepository interface {
	Create(ctx context.Context, memoryID int64) (*model.ModerationItem, error)
	ListPending(ctx context.Context, limit int) ([]*model.ModerationItem, error)
	ListEscalated(ctx context.Context, limit int) ([]*model.ModerationItem, error)
	ListByStatus(ctx context.Context, status model.ModerationStatus, page, pageSize int) ([]*model.ModerationItem, int64, error)
	UpdateReview(ctx context.Context, id int64, status int, safeScore float64, categories []string, reviewerID *int64, note string) error
	GetByID(ctx context.Context, id int64) (*model.ModerationItem, error)
	GetByMemoryID(ctx context.Context, memoryID int64) (*model.ModerationItem, error)
	IncrementReportCount(ctx context.Context, memoryID int64) error
	GetStats(ctx context.Context) (*model.ModerationStats, error)
	EscalateMemory(ctx context.Context, memoryID int64) error
}

type pgxModerationRepo struct {
	pool *pgxpool.Pool
}

// NewModerationRepository creates a new moderation repository.
func NewModerationRepository(pool *pgxpool.Pool) ModerationRepository {
	return &pgxModerationRepo{pool: pool}
}

func (r *pgxModerationRepo) Create(ctx context.Context, memoryID int64) (*model.ModerationItem, error) {
	item := &model.ModerationItem{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO moderation_queue (memory_id, status, report_count)
		 VALUES ($1, $2, 0)
		 ON CONFLICT (memory_id) DO UPDATE SET
			status = EXCLUDED.status,
			report_count = moderation_queue.report_count,
			reviewed_at = NULL,
			reviewer_id = NULL,
			review_note = ''
		 RETURNING id, memory_id, status, ai_safe_score, ai_categories, reviewer_id, review_note, report_count, created_at, reviewed_at`,
		memoryID, model.ModerationStatusPending,
	).Scan(
		&item.ID, &item.MemoryID, &item.Status,
		&item.AISafeScore, &item.AICategories,
		&item.ReviewerID, &item.ReviewNote,
		&item.ReportCount, &item.CreatedAt, &item.ReviewedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create moderation item: %w", err)
	}
	return item, nil
}

func (r *pgxModerationRepo) ListPending(ctx context.Context, limit int) ([]*model.ModerationItem, error) {
	return r.listByStatusWithJoin(ctx, model.ModerationStatusPending, 1, limit)
}

func (r *pgxModerationRepo) ListEscalated(ctx context.Context, limit int) ([]*model.ModerationItem, error) {
	return r.listByStatusWithJoin(ctx, model.ModerationStatusEscalated, 1, limit)
}

func (r *pgxModerationRepo) listByStatusWithJoin(ctx context.Context, status model.ModerationStatus, page, pageSize int) ([]*model.ModerationItem, error) {
	offset := (page - 1) * pageSize

	rows, err := r.pool.Query(ctx,
		`SELECT
			mq.id, mq.memory_id, mq.status, mq.ai_safe_score, mq.ai_categories,
			mq.reviewer_id, mq.review_note, mq.report_count, mq.created_at, mq.reviewed_at,
			m.title, m.content, m.user_id, u.nickname
		 FROM moderation_queue mq
		 JOIN memories m ON mq.memory_id = m.id
		 JOIN users u ON m.user_id = u.id
		 WHERE mq.status = $1
		 ORDER BY mq.created_at ASC
		 LIMIT $2 OFFSET $3`,
		status, pageSize, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list moderation items: %w", err)
	}
	defer rows.Close()

	items := make([]*model.ModerationItem, 0)
	for rows.Next() {
		item := &model.ModerationItem{}
		var categoriesJSON []byte
		var safeScore *float64
		err := rows.Scan(
			&item.ID, &item.MemoryID, &item.Status,
			&safeScore, &categoriesJSON,
			&item.ReviewerID, &item.ReviewNote,
			&item.ReportCount, &item.CreatedAt, &item.ReviewedAt,
			&item.MemoryTitle, &item.MemoryContent, &item.UserID, &item.UserNickname,
		)
		if err != nil {
			return nil, fmt.Errorf("scan moderation item: %w", err)
		}
		if safeScore != nil {
			item.AISafeScore = safeScore
		}
		if len(categoriesJSON) > 0 {
			_ = json.Unmarshal(categoriesJSON, &item.AICategories)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *pgxModerationRepo) ListByStatus(ctx context.Context, status model.ModerationStatus, page, pageSize int) ([]*model.ModerationItem, int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM moderation_queue WHERE status = $1`,
		status,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count moderation items: %w", err)
	}

	items, err := r.listByStatusWithJoin(ctx, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *pgxModerationRepo) UpdateReview(ctx context.Context, id int64, status int, safeScore float64, categories []string, reviewerID *int64, note string) error {
	categoriesJSON, err := json.Marshal(categories)
	if err != nil {
		return fmt.Errorf("marshal categories: %w", err)
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE moderation_queue
		 SET status = $1, ai_safe_score = $2, ai_categories = $3, reviewer_id = $4, review_note = $5, reviewed_at = NOW()
		 WHERE id = $6`,
		status, safeScore, categoriesJSON, reviewerID, note, id,
	)
	if err != nil {
		return fmt.Errorf("update moderation review: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domainerr.Wrap(domainerr.ErrNotFound, "moderation item not found")
	}
	return nil
}

func (r *pgxModerationRepo) GetByID(ctx context.Context, id int64) (*model.ModerationItem, error) {
	item := &model.ModerationItem{}
	var categoriesJSON []byte
	var safeScore *float64

	err := r.pool.QueryRow(ctx,
		`SELECT
			mq.id, mq.memory_id, mq.status, mq.ai_safe_score, mq.ai_categories,
			mq.reviewer_id, mq.review_note, mq.report_count, mq.created_at, mq.reviewed_at,
			m.title, m.content, m.user_id, u.nickname
		 FROM moderation_queue mq
		 JOIN memories m ON mq.memory_id = m.id
		 JOIN users u ON m.user_id = u.id
		 WHERE mq.id = $1`,
		id,
	).Scan(
		&item.ID, &item.MemoryID, &item.Status,
		&safeScore, &categoriesJSON,
		&item.ReviewerID, &item.ReviewNote,
		&item.ReportCount, &item.CreatedAt, &item.ReviewedAt,
		&item.MemoryTitle, &item.MemoryContent, &item.UserID, &item.UserNickname,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.Wrap(domainerr.ErrNotFound, "moderation item not found")
		}
		return nil, fmt.Errorf("get moderation item: %w", err)
	}

	if safeScore != nil {
		item.AISafeScore = safeScore
	}
	if len(categoriesJSON) > 0 {
		_ = json.Unmarshal(categoriesJSON, &item.AICategories)
	}

	return item, nil
}

func (r *pgxModerationRepo) GetByMemoryID(ctx context.Context, memoryID int64) (*model.ModerationItem, error) {
	item := &model.ModerationItem{}
	var categoriesJSON []byte
	var safeScore *float64

	err := r.pool.QueryRow(ctx,
		`SELECT
			mq.id, mq.memory_id, mq.status, mq.ai_safe_score, mq.ai_categories,
			mq.reviewer_id, mq.review_note, mq.report_count, mq.created_at, mq.reviewed_at,
			m.title, m.content, m.user_id, u.nickname
		 FROM moderation_queue mq
		 JOIN memories m ON mq.memory_id = m.id
		 JOIN users u ON m.user_id = u.id
		 WHERE mq.memory_id = $1`,
		memoryID,
	).Scan(
		&item.ID, &item.MemoryID, &item.Status,
		&safeScore, &categoriesJSON,
		&item.ReviewerID, &item.ReviewNote,
		&item.ReportCount, &item.CreatedAt, &item.ReviewedAt,
		&item.MemoryTitle, &item.MemoryContent, &item.UserID, &item.UserNickname,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.Wrap(domainerr.ErrNotFound, "moderation item not found")
		}
		return nil, fmt.Errorf("get moderation item by memory: %w", err)
	}

	if safeScore != nil {
		item.AISafeScore = safeScore
	}
	if len(categoriesJSON) > 0 {
		_ = json.Unmarshal(categoriesJSON, &item.AICategories)
	}

	return item, nil
}

func (r *pgxModerationRepo) IncrementReportCount(ctx context.Context, memoryID int64) error {
	// Try to update existing row
	tag, err := r.pool.Exec(ctx,
		`UPDATE moderation_queue SET report_count = report_count + 1 WHERE memory_id = $1`,
		memoryID,
	)
	if err != nil {
		return fmt.Errorf("increment report count: %w", err)
	}

	// If no row updated, create one
	if tag.RowsAffected() == 0 {
		_, err = r.Create(ctx, memoryID)
		if err != nil {
			return fmt.Errorf("create moderation item on report: %w", err)
		}
	}

	return nil
}

func (r *pgxModerationRepo) GetStats(ctx context.Context) (*model.ModerationStats, error) {
	stats := &model.ModerationStats{}
	err := r.pool.QueryRow(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE status = 0) as pending,
			COUNT(*) FILTER (WHERE status = 3) as escalated,
			COUNT(*) FILTER (WHERE status = 1) as approved,
			COUNT(*) FILTER (WHERE status = 2) as rejected,
			COUNT(*) as total
		 FROM moderation_queue`,
	).Scan(
		&stats.PendingCount, &stats.EscalatedCount,
		&stats.ApprovedCount, &stats.RejectedCount,
		&stats.TotalCount,
	)
	if err != nil {
		return nil, fmt.Errorf("get moderation stats: %w", err)
	}
	return stats, nil
}

func (r *pgxModerationRepo) EscalateMemory(ctx context.Context, memoryID int64) error {
	// Check if moderation item exists
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM moderation_queue WHERE memory_id = $1)`,
		memoryID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check moderation exists: %w", err)
	}

	if exists {
		_, err = r.pool.Exec(ctx,
			`UPDATE moderation_queue SET status = $1 WHERE memory_id = $2`,
			model.ModerationStatusEscalated, memoryID,
		)
	} else {
		_, err = r.Create(ctx, memoryID)
		if err == nil {
			_, err = r.pool.Exec(ctx,
				`UPDATE moderation_queue SET status = $1 WHERE memory_id = $2`,
				model.ModerationStatusEscalated, memoryID,
			)
		}
	}

	if err != nil {
		return fmt.Errorf("escalate memory: %w", err)
	}
	return nil
}
