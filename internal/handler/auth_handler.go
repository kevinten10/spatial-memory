package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/spatial-memory/spatial-memory/internal/model"
	domainerr "github.com/spatial-memory/spatial-memory/internal/pkg/errors"
	"github.com/spatial-memory/spatial-memory/internal/pkg/response"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// SendSMSCode godoc
// @Summary Send SMS verification code
// @Description Send a 6-digit verification code to the given phone number
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.SendSMSRequest true "Phone number"
// @Success 200 {object} map[string]string "Code sent successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid phone number"
// @Failure 429 {object} response.ErrorResponse "Rate limit exceeded"
// @Router /auth/sms/send [post]
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

// VerifySMSCode godoc
// @Summary Verify SMS code and login
// @Description Verify the SMS code and return access/refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.VerifySMSRequest true "Phone and code"
// @Success 200 {object} model.TokenPair "Login successful"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Invalid code"
// @Router /auth/sms/verify [post]
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

// WeChatLogin godoc
// @Summary WeChat OAuth login
// @Description Login using WeChat authorization code
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.WeChatLoginRequest true "WeChat auth code"
// @Success 200 {object} model.TokenPair "Login successful"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "WeChat login failed"
// @Router /auth/wechat [post]
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

// RefreshTokens godoc
// @Summary Refresh access token
// @Description Get a new access token using a refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} model.TokenPair "New tokens"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 401 {object} response.ErrorResponse "Invalid refresh token"
// @Router /auth/refresh [post]
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

// Logout godoc
// @Summary Logout user
// @Description Revoke the refresh token and logout
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} map[string]string "Logged out successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Router /auth/logout [post]
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
