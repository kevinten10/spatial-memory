package service

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/spatial-memory/spatial-memory/internal/repository"
)

// InteractionService handles likes, bookmarks, and reports.
type InteractionService interface {
	ToggleLike(ctx context.Context, memoryID, userID int64) (liked bool, err error)
	HasLiked(ctx context.Context, memoryID, userID int64) (bool, error)

	Bookmark(ctx context.Context, memoryID, userID int64) error
	Unbookmark(ctx context.Context, memoryID, userID int64) error
	HasBookmarked(ctx context.Context, memoryID, userID int64) (bool, error)

	Report(ctx context.Context, memoryID, userID int64, reason string) error
}

type interactionService struct {
	interactionRepo repository.InteractionRepository
	memoryRepo      repository.MemoryRepository
	moderationRepo  repository.ModerationRepository // Will be implemented in Chunk 6
}

func NewInteractionService(
	interactionRepo repository.InteractionRepository,
	memoryRepo repository.MemoryRepository,
	moderationRepo repository.ModerationRepository,
) InteractionService {
	return &interactionService{
		interactionRepo: interactionRepo,
		memoryRepo:      memoryRepo,
		moderationRepo:  moderationRepo,
	}
}

func (s *interactionService) ToggleLike(ctx context.Context, memoryID, userID int64) (bool, error) {
	return s.interactionRepo.ToggleLike(ctx, memoryID, userID)
}

func (s *interactionService) HasLiked(ctx context.Context, memoryID, userID int64) (bool, error) {
	return s.interactionRepo.HasLiked(ctx, memoryID, userID)
}

func (s *interactionService) Bookmark(ctx context.Context, memoryID, userID int64) error {
	return s.interactionRepo.CreateBookmark(ctx, memoryID, userID)
}

func (s *interactionService) Unbookmark(ctx context.Context, memoryID, userID int64) error {
	return s.interactionRepo.DeleteBookmark(ctx, memoryID, userID)
}

func (s *interactionService) HasBookmarked(ctx context.Context, memoryID, userID int64) (bool, error) {
	return s.interactionRepo.HasBookmarked(ctx, memoryID, userID)
}

func (s *interactionService) Report(ctx context.Context, memoryID, userID int64, reason string) error {
	if err := s.interactionRepo.CreateReport(ctx, memoryID, userID, reason); err != nil {
		return fmt.Errorf("create report: %w", err)
	}

	// Check if we should auto-escalate to moderation (3+ distinct reports)
	count, err := s.interactionRepo.CountReports(ctx, memoryID)
	if err != nil {
		log.Warn().Err(err).Int64("memory_id", memoryID).Msg("failed to count reports")
		return nil // Report was created, just couldn't check count
	}

	if count >= 3 {
		// Auto-escalate: add to moderation queue if not already there
		if s.moderationRepo != nil {
			if err := s.moderationRepo.EscalateMemory(ctx, memoryID); err != nil {
				log.Warn().Err(err).Int64("memory_id", memoryID).Msg("failed to escalate memory")
			}
		}
	}

	return nil
}
