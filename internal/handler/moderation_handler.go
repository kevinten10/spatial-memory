package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/spatial-memory/spatial-memory/internal/middleware"
	"github.com/spatial-memory/spatial-memory/internal/model"
	"github.com/spatial-memory/spatial-memory/internal/pkg/response"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

// ModerationHandler handles admin moderation endpoints.
type ModerationHandler struct {
	moderationService service.ModerationService
}

// NewModerationHandler creates a new moderation handler.
func NewModerationHandler(moderationService service.ModerationService) *ModerationHandler {
	return &ModerationHandler{
		moderationService: moderationService,
	}
}

// ListQueue godoc
// @Summary List moderation queue
// @Description Get the list of memories pending moderation review
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status: pending, approved, rejected"
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Items per page (default: 20)"
// @Success 200 {object} response.PaginatedResponse "Moderation queue"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden - Admin only"
// @Router /admin/moderation/queue [get]
func (h *ModerationHandler) ListQueue(c *gin.Context) {
	var query model.ModerationQueueQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid query parameters")
		return
	}
	query.SetDefaults()

	status := query.ToStatus()
	items, total, err := h.moderationService.GetQueue(c.Request.Context(), status, query.Page, query.PageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "failed to fetch moderation queue")
		return
	}

	response.Paginated(c, items, total, query.Page, query.PageSize)
}

// GetStats godoc
// @Summary Get moderation statistics
// @Description Get statistics about the moderation queue
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.ModerationStats "Moderation statistics"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden - Admin only"
// @Router /admin/moderation/stats [get]
func (h *ModerationHandler) GetStats(c *gin.Context) {
	stats, err := h.moderationService.GetStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "failed to fetch moderation stats")
		return
	}

	response.Success(c, stats)
}

// ManualReview godoc
// @Summary Manual review
// @Description Manually approve or reject a moderation item
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Moderation item ID"
// @Param request body model.ManualReviewRequest true "Review decision"
// @Success 200 {object} map[string]string "Review processed"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden - Admin only"
// @Failure 404 {object} response.ErrorResponse "Item not found"
// @Router /admin/moderation/{id}/review [put]
func (h *ModerationHandler) ManualReview(c *gin.Context) {
	reviewerID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid moderation id")
		return
	}

	var req model.ManualReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid request")
		return
	}

	if err := h.moderationService.ManualReview(c.Request.Context(), id, req.Approved, req.Note, reviewerID); err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "failed to process review")
		return
	}

	response.Success(c, gin.H{"message": "review processed"})
}

// GetItem godoc
// @Summary Get moderation item
// @Description Get a single moderation item by ID
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Moderation item ID"
// @Success 200 {object} model.ModerationItem "Moderation item"
// @Failure 400 {object} response.ErrorResponse "Invalid ID"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden - Admin only"
// @Failure 404 {object} response.ErrorResponse "Item not found"
// @Router /admin/moderation/{id} [get]
func (h *ModerationHandler) GetItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid moderation id")
		return
	}

	item, err := h.moderationService.GetItem(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, 40400, "moderation item not found")
		return
	}

	response.Success(c, item)
}
