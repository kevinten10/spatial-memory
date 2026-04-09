package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/spatial-memory/spatial-memory/internal/model"
	"github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/pkg/storage"
	"github.com/spatial-memory/spatial-memory/internal/repository"
)

// UploadService handles file upload logic and pre-signed URL generation.
type UploadService interface {
	RequestUpload(ctx context.Context, userID int64, req *model.UploadRequest) (*model.UploadResponse, error)
	ConfirmUpload(ctx context.Context, userID int64, req *model.ConfirmUploadRequest) error
}

type uploadService struct {
	storageClient storage.Client
	memoryRepo    repository.MemoryRepository
	mediaRepo     repository.MediaRepository
	publicBaseURL string
}

// NewUploadService creates a new upload service.
func NewUploadService(
	storageClient storage.Client,
	memoryRepo repository.MemoryRepository,
	mediaRepo repository.MediaRepository,
	publicBaseURL string,
) UploadService {
	return &uploadService{
		storageClient: storageClient,
		memoryRepo:    memoryRepo,
		mediaRepo:     mediaRepo,
		publicBaseURL: publicBaseURL,
	}
}

// RequestUpload validates the upload request and generates a pre-signed URL.
func (s *uploadService) RequestUpload(ctx context.Context, userID int64, req *model.UploadRequest) (*model.UploadResponse, error) {
	// 1. Verify memory exists and belongs to user
	memory, err := s.memoryRepo.GetByID(ctx, req.MemoryID)
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}
	if memory == nil {
		return nil, errors.ErrNotFound
	}
	if memory.UserID != userID {
		return nil, errors.ErrForbidden
	}

	// 2. Validate media type range
	if req.MediaType < model.MediaTypePhoto || req.MediaType > model.MediaTypeVoice {
		return nil, errors.ErrInvalidMediaType
	}

	// 3. Validate MIME type is allowed for this media type
	if !model.IsAllowedMimeType(req.MediaType, req.ContentType) {
		return nil, errors.ErrInvalidContentType
	}

	// 4. Validate file size
	if !model.IsAllowedFileSize(req.MediaType, req.FileSize) {
		return nil, errors.ErrFileTooLarge
	}

	// 5. Check for duplicate by content hash
	existing, err := s.mediaRepo.FindByHash(ctx, req.ContentHash)
	if err != nil {
		log.Error().Err(err).Str("hash", req.ContentHash).Msg("failed to check content hash")
	}
	if existing != nil {
		// File already exists, return existing info
		return &model.UploadResponse{
			StorageKey:  existing.StorageKey,
			ExpiresIn:   0,
			ContentHash: req.ContentHash,
			IsDuplicate: true,
			ExistingURL: s.publicBaseURL + "/" + existing.StorageKey,
		}, nil
	}

	// 6. Generate storage key
	storageKey := model.GenerateStorageKey(userID, req.MemoryID, req.MediaType, req.ContentType)

	// 7. Generate pre-signed upload URL
	uploadURL, err := s.storageClient.GeneratePresignedUploadURL(
		ctx,
		storageKey,
		req.ContentType,
		model.PreSignedURLExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("generate presigned url: %w", err)
	}

	return &model.UploadResponse{
		UploadURL:   uploadURL,
		StorageKey:  storageKey,
		ExpiresIn:   int(model.PreSignedURLExpiry.Seconds()),
		ContentHash: req.ContentHash,
		IsDuplicate: false,
	}, nil
}

// ConfirmUpload verifies the file was uploaded and creates the media record.
func (s *uploadService) ConfirmUpload(ctx context.Context, userID int64, req *model.ConfirmUploadRequest) error {
	// 1. Verify the file exists in storage
	objInfo, err := s.storageClient.HeadObject(ctx, req.StorageKey)
	if err != nil {
		return fmt.Errorf("verify upload: %w", err)
	}

	// 2. Check if media already exists for this storage key
	existing, err := s.mediaRepo.FindByStorageKey(ctx, req.StorageKey)
	if err != nil {
		return fmt.Errorf("check existing media: %w", err)
	}
	if existing != nil {
		// Already confirmed, nothing to do
		return nil
	}

	// 3. Parse storage key to extract memory_id
	// Format: memories/<user_id>/<memory_id>/<type>/<uuid>.<ext>
	var keyUserID, memoryID int64
	var mediaType model.MediaType
	_, err = fmt.Sscanf(req.StorageKey, "memories/%d/%d/", &keyUserID, &memoryID)
	if err != nil {
		return fmt.Errorf("parse storage key: %w", err)
	}

	// Verify user owns this memory
	if keyUserID != userID {
		return errors.ErrForbidden
	}

	// Determine media type from path
	switch {
	case contains(req.StorageKey, "/photos/"):
		mediaType = model.MediaTypePhoto
	case contains(req.StorageKey, "/videos/"):
		mediaType = model.MediaTypeVideo
	case contains(req.StorageKey, "/voice/"):
		mediaType = model.MediaTypeVoice
	default:
		mediaType = model.MediaTypePhoto
	}

	// 4. Create media record
	media := &model.MemoryMedia{
		MemoryID:    memoryID,
		MediaType:   mediaType,
		StorageKey:  req.StorageKey,
		ContentHash: req.ContentHash,
		SizeBytes:   objInfo.Size,
		MimeType:    objInfo.ContentType,
		Status:      model.MediaStatusActive,
		CreatedAt:   time.Now(),
	}

	if err := s.mediaRepo.Create(ctx, media); err != nil {
		return fmt.Errorf("create media record: %w", err)
	}

	log.Info().
		Int64("memory_id", memoryID).
		Int64("user_id", userID).
		Str("storage_key", req.StorageKey).
		Int64("size", objInfo.Size).
		Msg("upload confirmed")

	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
