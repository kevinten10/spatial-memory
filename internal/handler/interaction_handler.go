package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/spatial-memory/spatial-memory/internal/middleware"
	"github.com/spatial-memory/spatial-memory/internal/pkg/response"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

// InteractionHandler handles memory interactions (likes, bookmarks, reports).
type InteractionHandler struct {
	interactionService service.InteractionService
}

// NewInteractionHandler creates a new interaction handler.
func NewInteractionHandler(interactionService service.InteractionService) *InteractionHandler {
	return &InteractionHandler{interactionService: interactionService}
}

// Like godoc
// @Summary Like a memory
// @Description Add a like to a memory (toggle)
// @Tags interactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Memory ID"
// @Success 200 {object} map[string]bool "Like status"
// @Failure 400 {object} response.ErrorResponse "Invalid memory ID"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Router /memories/{id}/like [post]
func (h *InteractionHandler) Like(c *gin.Context) {
	userID := middleware.GetUserID(c)

	memoryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid memory id")
		return
	}

	liked, err := h.interactionService.ToggleLike(c.Request.Context(), memoryID, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "failed to toggle like")
		return
	}

	response.Success(c, gin.H{"liked": liked})
}

// Unlike godoc
// @Summary Unlike a memory
// @Description Remove a like from a memory
// @Tags interactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Memory ID"
// @Success 200 {object} map[string]bool "Like status"
// @Failure 400 {object} response.ErrorResponse "Invalid memory ID"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Router /memories/{id}/like [delete]
func (h *InteractionHandler) Unlike(c *gin.Context) {
	h.Like(c) // Toggle handles both like and unlike
}

// Bookmark godoc
// @Summary Bookmark a memory
// @Description Add a memory to bookmarks
// @Tags interactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Memory ID"
// @Success 200 {object} map[string]bool "Bookmark status"
// @Failure 400 {object} response.ErrorResponse "Invalid memory ID"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Router /memories/{id}/bookmark [post]
func (h *InteractionHandler) Bookmark(c *gin.Context) {
	userID := middleware.GetUserID(c)

	memoryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid memory id")
		return
	}

	if err := h.interactionService.Bookmark(c.Request.Context(), memoryID, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "failed to bookmark")
		return
	}

	response.Success(c, gin.H{"bookmarked": true})
}

// Unbookmark godoc
// @Summary Remove bookmark
// @Description Remove a memory from bookmarks
// @Tags interactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Memory ID"
// @Success 200 {object} map[string]bool "Bookmark status"
// @Failure 400 {object} response.ErrorResponse "Invalid memory ID"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Router /memories/{id}/bookmark [delete]
func (h *InteractionHandler) Unbookmark(c *gin.Context) {
	userID := middleware.GetUserID(c)

	memoryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid memory id")
		return
	}

	if err := h.interactionService.Unbookmark(c.Request.Context(), memoryID, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "failed to unbookmark")
		return
	}

	response.Success(c, gin.H{"bookmarked": false})
}

type reportRequest struct {
	Reason string `json:"reason" binding:"required,max=500"`
}

// Report godoc
// @Summary Report a memory
// @Description Report a memory for moderation review
// @Tags interactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Memory ID"
// @Param request body reportRequest true "Report reason"
// @Success 200 {object} map[string]string "Report submitted"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Router /memories/{id}/report [post]
func (h *InteractionHandler) Report(c *gin.Context) {
	userID := middleware.GetUserID(c)

	memoryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid memory id")
		return
	}

	var req reportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid request")
		return
	}

	if err := h.interactionService.Report(c.Request.Context(), memoryID, userID, req.Reason); err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "failed to report")
		return
	}

	response.Success(c, gin.H{"message": "reported"})
}
