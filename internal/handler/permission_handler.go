package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/pkg/response"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

// PermissionHandler handles memory permission endpoints.
type PermissionHandler struct {
	permissionService service.PermissionService
}

// NewPermissionHandler creates a new permission handler.
func NewPermissionHandler(permissionService service.PermissionService) *PermissionHandler {
	return &PermissionHandler{permissionService: permissionService}
}

// GrantCircleAccess godoc
// @Summary Grant circle access
// @Description Grant access to a memory for all members of a circle
// @Tags permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Memory ID"
// @Param request body object{circle_id=int} true "Circle ID"
// @Success 200 {object} map[string]string "Access granted"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Router /memories/{id}/grant/circle [post]
func (h *PermissionHandler) GrantCircleAccess(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	memoryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid memory id")
		return
	}

	var req struct {
		CircleID int64 `json:"circle_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, err.Error())
		return
	}

	if err := h.permissionService.GrantCircleAccess(c.Request.Context(), memoryID, req.CircleID, userID.(int64)); err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to grant circle access")
		return
	}

	response.Success(c, gin.H{"message": "circle access granted"})
}

// GrantUserAccess godoc
// @Summary Grant user access
// @Description Grant access to a memory for a specific user
// @Tags permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Memory ID"
// @Param request body object{user_id=int} true "User ID"
// @Success 200 {object} map[string]string "Access granted"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Router /memories/{id}/grant/user [post]
func (h *PermissionHandler) GrantUserAccess(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	memoryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid memory id")
		return
	}

	var req struct {
		UserID int64 `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, err.Error())
		return
	}

	if err := h.permissionService.GrantUserAccess(c.Request.Context(), memoryID, req.UserID, userID.(int64)); err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to grant user access")
		return
	}

	response.Success(c, gin.H{"message": "user access granted"})
}

// RevokeAccess godoc
// @Summary Revoke access
// @Description Remove access for a circle or user
// @Tags permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Memory ID"
// @Param request body object{circle_id=int,user_id=int} true "Circle ID or User ID"
// @Success 200 {object} map[string]string "Access revoked"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Router /memories/{id}/revoke [post]
func (h *PermissionHandler) RevokeAccess(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	memoryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid memory id")
		return
	}

	var req struct {
		CircleID *int64 `json:"circle_id,omitempty"`
		UserID   *int64 `json:"user_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, err.Error())
		return
	}

	if err := h.permissionService.RevokeAccess(c.Request.Context(), memoryID, req.CircleID, req.UserID, userID.(int64)); err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to revoke access")
		return
	}

	response.Success(c, gin.H{"message": "access revoked"})
}

// GenerateShareToken godoc
// @Summary Generate share token
// @Description Create a shareable token for accessing a memory
// @Tags permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Memory ID"
// @Param request body object{expires_in_hours=int} false "Expiration time in hours"
// @Success 200 {object} map[string]interface{} "Share token"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 403 {object} response.ErrorResponse "Forbidden"
// @Router /memories/{id}/share [post]
func (h *PermissionHandler) GenerateShareToken(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	memoryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid memory id")
		return
	}

	var req struct {
		ExpiresInHours int `json:"expires_in_hours,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, err.Error())
		return
	}

	var expiresIn time.Duration
	if req.ExpiresInHours > 0 {
		expiresIn = time.Duration(req.ExpiresInHours) * time.Hour
	}

	token, err := h.permissionService.GenerateShareToken(c.Request.Context(), memoryID, userID.(int64), expiresIn)
	if err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to generate share token")
		return
	}

	response.Success(c, gin.H{
		"token":      token,
		"expires_in": req.ExpiresInHours,
	})
}
