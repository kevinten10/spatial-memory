package repository

import "context"

// ModerationRepository handles content moderation queue.
// Full implementation in Chunk 6 (Moderation Pipeline).
type ModerationRepository interface {
	EscalateMemory(ctx context.Context, memoryID int64) error
}

type pgxModerationRepo struct {
	pool interface{} // Will use *pgxpool.Pool
}

// NewModerationRepository creates a stub moderation repository.
// TODO: implement full moderation in Chunk 6.
func NewModerationRepository(pool interface{}) ModerationRepository {
	return &pgxModerationRepo{pool: pool}
}

func (r *pgxModerationRepo) EscalateMemory(ctx context.Context, memoryID int64) error {
	// TODO: implement in Chunk 6
	return nil
}
