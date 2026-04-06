package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/spatial-memory/spatial-memory/internal/middleware"
	"github.com/spatial-memory/spatial-memory/internal/model"
	domainerr "github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/pkg/response"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

type MemoryHandler struct {
	memoryService service.MemoryService
}

func NewMemoryHandler(memoryService service.MemoryService) *MemoryHandler {
	return &MemoryHandler{memoryService: memoryService}
}

// Create creates a new memory.
// POST /api/v1/memories
func (h *MemoryHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req model.CreateMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid request")
		return
	}

	memory, err := h.memoryService.Create(c.Request.Context(), userID, &req)
	if err != nil {
		if de, ok := domainerr.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to create memory")
		return
	}

	response.Created(c, memory)
}

// Get returns a memory by ID.
// GET /api/v1/memories/:id
func (h *MemoryHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid memory id")
		return
	}

	memory, err := h.memoryService.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		if de, ok := domainerr.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to get memory")
		return
	}

	// Async view increment
	h.memoryService.IncrementView(c.Request.Context(), id)

	response.Success(c, memory)
}

// Update updates a memory.
// PUT /api/v1/memories/:id
func (h *MemoryHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid memory id")
		return
	}

	var req model.UpdateMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid request")
		return
	}

	memory, err := h.memoryService.Update(c.Request.Context(), id, userID, &req)
	if err != nil {
		if de, ok := domainerr.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to update memory")
		return
	}

	response.Success(c, memory)
}

// Delete deletes a memory.
// DELETE /api/v1/memories/:id
func (h *MemoryHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid memory id")
		return
	}

	if err := h.memoryService.Delete(c.Request.Context(), id, userID); err != nil {
		if de, ok := domainerr.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to delete memory")
		return
	}

	response.Success(c, gin.H{"message": "deleted"})
}

// ListMine returns the authenticated user's memories.
// GET /api/v1/memories/mine
func (h *MemoryHandler) ListMine(c *gin.Context) {
	userID := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	memories, total, err := h.memoryService.ListByUser(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "failed to list memories")
		return
	}

	response.Paginated(c, memories, total, page, pageSize)
}

// Nearby returns memories near a location.
// GET /api/v1/memories/nearby?lat=&lng=&radius=&sort=&limit=
func (h *MemoryHandler) Nearby(c *gin.Context) {
	var query model.NearbyQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid query parameters")
		return
	}

	memories, err := h.memoryService.FindNearby(c.Request.Context(), &query)
	if err != nil {
		if de, ok := domainerr.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to find nearby memories")
		return
	}

	response.Success(c, memories)
}
