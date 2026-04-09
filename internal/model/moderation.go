package model

import "time"

// ModerationStatus represents the state of a moderation queue item.
type ModerationStatus int

const (
	ModerationStatusPending    ModerationStatus = 0
	ModerationStatusApproved   ModerationStatus = 1
	ModerationStatusRejected   ModerationStatus = 2
	ModerationStatusEscalated  ModerationStatus = 3
)

// ModerationItem represents a content moderation queue entry.
type ModerationItem struct {
	ID           int64            `json:"id"`
	MemoryID     int64            `json:"memory_id"`
	Status       ModerationStatus `json:"status"`
	AISafeScore  *float64         `json:"ai_safe_score,omitempty"`
	AICategories []string         `json:"ai_categories,omitempty"`
	ReviewerID   *int64           `json:"reviewer_id,omitempty"`
	ReviewNote   string           `json:"review_note,omitempty"`
	ReportCount  int              `json:"report_count"`
	CreatedAt    time.Time        `json:"created_at"`
	ReviewedAt   *time.Time       `json:"reviewed_at,omitempty"`

	// Joined fields
	MemoryTitle   string `json:"memory_title,omitempty"`
	MemoryContent string `json:"memory_content,omitempty"`
	UserID        int64  `json:"user_id,omitempty"`
	UserNickname  string `json:"user_nickname,omitempty"`
}

// ModerationResult represents the result of AI moderation.
type ModerationResult struct {
	Safe       bool     `json:"safe"`
	Confidence float64  `json:"confidence"`
	Categories []string `json:"categories,omitempty"`
}

// ModerationStats represents moderation queue statistics.
type ModerationStats struct {
	PendingCount    int64 `json:"pending_count"`
	EscalatedCount  int64 `json:"escalated_count"`
	ApprovedCount   int64 `json:"approved_count"`
	RejectedCount   int64 `json:"rejected_count"`
	TotalCount      int64 `json:"total_count"`
}

// ManualReviewRequest represents a manual review request payload.
type ManualReviewRequest struct {
	Approved bool   `json:"approved" binding:"required"`
	Note     string `json:"note" binding:"max=500"`
}

// ModerationQueueQuery represents query parameters for listing moderation queue.
type ModerationQueueQuery struct {
	Status   string `form:"status" binding:"omitempty,oneof=pending escalated approved rejected"`
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
}

func (q *ModerationQueueQuery) SetDefaults() {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.Status == "" {
		q.Status = "pending"
	}
}

// ToStatus converts string status to ModerationStatus.
func (q *ModerationQueueQuery) ToStatus() ModerationStatus {
	switch q.Status {
	case "approved":
		return ModerationStatusApproved
	case "rejected":
		return ModerationStatusRejected
	case "escalated":
		return ModerationStatusEscalated
	default:
		return ModerationStatusPending
	}
}
