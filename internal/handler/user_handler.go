package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/spatial-memory/spatial-memory/internal/middleware"
	"github.com/spatial-memory/spatial-memory/internal/model"
	domainerr "github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/pkg/response"
	"github.com/spatial-memory/spatial-memory/internal/repository"
)

// UserHandler handles user profile endpoints.
type UserHandler struct {
	userRepo repository.UserRepository
}

// NewUserHandler creates a new user handler.
func NewUserHandler(userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// GetMe godoc
// @Summary Get current user profile
// @Description Get the authenticated user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.User "User profile"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 500 {object} response.ErrorResponse "Server error"
// @Router /users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if de, ok := domainerr.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to get user")
		return
	}

	response.Success(c, user)
}

// UpdateMe godoc
// @Summary Update current user profile
// @Description Update the authenticated user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.UpdateUserRequest true "Update fields"
// @Success 200 {object} model.User "Updated user profile"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Router /users/me [put]
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid request")
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "failed to get user")
		return
	}

	if req.Nickname != nil {
		user.Nickname = *req.Nickname
	}
	if req.AvatarURL != nil {
		user.AvatarURL = *req.AvatarURL
	}
	if req.Bio != nil {
		user.Bio = *req.Bio
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "failed to update user")
		return
	}

	response.Success(c, user)
}

// GetUser godoc
// @Summary Get public user profile
// @Description Get a public user profile by ID
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} model.UserProfile "Public user profile"
// @Failure 400 {object} response.ErrorResponse "Invalid user ID"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "User not found"
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid user id")
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if de, ok := domainerr.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to get user")
		return
	}

	response.Success(c, user.ToProfile())
}
