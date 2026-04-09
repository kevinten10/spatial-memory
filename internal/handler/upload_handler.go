package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/spatial-memory/spatial-memory/internal/model"
	"github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/pkg/response"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

// UploadHandler handles media upload endpoints.
type UploadHandler struct {
	uploadService service.UploadService
}

// NewUploadHandler creates a new upload handler.
func NewUploadHandler(uploadService service.UploadService) *UploadHandler {
	return &UploadHandler{uploadService: uploadService}
}

// RequestUpload godoc
// @Summary Request pre-signed upload URL
// @Description Get a pre-signed URL for direct upload to R2 storage
// @Tags uploads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.UploadRequest true "Upload request"
// @Success 200 {object} model.UploadResponse "Pre-signed URL"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 413 {object} response.ErrorResponse "File too large"
// @Router /uploads/request [post]
func (h *UploadHandler) RequestUpload(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	var req model.UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, err.Error())
		return
	}

	resp, err := h.uploadService.RequestUpload(c.Request.Context(), userID.(int64), &req)
	if err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to request upload")
		return
	}

	response.Success(c, resp)
}

// ConfirmUpload godoc
// @Summary Confirm upload completion
// @Description Confirm that a file has been uploaded to R2 and create media record
// @Tags uploads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.ConfirmUploadRequest true "Confirm upload"
// @Success 200 {object} map[string]string "Upload confirmed"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized"
// @Failure 404 {object} response.ErrorResponse "Upload not found"
// @Router /uploads/confirm [post]
func (h *UploadHandler) ConfirmUpload(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	var req model.ConfirmUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, err.Error())
		return
	}

	if err := h.uploadService.ConfirmUpload(c.Request.Context(), userID.(int64), &req); err != nil {
		if de, ok := errors.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to confirm upload")
		return
	}

	response.Success(c, gin.H{"message": "upload confirmed"})
}
