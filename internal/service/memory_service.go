package service

import (
	"context"
	"fmt"
	"math"

	"github.com/rs/zerolog/log"

	"github.com/spatial-memory/spatial-memory/internal/model"
	"github.com/spatial-memory/spatial-memory/internal/repository"
)

type MemoryService interface {
	Create(ctx context.Context, userID int64, req *model.CreateMemoryRequest) (*model.Memory, error)
	GetByID(ctx context.Context, id, requestUserID int64) (*model.Memory, error)
	Update(ctx context.Context, memoryID, userID int64, req *model.UpdateMemoryRequest) (*model.Memory, error)
	Delete(ctx context.Context, memoryID, userID int64) error
	ListByUser(ctx context.Context, userID int64, page, pageSize int) ([]*model.Memory, int64, error)
	FindNearby(ctx context.Context, query *model.NearbyQuery) ([]*model.Memory, error)
	IncrementView(ctx context.Context, memoryID int64)
}

type memoryService struct {
	memoryRepo   repository.MemoryRepository
	spatialCache repository.SpatialCache
	permRepo     repository.PermissionRepository // Will be implemented in Chunk 5
}

func NewMemoryService(
	memoryRepo repository.MemoryRepository,
	spatialCache repository.SpatialCache,
	permRepo repository.PermissionRepository,
) MemoryService {
	return &memoryService{
		memoryRepo:   memoryRepo,
		spatialCache: spatialCache,
		permRepo:     permRepo,
	}
}

func (s *memoryService) Create(ctx context.Context, userID int64, req *model.CreateMemoryRequest) (*model.Memory, error) {
	memory := &model.Memory{
		UserID:     userID,
		Title:      req.Title,
		Content:    req.Content,
		Location:   req.Location,
		Address:    req.Address,
		Visibility: req.Visibility,
		Status:     model.MemoryStatusActive,
	}

	// Public memories require moderation
	if req.Visibility == model.VisibilityPublic {
		memory.Status = model.MemoryStatusPendingReview
	}

	if err := s.memoryRepo.Create(ctx, memory); err != nil {
		return nil, fmt.Errorf("create memory: %w", err)
	}

	// Add to cache if not pending review
	if memory.Status == model.MemoryStatusActive {
		if err := s.spatialCache.AddMemory(ctx, memory.ID, memory.Location.Lat, memory.Location.Lng); err != nil {
			log.Warn().Err(err).Int64("memory_id", memory.ID).Msg("failed to cache memory location")
		}
	}

	return memory, nil
}

func (s *memoryService) GetByID(ctx context.Context, id, requestUserID int64) (*model.Memory, error) {
	memory, err := s.memoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check access permissions
	if !s.canAccess(ctx, memory, requestUserID) {
		return nil, fmt.Errorf("access denied")
	}

	// Load media
	media, err := s.memoryRepo.ListMediaByMemory(ctx, id)
	if err != nil {
		log.Warn().Err(err).Int64("memory_id", id).Msg("failed to load media")
	} else {
		memory.Media = media
	}

	return memory, nil
}

func (s *memoryService) Update(ctx context.Context, memoryID, userID int64, req *model.UpdateMemoryRequest) (*model.Memory, error) {
	memory, err := s.memoryRepo.GetByID(ctx, memoryID)
	if err != nil {
		return nil, err
	}

	// Only owner can update
	if memory.UserID != userID {
		return nil, fmt.Errorf("not authorized")
	}

	if req.Title != nil {
		memory.Title = *req.Title
	}
	if req.Content != nil {
		memory.Content = *req.Content
	}
	if req.Visibility != nil {
		memory.Visibility = *req.Visibility
		// Changing to public requires moderation
		if memory.Visibility == model.VisibilityPublic {
			memory.Status = model.MemoryStatusPendingReview
		}
	}

	if err := s.memoryRepo.Update(ctx, memory); err != nil {
		return nil, fmt.Errorf("update memory: %w", err)
	}

	// Invalidate cache on update
	if err := s.spatialCache.Invalidate(ctx, memory.Location.Lat, memory.Location.Lng, 1000); err != nil {
		log.Warn().Err(err).Int64("memory_id", memory.ID).Msg("failed to invalidate cache")
	}

	return memory, nil
}

func (s *memoryService) Delete(ctx context.Context, memoryID, userID int64) error {
	memory, err := s.memoryRepo.GetByID(ctx, memoryID)
	if err != nil {
		return err
	}

	// Only owner can delete
	if memory.UserID != userID {
		return fmt.Errorf("not authorized")
	}

	if err := s.memoryRepo.SoftDelete(ctx, memoryID); err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}

	// Remove from cache
	if err := s.spatialCache.RemoveMemory(ctx, memoryID); err != nil {
		log.Warn().Err(err).Int64("memory_id", memoryID).Msg("failed to remove from cache")
	}

	return nil
}

func (s *memoryService) ListByUser(ctx context.Context, userID int64, page, pageSize int) ([]*model.Memory, int64, error) {
	return s.memoryRepo.ListByUser(ctx, userID, page, pageSize)
}

func (s *memoryService) FindNearby(ctx context.Context, query *model.NearbyQuery) ([]*model.Memory, error) {
	query.SetDefaults()

	// Try cache first
	cachedIDs, err := s.spatialCache.SearchNearby(ctx, query.Lat, query.Lng, query.Radius)
	if err != nil {
		log.Warn().Err(err).Msg("cache search failed, falling back to DB")
	}

	if len(cachedIDs) > 0 {
		// Batch fetch from DB
		memories := make([]*model.Memory, 0, len(cachedIDs))
		for _, id := range cachedIDs {
			m, err := s.memoryRepo.GetByID(ctx, id)
			if err != nil {
				continue
			}
			// Recalculate distance
			dist := haversine(query.Lat, query.Lng, m.Location.Lat, m.Location.Lng)
			m.DistanceMeters = dist
			memories = append(memories, m)
		}
		return memories, nil
	}

	// Cache miss: query DB
	offset := (query.Page - 1) * query.PageSize
	memories, err := s.memoryRepo.FindNearby(ctx, query.Lat, query.Lng, query.Radius, query.Sort, query.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("find nearby: %w", err)
	}

	// Populate cache
	for _, m := range memories {
		if err := s.spatialCache.AddMemory(ctx, m.ID, m.Location.Lat, m.Location.Lng); err != nil {
			log.Warn().Err(err).Int64("memory_id", m.ID).Msg("failed to cache memory")
		}
	}

	return memories, nil
}

func (s *memoryService) IncrementView(ctx context.Context, memoryID int64) {
	// Async increment via goroutine
	go func() {
		if err := s.memoryRepo.IncrementViews(ctx, memoryID, 1); err != nil {
			log.Warn().Err(err).Int64("memory_id", memoryID).Msg("failed to increment view")
		}
	}()
}

// canAccess checks if requestUserID can view the memory
func (s *memoryService) canAccess(ctx context.Context, memory *model.Memory, requestUserID int64) bool {
	// Owner always has access
	if memory.UserID == requestUserID {
		return true
	}

	// Deleted memories not accessible
	if memory.Status == model.MemoryStatusDeleted {
		return false
	}

	switch memory.Visibility {
	case model.VisibilityPublic:
		// Public memories are accessible if active (not pending review)
		return memory.Status == model.MemoryStatusActive
	case model.VisibilityCircle:
		// TODO: Check if requestUserID is in owner's circle (Chunk 5)
		return false
	case model.VisibilityPrivate:
		return false
	}

	return false
}

// haversine calculates distance in meters between two WGS84 points
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Earth radius in meters

	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	deltaPhi := (lat2 - lat1) * math.Pi / 180
	deltaLambda := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
