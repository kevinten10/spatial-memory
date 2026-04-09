package service

import (
	"context"
	"fmt"

	"github.com/spatial-memory/spatial-memory/internal/model"
	"github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/repository"
)

// CircleService handles friend circle business logic.
type CircleService interface {
	Create(ctx context.Context, ownerID int64, req *model.CreateCircleRequest) (*model.FriendCircle, error)
	GetByID(ctx context.Context, circleID int64) (*model.FriendCircle, error)
	ListMyCircles(ctx context.Context, userID int64, page, pageSize int) ([]*model.FriendCircle, error)
	ListJoinedCircles(ctx context.Context, userID int64, page, pageSize int) ([]*model.FriendCircle, error)
	Update(ctx context.Context, circleID, userID int64, req *model.UpdateCircleRequest) (*model.FriendCircle, error)
	Delete(ctx context.Context, circleID, userID int64) error
	AddMember(ctx context.Context, circleID, ownerID, memberID int64) error
	RemoveMember(ctx context.Context, circleID, ownerID, memberID int64) error
	ListMembers(ctx context.Context, circleID int64, page, pageSize int) ([]int64, error)
}

type circleService struct {
	circleRepo repository.CircleRepository
	userRepo   repository.UserRepository
}

// NewCircleService creates a new circle service.
func NewCircleService(circleRepo repository.CircleRepository, userRepo repository.UserRepository) CircleService {
	return &circleService{
		circleRepo: circleRepo,
		userRepo:   userRepo,
	}
}

func (s *circleService) Create(ctx context.Context, ownerID int64, req *model.CreateCircleRequest) (*model.FriendCircle, error) {
	// Check circle limit
	count, err := s.circleRepo.CountByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("check circle limit: %w", err)
	}
	if count >= model.MaxCirclesPerUser {
		return nil, errors.Wrap(errors.ErrValidation, fmt.Sprintf("maximum %d circles allowed", model.MaxCirclesPerUser))
	}

	circle := &model.FriendCircle{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     ownerID,
	}

	if err := s.circleRepo.Create(ctx, circle); err != nil {
		return nil, fmt.Errorf("create circle: %w", err)
	}

	return circle, nil
}

func (s *circleService) GetByID(ctx context.Context, circleID int64) (*model.FriendCircle, error) {
	circle, err := s.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return nil, fmt.Errorf("get circle: %w", err)
	}
	if circle == nil {
		return nil, errors.ErrNotFound
	}
	return circle, nil
}

func (s *circleService) ListMyCircles(ctx context.Context, userID int64, page, pageSize int) ([]*model.FriendCircle, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return s.circleRepo.ListByOwner(ctx, userID, page, pageSize)
}

func (s *circleService) ListJoinedCircles(ctx context.Context, userID int64, page, pageSize int) ([]*model.FriendCircle, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return s.circleRepo.ListByMember(ctx, userID, page, pageSize)
}

func (s *circleService) Update(ctx context.Context, circleID, userID int64, req *model.UpdateCircleRequest) (*model.FriendCircle, error) {
	circle, err := s.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return nil, fmt.Errorf("get circle: %w", err)
	}
	if circle == nil {
		return nil, errors.ErrNotFound
	}

	if circle.OwnerID != userID {
		return nil, errors.ErrForbidden
	}

	if req.Name != nil {
		circle.Name = *req.Name
	}
	if req.Description != nil {
		circle.Description = *req.Description
	}

	if err := s.circleRepo.Update(ctx, circle); err != nil {
		return nil, fmt.Errorf("update circle: %w", err)
	}

	return circle, nil
}

func (s *circleService) Delete(ctx context.Context, circleID, userID int64) error {
	circle, err := s.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return fmt.Errorf("get circle: %w", err)
	}
	if circle == nil {
		return errors.ErrNotFound
	}

	if circle.OwnerID != userID {
		return errors.ErrForbidden
	}

	return s.circleRepo.Delete(ctx, circleID)
}

func (s *circleService) AddMember(ctx context.Context, circleID, ownerID, memberID int64) error {
	circle, err := s.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return fmt.Errorf("get circle: %w", err)
	}
	if circle == nil {
		return errors.ErrNotFound
	}

	if circle.OwnerID != ownerID {
		return errors.ErrForbidden
	}

	// Check member limit
	memberCount, err := s.circleRepo.CountMembers(ctx, circleID)
	if err != nil {
		return fmt.Errorf("count members: %w", err)
	}
	if memberCount >= model.MaxMembersPerCircle {
		return errors.Wrap(errors.ErrValidation, fmt.Sprintf("maximum %d members allowed", model.MaxMembersPerCircle))
	}

	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, memberID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return errors.ErrNotFound
	}

	return s.circleRepo.AddMember(ctx, circleID, memberID)
}

func (s *circleService) RemoveMember(ctx context.Context, circleID, ownerID, memberID int64) error {
	circle, err := s.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return fmt.Errorf("get circle: %w", err)
	}
	if circle == nil {
		return errors.ErrNotFound
	}

	if circle.OwnerID != ownerID {
		return errors.ErrForbidden
	}

	// Cannot remove owner
	if memberID == circle.OwnerID {
		return errors.Wrap(errors.ErrValidation, "cannot remove circle owner")
	}

	return s.circleRepo.RemoveMember(ctx, circleID, memberID)
}

func (s *circleService) ListMembers(ctx context.Context, circleID int64, page, pageSize int) ([]int64, error) {
	circle, err := s.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return nil, fmt.Errorf("get circle: %w", err)
	}
	if circle == nil {
		return nil, errors.ErrNotFound
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return s.circleRepo.ListMembers(ctx, circleID, page, pageSize)
}
