package model

import "time"

// FriendCircle represents a group of users who can share memories.
type FriendCircle struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     int64     `json:"owner_id"`
	MemberCount int       `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CircleMember represents a user's membership in a circle.
type CircleMember struct {
	ID       int64     `json:"id"`
	CircleID int64     `json:"circle_id"`
	UserID   int64     `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}

// CreateCircleRequest is used to create a new circle.
type CreateCircleRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=50"`
	Description string `json:"description" binding:"max=200"`
}

// UpdateCircleRequest is used to update a circle.
type UpdateCircleRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=50"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=200"`
}

// AddMemberRequest is used to add a member to a circle.
type AddMemberRequest struct {
	UserID int64 `json:"user_id" binding:"required"`
}

// CircleMembershipInfo contains circle info with membership status.
type CircleMembershipInfo struct {
	FriendCircle
	IsMember bool      `json:"is_member"`
	JoinedAt time.Time `json:"joined_at,omitempty"`
}

// Constants for circle limits.
const (
	MaxCirclesPerUser   = 20
	MaxMembersPerCircle = 100
)
