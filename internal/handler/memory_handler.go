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

// MemoryHandler handles memory endpoints.
type MemoryHandler struct {
	memoryService service.MemoryService
}

// NewMemoryHandler creates a new memory handler.
func NewMemoryHandler(memoryService service.MemoryService) *MemoryHandler {
	return &MemoryHandler{memoryService: memoryService}
}

// Create godoc
// @Summary Create a new memory
// @Description Create a new memory at a geographic location
// @Tags memories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateMemoryRequest true "Memory data"
// @Success 201 {object} model.Memory "Created memory"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Router /memories [post]
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

// Get godoc
// @Summary Get a memory by ID
// @Description Get a memory's details by its ID
// @Tags memories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Memory ID"
// @Success 200 {object} model.Memory "Memory details"
// @Failure 400 {object} response.ErrorResponse "Invalid memory ID"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Memory not found"
// @Router /memories/{id} [get]
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

// Update godoc
// @Summary Update a memory
// @Description Update a memory's title, content, or visibility
// @Tags memories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Memory ID"
// @Param request body model.UpdateMemoryRequest true "Update fields"
// @Success 200 {object} model.Memory "Updated memory"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 404 {object} response.ErrorResponse "Memory not found"
// @Router /memories/{id} [put]
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

// Delete godoc
// @Summary Delete a memory
// @Description Soft delete a memory by ID
// @Tags memories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Memory ID"
// @Success 200 {object} map[string]string "Deleted successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid memory ID"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Router /memories/{id} [delete]
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

// ListMine godoc
// @Summary List user's memories
// @Description Get a paginated list of the authenticated user's memories
// @Tags memories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Items per page (default: 20)"
// @Success 200 {object} response.PaginatedResponse "List of memories"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Router /memories/mine [get]
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

// Nearby godoc
// @Summary Find nearby memories
// @Description Find memories near a geographic location
// @Tags memories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param lat query number true "Latitude (-90 to 90)"
// @Param lng query number true "Longitude (-180 to 180)"
// @Param radius query int false "Search radius in meters (default: 1000, max: 50000)"
// @Param sort query string false "Sort by: distance, recent, popular (default: distance)"
// @Param limit query int false "Maximum results (default: 20, max: 100)"
// @Param cluster query bool false "Enable clustering for large result sets"
// @Success 200 {array} model.Memory "List of nearby memories"
// @Failure 400 {object} response.ErrorResponse "Invalid query parameters"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Router /memories/nearby [get]
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
