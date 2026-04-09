package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/spatial-memory/spatial-memory/internal/model"
	"github.com/spatial-memory/spatial-memory/internal/pkg/moderation"
	"github.com/spatial-memory/spatial-memory/internal/repository"
)

// ModerationService handles content moderation business logic.
type ModerationService interface {
	SubmitForModeration(ctx context.Context, memoryID int64) error
	ProcessQueue(batchSize int) error
	ManualReview(ctx context.Context, moderationID int64, approved bool, note string, reviewerID int64) error
	StartWorker(interval time.Duration)
	StopWorker()
	GetQueue(ctx context.Context, status model.ModerationStatus, page, pageSize int) ([]*model.ModerationItem, int64, error)
	GetStats(ctx context.Context) (*model.ModerationStats, error)
	GetItem(ctx context.Context, id int64) (*model.ModerationItem, error)
}

type moderationService struct {
	moderationRepo repository.ModerationRepository
	memoryRepo     repository.MemoryRepository
	glmClient      moderation.GLMClient

	workerCtx    context.Context
	workerCancel context.CancelFunc
	workerWg     sync.WaitGroup
	workerMu     sync.Mutex
}

// NewModerationService creates a new moderation service.
func NewModerationService(
	moderationRepo repository.ModerationRepository,
	memoryRepo repository.MemoryRepository,
	glmClient moderation.GLMClient,
) ModerationService {
	return &moderationService{
		moderationRepo: moderationRepo,
		memoryRepo:     memoryRepo,
		glmClient:      glmClient,
	}
}

func (s *moderationService) SubmitForModeration(ctx context.Context, memoryID int64) error {
	_, err := s.moderationRepo.Create(ctx, memoryID)
	if err != nil {
		return fmt.Errorf("submit for moderation: %w", err)
	}
	log.Info().Int64("memory_id", memoryID).Msg("submitted for moderation")
	return nil
}

func (s *moderationService) ProcessQueue(batchSize int) error {
	ctx := context.Background()

	// Fetch pending items
	pending, err := s.moderationRepo.ListPending(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("fetch pending items: %w", err)
	}

	// Process each item
	for _, item := range pending {
		if err := s.processItem(ctx, item); err != nil {
			log.Error().Err(err).Int64("moderation_id", item.ID).Msg("failed to process moderation item")
			// Continue processing other items
		}
	}

	return nil
}

func (s *moderationService) processItem(ctx context.Context, item *model.ModerationItem) error {
	// Get memory details
	memory, err := s.memoryRepo.GetByID(ctx, item.MemoryID)
	if err != nil {
		return fmt.Errorf("get memory: %w", err)
	}

	// Get media for the memory
	media, err := s.memoryRepo.ListMediaByMemory(ctx, item.MemoryID)
	if err != nil {
		log.Warn().Err(err).Int64("memory_id", item.MemoryID).Msg("failed to load media for moderation")
	}

	// Moderate text content
	textResult, err := s.glmClient.ModerateText(ctx, memory.Content)
	if err != nil {
		log.Error().Err(err).Int64("memory_id", item.MemoryID).Msg("text moderation failed, escalating")
		return s.escalateItem(ctx, item.ID, "AI text moderation failed: "+err.Error())
	}

	// Moderate images if present
	imageSafe := true
	var imageCategories []string
	for _, m := range media {
		if m.MediaType == model.MediaTypePhoto {
			imgResult, err := s.glmClient.ModerateImage(ctx, m.URL)
			if err != nil {
				log.Error().Err(err).Int64("memory_id", item.MemoryID).Str("url", m.URL).Msg("image moderation failed, escalating")
				return s.escalateItem(ctx, item.ID, "AI image moderation failed: "+err.Error())
			}
			if !imgResult.Safe {
				imageSafe = false
				imageCategories = append(imageCategories, imgResult.Categories...)
			}
		}
	}

	// Combine results
	finalSafe := textResult.Safe && imageSafe
	var finalCategories []string
	if !textResult.Safe {
		finalCategories = append(finalCategories, textResult.Categories...)
	}
	finalCategories = append(finalCategories, imageCategories...)

	// Deduplicate categories
	finalCategories = uniqueStrings(finalCategories)

	// Calculate confidence (use minimum of text and image if both present)
	confidence := textResult.Confidence
	if !imageSafe {
		confidence = 0.5 // Conservative if image flagged
	}

	// Auto-decision logic
	var status int
	note := ""

	if finalSafe && confidence > 0.95 {
		// Auto-approve
		status = int(model.ModerationStatusApproved)
		note = "Auto-approved by AI (safe score: " + fmt.Sprintf("%.2f", confidence) + ")"
		// Update memory status to active
		if err := s.memoryRepo.UpdateStatus(ctx, item.MemoryID, model.MemoryStatusActive); err != nil {
			log.Error().Err(err).Int64("memory_id", item.MemoryID).Msg("failed to update memory status after approval")
		}
	} else if !finalSafe && confidence > 0.95 {
		// Auto-reject
		status = int(model.ModerationStatusRejected)
		note = "Auto-rejected by AI (unsafe, score: " + fmt.Sprintf("%.2f", confidence) + ", categories: " + fmt.Sprintf("%v", finalCategories) + ")"
		// Update memory status to rejected
		if err := s.memoryRepo.UpdateStatus(ctx, item.MemoryID, model.MemoryStatusRejected); err != nil {
			log.Error().Err(err).Int64("memory_id", item.MemoryID).Msg("failed to update memory status after rejection")
		}
	} else {
		// Escalate for manual review
		return s.escalateItem(ctx, item.ID, "Uncertain AI result (safe: "+fmt.Sprintf("%v", finalSafe)+", score: "+fmt.Sprintf("%.2f", confidence)+")")
	}

	// Update moderation item
	if err := s.moderationRepo.UpdateReview(ctx, item.ID, status, confidence, finalCategories, nil, note); err != nil {
		return fmt.Errorf("update review: %w", err)
	}

	log.Info().
		Int64("moderation_id", item.ID).
		Int64("memory_id", item.MemoryID).
		Int("status", status).
		Msg("moderation processed")

	return nil
}

func (s *moderationService) escalateItem(ctx context.Context, moderationID int64, reason string) error {
	if err := s.moderationRepo.UpdateReview(ctx, moderationID, int(model.ModerationStatusEscalated), 0.5, []string{}, nil, reason); err != nil {
		return fmt.Errorf("escalate item: %w", err)
	}
	return nil
}

func (s *moderationService) ManualReview(ctx context.Context, moderationID int64, approved bool, note string, reviewerID int64) error {
	item, err := s.moderationRepo.GetByID(ctx, moderationID)
	if err != nil {
		return fmt.Errorf("get moderation item: %w", err)
	}

	var status int
	if approved {
		status = int(model.ModerationStatusApproved)
		// Update memory status to active
		if err := s.memoryRepo.UpdateStatus(ctx, item.MemoryID, model.MemoryStatusActive); err != nil {
			return fmt.Errorf("update memory status: %w", err)
		}
	} else {
		status = int(model.ModerationStatusRejected)
		// Update memory status to rejected
		if err := s.memoryRepo.UpdateStatus(ctx, item.MemoryID, model.MemoryStatusRejected); err != nil {
			return fmt.Errorf("update memory status: %w", err)
		}
	}

	if err := s.moderationRepo.UpdateReview(ctx, moderationID, status, 0, []string{}, &reviewerID, note); err != nil {
		return fmt.Errorf("update review: %w", err)
	}

	log.Info().
		Int64("moderation_id", moderationID).
		Int64("reviewer_id", reviewerID).
		Bool("approved", approved).
		Msg("manual review completed")

	return nil
}

func (s *moderationService) StartWorker(interval time.Duration) {
	s.workerMu.Lock()
	defer s.workerMu.Unlock()

	if s.workerCancel != nil {
		log.Warn().Msg("moderation worker already running")
		return
	}

	s.workerCtx, s.workerCancel = context.WithCancel(context.Background())
	s.workerWg.Add(1)

	go func() {
		defer s.workerWg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Run immediately on start
		if err := s.ProcessQueue(10); err != nil {
			log.Error().Err(err).Msg("initial moderation queue processing failed")
		}

		for {
			select {
			case <-ticker.C:
				if err := s.ProcessQueue(10); err != nil {
					log.Error().Err(err).Msg("moderation queue processing failed")
				}
			case <-s.workerCtx.Done():
				log.Info().Msg("moderation worker stopped")
				return
			}
		}
	}()

	log.Info().Dur("interval", interval).Msg("moderation worker started")
}

func (s *moderationService) StopWorker() {
	s.workerMu.Lock()
	defer s.workerMu.Unlock()

	if s.workerCancel != nil {
		s.workerCancel()
		s.workerWg.Wait()
		s.workerCancel = nil
		s.workerCtx = nil
	}
}

func (s *moderationService) GetQueue(ctx context.Context, status model.ModerationStatus, page, pageSize int) ([]*model.ModerationItem, int64, error) {
	return s.moderationRepo.ListByStatus(ctx, status, page, pageSize)
}

func (s *moderationService) GetStats(ctx context.Context) (*model.ModerationStats, error) {
	return s.moderationRepo.GetStats(ctx)
}

func (s *moderationService) GetItem(ctx context.Context, id int64) (*model.ModerationItem, error) {
	return s.moderationRepo.GetByID(ctx, id)
}

// uniqueStrings removes duplicates from a string slice.
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
