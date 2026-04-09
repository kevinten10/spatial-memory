package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/repository"
)

// PermissionService handles memory access permission business logic.
type PermissionService interface {
	// GrantCircleAccess grants access to a memory for all members of a circle.
	GrantCircleAccess(ctx context.Context, memoryID, circleID, userID int64) error
	// GrantUserAccess grants access to a memory for a specific user.
	GrantUserAccess(ctx context.Context, memoryID, targetUserID, userID int64) error
	// RevokeAccess removes access for a circle or user.
	RevokeAccess(ctx context.Context, memoryID int64, circleID, userID *int64, currentUserID int64) error
	// GenerateShareToken creates a shareable token for a memory.
	GenerateShareToken(ctx context.Context, memoryID, userID int64, expiresIn time.Duration) (token string, err error)
	// ValidateShareToken checks if a token grants access to a memory.
	ValidateShareToken(ctx context.Context, memoryID int64, token string) (bool, error)
}

type permissionService struct {
	permRepo   repository.PermissionRepository
	memoryRepo repository.MemoryRepository
	circleRepo repository.CircleRepository
}

// NewPermissionService creates a new permission service.
func NewPermissionService(
	permRepo repository.PermissionRepository,
	memoryRepo repository.MemoryRepository,
	circleRepo repository.CircleRepository,
) PermissionService {
	return &permissionService{
		permRepo:   permRepo,
		memoryRepo: memoryRepo,
		circleRepo: circleRepo,
	}
}

// GrantCircleAccess grants access to a memory for all members of a circle.
func (s *permissionService) GrantCircleAccess(ctx context.Context, memoryID, circleID, userID int64) error {
	// Verify memory exists and user is the owner
	memory, err := s.memoryRepo.GetByID(ctx, memoryID)
	if err != nil {
		return fmt.Errorf("get memory: %w", err)
	}
	if memory == nil {
		return errors.ErrNotFound
	}

	if memory.UserID != userID {
		return errors.ErrForbidden
	}

	// Verify circle exists
	circle, err := s.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return fmt.Errorf("get circle: %w", err)
	}
	if circle == nil {
		return errors.ErrNotFound
	}

	// Grant access
	if err := s.permRepo.GrantCircleAccess(ctx, memoryID, circleID); err != nil {
		return fmt.Errorf("grant circle access: %w", err)
	}

	return nil
}

// GrantUserAccess grants access to a memory for a specific user.
func (s *permissionService) GrantUserAccess(ctx context.Context, memoryID, targetUserID, userID int64) error {
	// Verify memory exists and user is the owner
	memory, err := s.memoryRepo.GetByID(ctx, memoryID)
	if err != nil {
		return fmt.Errorf("get memory: %w", err)
	}
	if memory == nil {
		return errors.ErrNotFound
	}

	if memory.UserID != userID {
		return errors.ErrForbidden
	}

	// Cannot grant access to self (owner already has access)
	if targetUserID == userID {
		return errors.Wrap(errors.ErrValidation, "cannot grant access to yourself")
	}

	// Grant access
	if err := s.permRepo.GrantUserAccess(ctx, memoryID, targetUserID); err != nil {
		return fmt.Errorf("grant user access: %w", err)
	}

	return nil
}

// RevokeAccess removes access for a circle or user.
func (s *permissionService) RevokeAccess(ctx context.Context, memoryID int64, circleID, userID *int64, currentUserID int64) error {
	// Verify memory exists and current user is the owner
	memory, err := s.memoryRepo.GetByID(ctx, memoryID)
	if err != nil {
		return fmt.Errorf("get memory: %w", err)
	}
	if memory == nil {
		return errors.ErrNotFound
	}

	if memory.UserID != currentUserID {
		return errors.ErrForbidden
	}

	// Must specify either circleID or userID
	if circleID == nil && userID == nil {
		return errors.Wrap(errors.ErrValidation, "must specify circle_id or user_id")
	}

	// Revoke access
	if err := s.permRepo.RevokeAccess(ctx, memoryID, circleID, userID, nil); err != nil {
		return fmt.Errorf("revoke access: %w", err)
	}

	return nil
}

// GenerateShareToken creates a shareable token for a memory.
// The token is 32 random bytes, stored as SHA-256 hash, returned as base64url-encoded string.
func (s *permissionService) GenerateShareToken(ctx context.Context, memoryID, userID int64, expiresIn time.Duration) (string, error) {
	// Verify memory exists and user is the owner
	memory, err := s.memoryRepo.GetByID(ctx, memoryID)
	if err != nil {
		return "", fmt.Errorf("get memory: %w", err)
	}
	if memory == nil {
		return "", errors.ErrNotFound
	}

	if memory.UserID != userID {
		return "", errors.ErrForbidden
	}

	// Generate 32 random bytes
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	// Create SHA-256 hash for storage
	hash := sha256.Sum256(randomBytes)
	tokenHash := fmt.Sprintf("%x", hash)

	// Calculate expiration
	var expiresAt *time.Time
	if expiresIn > 0 {
		t := time.Now().Add(expiresIn)
		expiresAt = &t
	}

	// Store the hash
	if err := s.permRepo.GrantTokenAccess(ctx, memoryID, tokenHash, expiresAt); err != nil {
		return "", fmt.Errorf("grant token access: %w", err)
	}

	// Return base64url-encoded raw token
	token := base64.URLEncoding.EncodeToString(randomBytes)
	return token, nil
}

// ValidateShareToken checks if a token grants access to a memory.
func (s *permissionService) ValidateShareToken(ctx context.Context, memoryID int64, token string) (bool, error) {
	// Decode base64url token
	randomBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return false, errors.Wrap(errors.ErrValidation, "invalid token format")
	}

	// Verify length (should be 32 bytes)
	if len(randomBytes) != 32 {
		return false, errors.Wrap(errors.ErrValidation, "invalid token length")
	}

	// Calculate SHA-256 hash
	hash := sha256.Sum256(randomBytes)
	tokenHash := fmt.Sprintf("%x", hash)

	// Check against repository
	return s.permRepo.CanAccessByToken(ctx, memoryID, tokenHash)
}
