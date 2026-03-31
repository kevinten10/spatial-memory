package model

import "time"

type UserStatus int

const (
	UserStatusBanned UserStatus = 0
	UserStatusActive UserStatus = 1
)

type User struct {
	ID           int64      `json:"id"`
	Phone        *string    `json:"phone,omitempty"`
	WeChatOpenID *string    `json:"-"`
	Nickname     string     `json:"nickname"`
	AvatarURL    string     `json:"avatar_url"`
	Bio          string     `json:"bio"`
	Status       UserStatus `json:"status"`
	IsAdmin      bool       `json:"is_admin,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// UserProfile is the public view of a user (no sensitive fields).
type UserProfile struct {
	ID        int64     `json:"id"`
	Nickname  string    `json:"nickname"`
	AvatarURL string    `json:"avatar_url"`
	Bio       string    `json:"bio"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *User) ToProfile() UserProfile {
	return UserProfile{
		ID:        u.ID,
		Nickname:  u.Nickname,
		AvatarURL: u.AvatarURL,
		Bio:       u.Bio,
		CreatedAt: u.CreatedAt,
	}
}

// UpdateUserRequest is the payload for updating user profile.
type UpdateUserRequest struct {
	Nickname  *string `json:"nickname" binding:"omitempty,max=50"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,url"`
	Bio       *string `json:"bio" binding:"omitempty,max=500"`
}
