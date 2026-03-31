package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/spatial-memory/spatial-memory/internal/model"
	domainerr "github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/pkg/response"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// SendSMSCode sends a verification code to the given phone number.
// POST /api/v1/auth/sms/send
func (h *AuthHandler) SendSMSCode(c *gin.Context) {
	var req model.SendSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid phone number")
		return
	}

	if err := h.authService.SendSMSCode(c.Request.Context(), req.Phone); err != nil {
		if de, ok := domainerr.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "failed to send code")
		return
	}

	response.Success(c, gin.H{"message": "code sent"})
}

// VerifySMSCode verifies the code and returns tokens.
// POST /api/v1/auth/sms/verify
func (h *AuthHandler) VerifySMSCode(c *gin.Context) {
	var req model.VerifySMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid request")
		return
	}

	tokens, err := h.authService.VerifySMSCode(c.Request.Context(), req.Phone, req.Code)
	if err != nil {
		if de, ok := domainerr.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "verification failed")
		return
	}

	response.Success(c, tokens)
}

// WeChatLogin handles WeChat OAuth login.
// POST /api/v1/auth/wechat
func (h *AuthHandler) WeChatLogin(c *gin.Context) {
	var req model.WeChatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid request")
		return
	}

	tokens, err := h.authService.WeChatLogin(c.Request.Context(), req.Code)
	if err != nil {
		if de, ok := domainerr.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "wechat login failed")
		return
	}

	response.Success(c, tokens)
}

// RefreshTokens refreshes the access token using a refresh token.
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshTokens(c *gin.Context) {
	var req model.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid request")
		return
	}

	tokens, err := h.authService.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if de, ok := domainerr.AsDomainError(err); ok {
			response.Error(c, de.Status, de.Code, de.Msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, 50000, "token refresh failed")
		return
	}

	response.Success(c, tokens)
}

// Logout revokes the refresh token.
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var req model.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 40000, "invalid request")
		return
	}

	if err := h.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.Error(c, http.StatusInternalServerError, 50000, "logout failed")
		return
	}

	response.Success(c, gin.H{"message": "logged out"})
}
