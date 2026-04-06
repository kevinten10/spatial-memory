package model

import "time"

// Visibility levels for memories.
type Visibility int

const (
	VisibilityPrivate Visibility = 0
	VisibilityCircle  Visibility = 1
	VisibilityPublic  Visibility = 2
)

// MemoryStatus represents the state of a memory.
type MemoryStatus int

const (
	MemoryStatusDeleted       MemoryStatus = 0
	MemoryStatusActive        MemoryStatus = 1
	MemoryStatusPendingReview MemoryStatus = 2
	MemoryStatusRejected      MemoryStatus = 3
)

// MediaType for memory attachments.
type MediaType int

const (
	MediaTypePhoto MediaType = 0
	MediaTypeVideo MediaType = 1
	MediaTypeVoice MediaType = 2
)

// GeoPoint represents a WGS84 coordinate.
type GeoPoint struct {
	Lat float64 `json:"lat" binding:"required,latitude"`
	Lng float64 `json:"lng" binding:"required,longitude"`
}

// Memory is the core entity for spatial content.
type Memory struct {
	ID             int64        `json:"id"`
	UserID         int64        `json:"user_id"`
	Title          string       `json:"title"`
	Content        string       `json:"content"`
	Location       GeoPoint     `json:"location"`
	Address        string       `json:"address"`
	Visibility     Visibility   `json:"visibility"`
	Status         MemoryStatus `json:"status"`
	LikeCount      int          `json:"like_count"`
	ViewCount      int          `json:"view_count"`
	BookmarkCount  int          `json:"bookmark_count"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	Media          []MemoryMedia `json:"media,omitempty"`
	IsLiked        bool         `json:"is_liked,omitempty"`
	IsBookmarked   bool         `json:"is_bookmarked,omitempty"`
	DistanceMeters float64      `json:"distance_meters,omitempty"`
}

// MemoryMedia represents a photo, video, or voice attachment.
type MemoryMedia struct {
	ID           int64     `json:"id"`
	MemoryID     int64     `json:"memory_id"`
	UserID       int64     `json:"user_id"`
	MediaType    MediaType `json:"media_type"`
	StorageKey   string    `json:"-"`
	URL          string    `json:"url"`
	ContentHash  string    `json:"-"`
	FileSize     int64     `json:"file_size"`
	MimeType     string    `json:"mime_type"`
	Duration     int       `json:"duration,omitempty"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
}

// --- Request / Response DTOs ---

type CreateMemoryRequest struct {
	Title      string     `json:"title" binding:"required,max=200"`
	Content    string     `json:"content" binding:"max=5000"`
	Location   GeoPoint   `json:"location" binding:"required"`
	Address    string     `json:"address" binding:"max=500"`
	Visibility Visibility `json:"visibility" binding:"min=0,max=2"`
}

type UpdateMemoryRequest struct {
	Title      *string     `json:"title" binding:"omitempty,max=200"`
	Content    *string     `json:"content" binding:"omitempty,max=5000"`
	Visibility *Visibility `json:"visibility" binding:"omitempty,min=0,max=2"`
}

type NearbyQuery struct {
	Lat       float64 `form:"lat" binding:"required,latitude"`
	Lng       float64 `form:"lng" binding:"required,longitude"`
	Radius    int     `form:"radius" binding:"min=10,max=50000"` // meters
	Sort      string  `form:"sort" binding:"omitempty,oneof=distance recent popular"`
	Limit     int     `form:"limit" binding:"min=1,max=100"`
	Cluster   bool    `form:"cluster"`
	Page      int     `form:"page" binding:"min=1"`
	PageSize  int     `form:"page_size" binding:"min=1,max=100"`
}

func (q *NearbyQuery) SetDefaults() {
	if q.Radius == 0 {
		q.Radius = 1000 // 1km default
	}
	if q.Sort == "" {
		q.Sort = "distance"
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
}
