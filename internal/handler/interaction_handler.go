package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/spatial-memory/spatial-memory/internal/middleware"
	"github.com/spatial-memory/spatial-memory/internal/pkg/response"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

type InteractionHandler struct {
	interactionService service.InteractionService
}

func NewInteractionHandler(interactionService service.InteractionService) *InteractionHandler {
	return &InteractionHandler{interactionService: interactionService}
}

// Like adds a like to a memory.
// POST /api/v1/memories/:id/like
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

// Unlike removes a like from a memory.
// DELETE /api/v1/memories/:id/like
func (h *InteractionHandler) Unlike(c *gin.Context) {
	h.Like(c) // Toggle handles both like and unlike
}

// Bookmark adds a bookmark to a memory.
// POST /api/v1/memories/:id/bookmark
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

// Unbookmark removes a bookmark from a memory.
// DELETE /api/v1/memories/:id/bookmark
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

// Report reports a memory.
// POST /api/v1/memories/:id/report
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
