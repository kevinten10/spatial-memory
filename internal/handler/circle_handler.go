package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/spatial-memory/spatial-memory/internal/model"
	"github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/pkg/response"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

// CircleHandler handles friend circle endpoints.
type CircleHandler struct {
	circleService service.CircleService
}

// NewCircleHandler creates a new circle handler.
func NewCircleHandler(circleService service.CircleService) *CircleHandler {
	return &CircleHandler{circleService: circleService}
}

// Create godoc
// @Summary Create a friend circle
// @Description Create a new friend circle for sharing memories
// @Tags circles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateCircleRequest true "Circle data"
// @Success 201 {object} model.FriendCircle "Created circle"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 409 {object} response.ErrorResponse "Max circles limit reached"
// @Router /circles [post]
func (h *CircleHandler) Create(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	var req model.CreateCircleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, err.Error())
		return
	}

	circle, err := h.circleService.Create(c.Request.Context(), userID.(int64), &req)
	if err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to create circle")
		return
	}

	response.Created(c, circle)
}

// Get godoc
// @Summary Get a circle by ID
// @Description Get a friend circle's details
// @Tags circles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Circle ID"
// @Success 200 {object} model.FriendCircle "Circle details"
// @Failure 400 {object} response.ErrorResponse "Invalid circle ID"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Circle not found"
// @Router /circles/{id} [get]
func (h *CircleHandler) Get(c *gin.Context) {
	circleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid circle id")
		return
	}

	circle, err := h.circleService.GetByID(c.Request.Context(), circleID)
	if err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to get circle")
		return
	}

	response.Success(c, circle)
}

// ListMine godoc
// @Summary List my circles
// @Description Get a list of circles owned by the current user
// @Tags circles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Items per page (default: 20)"
// @Success 200 {array} model.FriendCircle "List of circles"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Router /circles/mine [get]
func (h *CircleHandler) ListMine(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	circles, err := h.circleService.ListMyCircles(c.Request.Context(), userID.(int64), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "failed to list circles")
		return
	}

	response.Success(c, circles)
}

// ListJoined godoc
// @Summary List joined circles
// @Description Get a list of circles the user is a member of
// @Tags circles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Items per page (default: 20)"
// @Success 200 {array} model.FriendCircle "List of circles"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Router /circles/joined [get]
func (h *CircleHandler) ListJoined(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	circles, err := h.circleService.ListJoinedCircles(c.Request.Context(), userID.(int64), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "failed to list circles")
		return
	}

	response.Success(c, circles)
}

// Update godoc
// @Summary Update a circle
// @Description Update a circle's name or description
// @Tags circles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Circle ID"
// @Param request body model.UpdateCircleRequest true "Update fields"
// @Success 200 {object} model.FriendCircle "Updated circle"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Router /circles/{id} [put]
func (h *CircleHandler) Update(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	circleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid circle id")
		return
	}

	var req model.UpdateCircleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, err.Error())
		return
	}

	circle, err := h.circleService.Update(c.Request.Context(), circleID, userID.(int64), &req)
	if err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to update circle")
		return
	}

	response.Success(c, circle)
}

// Delete godoc
// @Summary Delete a circle
// @Description Delete a friend circle
// @Tags circles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Circle ID"
// @Success 200 {object} map[string]string "Deleted successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid circle ID"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Router /circles/{id} [delete]
func (h *CircleHandler) Delete(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	circleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid circle id")
		return
	}

	if err := h.circleService.Delete(c.Request.Context(), circleID, userID.(int64)); err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to delete circle")
		return
	}

	response.Success(c, gin.H{"message": "deleted"})
}

// AddMember godoc
// @Summary Add member to circle
// @Description Add a user to a friend circle
// @Tags circles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Circle ID"
// @Param request body model.AddMemberRequest true "User ID to add"
// @Success 200 {object} map[string]string "Member added"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Failure 409 {object} response.ErrorResponse "Max members limit reached"
// @Router /circles/{id}/members [post]
func (h *CircleHandler) AddMember(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	circleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid circle id")
		return
	}

	var req model.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, err.Error())
		return
	}

	if err := h.circleService.AddMember(c.Request.Context(), circleID, userID.(int64), req.UserID); err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to add member")
		return
	}

	response.Success(c, gin.H{"message": "member added"})
}

// RemoveMember godoc
// @Summary Remove member from circle
// @Description Remove a user from a friend circle
// @Tags circles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Circle ID"
// @Param user_id path int true "User ID to remove"
// @Success 200 {object} map[string]string "Member removed"
// @Failure 400 {object} response.ErrorResponse "Invalid IDs"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Router /circles/{id}/members/{user_id} [delete]
func (h *CircleHandler) RemoveMember(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	circleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid circle id")
		return
	}

	memberID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid user id")
		return
	}

	if err := h.circleService.RemoveMember(c.Request.Context(), circleID, userID.(int64), memberID); err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to remove member")
		return
	}

	response.Success(c, gin.H{"message": "member removed"})
}

// ListMembers godoc
// @Summary List circle members
// @Description Get a list of members in a circle
// @Tags circles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Circle ID"
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Items per page (default: 20)"
// @Success 200 {array} model.CircleMember "List of members"
// @Failure 400 {object} response.ErrorResponse "Invalid circle ID"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Router /circles/{id}/members [get]
func (h *CircleHandler) ListMembers(c *gin.Context) {
	circleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid circle id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	members, err := h.circleService.ListMembers(c.Request.Context(), circleID, page, pageSize)
	if err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to list members")
		return
	}

	response.Success(c, members)
}
