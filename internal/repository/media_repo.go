package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spatial-memory/spatial-memory/internal/model"
)

// MediaRepository handles media metadata storage.
type MediaRepository interface {
	Create(ctx context.Context, media *model.MemoryMedia) error
	GetByID(ctx context.Context, id int64) (*model.MemoryMedia, error)
	ListByMemoryID(ctx context.Context, memoryID int64) ([]*model.MemoryMedia, error)
	FindByHash(ctx context.Context, contentHash string) (*model.MemoryMedia, error)
	FindByStorageKey(ctx context.Context, storageKey string) (*model.MemoryMedia, error)
	Delete(ctx context.Context, id int64) error
	DeleteByMemoryID(ctx context.Context, memoryID int64) error
}

type pgxMediaRepo struct {
	pool *pgxpool.Pool
}

func NewMediaRepository(pool *pgxpool.Pool) MediaRepository {
	return &pgxMediaRepo{pool: pool}
}

func (r *pgxMediaRepo) Create(ctx context.Context, media *model.MemoryMedia) error {
	query := `
		INSERT INTO memory_media (memory_id, media_type, storage_key, content_hash, size_bytes, mime_type, width, height, duration_seconds, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(ctx, query,
		media.MemoryID,
		media.MediaType,
		media.StorageKey,
		media.ContentHash,
		media.SizeBytes,
		media.MimeType,
		media.Width,
		media.Height,
		media.DurationSeconds,
		media.Status,
	).Scan(&media.ID, &media.CreatedAt)

	if err != nil {
		return fmt.Errorf("create media: %w", err)
	}

	return nil
}

func (r *pgxMediaRepo) GetByID(ctx context.Context, id int64) (*model.MemoryMedia, error) {
	query := `
		SELECT id, memory_id, media_type, storage_key, content_hash, size_bytes, mime_type,
		       width, height, duration_seconds, status, created_at, updated_at
		FROM memory_media
		WHERE id = $1
	`

	media := &model.MemoryMedia{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&media.ID,
		&media.MemoryID,
		&media.MediaType,
		&media.StorageKey,
		&media.ContentHash,
		&media.SizeBytes,
		&media.MimeType,
		&media.Width,
		&media.Height,
		&media.DurationSeconds,
		&media.Status,
		&media.CreatedAt,
		&media.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get media: %w", err)
	}

	return media, nil
}

func (r *pgxMediaRepo) ListByMemoryID(ctx context.Context, memoryID int64) ([]*model.MemoryMedia, error) {
	query := `
		SELECT id, memory_id, media_type, storage_key, content_hash, size_bytes, mime_type,
		       width, height, duration_seconds, status, created_at, updated_at
		FROM memory_media
		WHERE memory_id = $1 AND status = $2
		ORDER BY id ASC
	`

	rows, err := r.pool.Query(ctx, query, memoryID, model.MediaStatusActive)
	if err != nil {
		return nil, fmt.Errorf("list media: %w", err)
	}
	defer rows.Close()

	var mediaList []*model.MemoryMedia
	for rows.Next() {
		media := &model.MemoryMedia{}
		err := rows.Scan(
			&media.ID,
			&media.MemoryID,
			&media.MediaType,
			&media.StorageKey,
			&media.ContentHash,
			&media.SizeBytes,
			&media.MimeType,
			&media.Width,
			&media.Height,
			&media.DurationSeconds,
			&media.Status,
			&media.CreatedAt,
			&media.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan media: %w", err)
		}
		mediaList = append(mediaList, media)
	}

	return mediaList, rows.Err()
}

func (r *pgxMediaRepo) FindByHash(ctx context.Context, contentHash string) (*model.MemoryMedia, error) {
	query := `
		SELECT id, memory_id, media_type, storage_key, content_hash, size_bytes, mime_type,
		       width, height, duration_seconds, status, created_at, updated_at
		FROM memory_media
		WHERE content_hash = $1 AND status = $2
		LIMIT 1
	`

	media := &model.MemoryMedia{}
	err := r.pool.QueryRow(ctx, query, contentHash, model.MediaStatusActive).Scan(
		&media.ID,
		&media.MemoryID,
		&media.MediaType,
		&media.StorageKey,
		&media.ContentHash,
		&media.SizeBytes,
		&media.MimeType,
		&media.Width,
		&media.Height,
		&media.DurationSeconds,
		&media.Status,
		&media.CreatedAt,
		&media.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("find media by hash: %w", err)
	}

	return media, nil
}

func (r *pgxMediaRepo) FindByStorageKey(ctx context.Context, storageKey string) (*model.MemoryMedia, error) {
	query := `
		SELECT id, memory_id, media_type, storage_key, content_hash, size_bytes, mime_type,
		       width, height, duration_seconds, status, created_at, updated_at
		FROM memory_media
		WHERE storage_key = $1
		LIMIT 1
	`

	media := &model.MemoryMedia{}
	err := r.pool.QueryRow(ctx, query, storageKey).Scan(
		&media.ID,
		&media.MemoryID,
		&media.MediaType,
		&media.StorageKey,
		&media.ContentHash,
		&media.SizeBytes,
		&media.MimeType,
		&media.Width,
		&media.Height,
		&media.DurationSeconds,
		&media.Status,
		&media.CreatedAt,
		&media.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("find media by storage key: %w", err)
	}

	return media, nil
}

func (r *pgxMediaRepo) Delete(ctx context.Context, id int64) error {
	query := `UPDATE memory_media SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, model.MediaStatusDeleted, id)
	if err != nil {
		return fmt.Errorf("delete media: %w", err)
	}
	return nil
}

func (r *pgxMediaRepo) DeleteByMemoryID(ctx context.Context, memoryID int64) error {
	query := `UPDATE memory_media SET status = $1, updated_at = NOW() WHERE memory_id = $2`
	_, err := r.pool.Exec(ctx, query, model.MediaStatusDeleted, memoryID)
	if err != nil {
		return fmt.Errorf("delete media by memory: %w", err)
	}
	return nil
}
