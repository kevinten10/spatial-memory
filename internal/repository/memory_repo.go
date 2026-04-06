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

type MemoryRepository interface {
	Create(ctx context.Context, memory *model.Memory) error
	GetByID(ctx context.Context, id int64) (*model.Memory, error)
	Update(ctx context.Context, memory *model.Memory) error
	SoftDelete(ctx context.Context, id int64) error
	ListByUser(ctx context.Context, userID int64, page, pageSize int) ([]*model.Memory, int64, error)

	// FindNearby returns memories within radius meters of (lat, lng)
	// Filters by visibility and status, orders by distance or recent.
	FindNearby(ctx context.Context, lat, lng float64, radius int, sort string, limit, offset int) ([]*model.Memory, error)

	// Media CRUD
	CreateMedia(ctx context.Context, media *model.MemoryMedia) error
	DeleteMedia(ctx context.Context, id int64) error
	GetMediaByContentHash(ctx context.Context, hash string) (*model.MemoryMedia, error)
	ListMediaByMemory(ctx context.Context, memoryID int64) ([]model.MemoryMedia, error)

	// Counters
	IncrementLikes(ctx context.Context, id int64, delta int) error
	IncrementViews(ctx context.Context, id int64, delta int) error
	IncrementBookmarks(ctx context.Context, id int64, delta int) error
}

type pgxMemoryRepo struct {
	pool *pgxpool.Pool
}

func NewMemoryRepository(pool *pgxpool.Pool) MemoryRepository {
	return &pgxMemoryRepo{pool: pool}
}

func (r *pgxMemoryRepo) Create(ctx context.Context, memory *model.Memory) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO memories (user_id, title, content, location, address, visibility, status)
		 VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography, $6, $7, $8)
		 RETURNING id, created_at, updated_at`,
		memory.UserID, memory.Title, memory.Content,
		memory.Location.Lng, memory.Location.Lat, // PostGIS: ST_MakePoint(x, y) = (lng, lat)
		memory.Address, memory.Visibility, memory.Status,
	).Scan(&memory.ID, &memory.CreatedAt, &memory.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert memory: %w", err)
	}
	return nil
}

func (r *pgxMemoryRepo) GetByID(ctx context.Context, id int64) (*model.Memory, error) {
	memory := &model.Memory{}
	var lat, lng float64
	err := r.pool.QueryRow(ctx,
		`SELECT m.id, m.user_id, m.title, m.content,
		        ST_X(m.location::geometry) as lng, ST_Y(m.location::geometry) as lat,
		        m.address, m.visibility, m.status,
		        m.like_count, m.view_count, m.bookmark_count,
		        m.created_at, m.updated_at
		 FROM memories m
		 WHERE m.id = $1`, id,
	).Scan(
		&memory.ID, &memory.UserID, &memory.Title, &memory.Content,
		&lng, &lat, &memory.Address, &memory.Visibility, &memory.Status,
		&memory.LikeCount, &memory.ViewCount, &memory.BookmarkCount,
		&memory.CreatedAt, &memory.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.Wrap(domainerr.ErrNotFound, "memory not found")
		}
		return nil, fmt.Errorf("get memory by id: %w", err)
	}
	memory.Location = model.GeoPoint{Lat: lat, Lng: lng}
	return memory, nil
}

func (r *pgxMemoryRepo) Update(ctx context.Context, memory *model.Memory) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE memories SET title = $1, content = $2, visibility = $3
		 WHERE id = $4`,
		memory.Title, memory.Content, memory.Visibility, memory.ID,
	)
	if err != nil {
		return fmt.Errorf("update memory: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domainerr.Wrap(domainerr.ErrNotFound, "memory not found")
	}
	return nil
}

func (r *pgxMemoryRepo) SoftDelete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE memories SET status = $1 WHERE id = $2`,
		model.MemoryStatusDeleted, id,
	)
	if err != nil {
		return fmt.Errorf("soft delete memory: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domainerr.Wrap(domainerr.ErrNotFound, "memory not found")
	}
	return nil
}

func (r *pgxMemoryRepo) ListByUser(ctx context.Context, userID int64, page, pageSize int) ([]*model.Memory, int64, error) {
	offset := (page - 1) * pageSize

	// Get total count
	var total int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM memories WHERE user_id = $1 AND status = $2`,
		userID, model.MemoryStatusActive,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count user memories: %w", err)
	}

	// Get paginated results
	rows, err := r.pool.Query(ctx,
		`SELECT m.id, m.user_id, m.title, m.content,
		        ST_X(m.location::geometry) as lng, ST_Y(m.location::geometry) as lat,
		        m.address, m.visibility, m.status,
		        m.like_count, m.view_count, m.bookmark_count,
		        m.created_at, m.updated_at
		 FROM memories m
		 WHERE m.user_id = $1 AND m.status = $2
		 ORDER BY m.created_at DESC
		 LIMIT $3 OFFSET $4`,
		userID, model.MemoryStatusActive, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list user memories: %w", err)
	}
	defer rows.Close()

	memories := make([]*model.Memory, 0)
	for rows.Next() {
		memory := &model.Memory{}
		var lat, lng float64
		err := rows.Scan(
			&memory.ID, &memory.UserID, &memory.Title, &memory.Content,
			&lng, &lat, &memory.Address, &memory.Visibility, &memory.Status,
			&memory.LikeCount, &memory.ViewCount, &memory.BookmarkCount,
			&memory.CreatedAt, &memory.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan memory: %w", err)
		}
		memory.Location = model.GeoPoint{Lat: lat, Lng: lng}
		memories = append(memories, memory)
	}

	return memories, total, rows.Err()
}

func (r *pgxMemoryRepo) FindNearby(ctx context.Context, lat, lng float64, radius int, sort string, limit, offset int) ([]*model.Memory, error) {
	// Build ORDER BY clause based on sort parameter
	orderBy := "distance"
	if sort == "recent" {
		orderBy = "m.created_at DESC"
	} else if sort == "popular" {
		orderBy = "m.like_count DESC, distance"
	}

	query := fmt.Sprintf(
		`SELECT m.id, m.user_id, m.title, m.content,
		        ST_X(m.location::geometry) as lng, ST_Y(m.location::geometry) as lat,
		        m.address, m.visibility, m.status,
		        m.like_count, m.view_count, m.bookmark_count,
		        m.created_at, m.updated_at,
		        ST_Distance(m.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography)::int as distance
		 FROM memories m
		 WHERE m.status = $3
		   AND m.visibility = $4
		   AND ST_DWithin(m.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $5)
		 ORDER BY %s
		 LIMIT $6 OFFSET $7`,
		orderBy,
	)

	rows, err := r.pool.Query(ctx, query, lng, lat, model.MemoryStatusActive, model.VisibilityPublic, radius, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("find nearby memories: %w", err)
	}
	defer rows.Close()

	memories := make([]*model.Memory, 0)
	for rows.Next() {
		memory := &model.Memory{}
		var mlat, mlng float64
		err := rows.Scan(
			&memory.ID, &memory.UserID, &memory.Title, &memory.Content,
			&mlng, &mlat, &memory.Address, &memory.Visibility, &memory.Status,
			&memory.LikeCount, &memory.ViewCount, &memory.BookmarkCount,
			&memory.CreatedAt, &memory.UpdatedAt,
			&memory.DistanceMeters,
		)
		if err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		memory.Location = model.GeoPoint{Lat: mlat, Lng: mlng}
		memories = append(memories, memory)
	}

	return memories, rows.Err()
}

// --- Media CRUD ---

func (r *pgxMemoryRepo) CreateMedia(ctx context.Context, media *model.MemoryMedia) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO memory_media (memory_id, user_id, media_type, storage_key, url, content_hash,
		  file_size, mime_type, duration, width, height, sort_order)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id, created_at`,
		media.MemoryID, media.UserID, media.MediaType, media.StorageKey, media.URL,
		media.ContentHash, media.FileSize, media.MimeType, media.Duration,
		media.Width, media.Height, media.SortOrder,
	).Scan(&media.ID, &media.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert media: %w", err)
	}
	return nil
}

func (r *pgxMemoryRepo) DeleteMedia(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM memory_media WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete media: %w", err)
	}
	return nil
}

func (r *pgxMemoryRepo) GetMediaByContentHash(ctx context.Context, hash string) (*model.MemoryMedia, error) {
	media := &model.MemoryMedia{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, memory_id, user_id, media_type, storage_key, url, content_hash,
		  file_size, mime_type, duration, width, height, sort_order, created_at
		 FROM memory_media WHERE content_hash = $1`,
		hash,
	).Scan(
		&media.ID, &media.MemoryID, &media.UserID, &media.MediaType, &media.StorageKey,
		&media.URL, &media.ContentHash, &media.FileSize, &media.MimeType,
		&media.Duration, &media.Width, &media.Height, &media.SortOrder, &media.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.Wrap(domainerr.ErrNotFound, "media not found")
		}
		return nil, fmt.Errorf("get media by hash: %w", err)
	}
	return media, nil
}

func (r *pgxMemoryRepo) ListMediaByMemory(ctx context.Context, memoryID int64) ([]model.MemoryMedia, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, memory_id, user_id, media_type, storage_key, url, content_hash,
		  file_size, mime_type, duration, width, height, sort_order, created_at
		 FROM memory_media WHERE memory_id = $1 ORDER BY sort_order`,
		memoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("list media: %w", err)
	}
	defer rows.Close()

	medias := make([]model.MemoryMedia, 0)
	for rows.Next() {
		media := model.MemoryMedia{}
		err := rows.Scan(
			&media.ID, &media.MemoryID, &media.UserID, &media.MediaType, &media.StorageKey,
			&media.URL, &media.ContentHash, &media.FileSize, &media.MimeType,
			&media.Duration, &media.Width, &media.Height, &media.SortOrder, &media.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan media: %w", err)
		}
		medias = append(medias, media)
	}

	return medias, rows.Err()
}

// --- Counters ---

func (r *pgxMemoryRepo) IncrementLikes(ctx context.Context, id int64, delta int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE memories SET like_count = like_count + $1 WHERE id = $2`,
		delta, id,
	)
	return err
}

func (r *pgxMemoryRepo) IncrementViews(ctx context.Context, id int64, delta int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE memories SET view_count = view_count + $1 WHERE id = $2`,
		delta, id,
	)
	return err
}

func (r *pgxMemoryRepo) IncrementBookmarks(ctx context.Context, id int64, delta int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE memories SET bookmark_count = bookmark_count + $1 WHERE id = $2`,
		delta, id,
	)
	return err
}
