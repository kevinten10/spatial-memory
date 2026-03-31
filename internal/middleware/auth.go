package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/spatial-memory/spatial-memory/internal/pkg/response"
	"github.com/spatial-memory/spatial-memory/internal/service"
)

const (
	// ContextKeyUserID is the gin context key for the authenticated user's ID.
	ContextKeyUserID = "user_id"
	// ContextKeyIsAdmin is the gin context key for admin flag.
	ContextKeyIsAdmin = "is_admin"
)

// Auth returns middleware that validates JWT Bearer tokens.
func Auth(tokenService service.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Error(c, http.StatusUnauthorized, 40100, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(c, http.StatusUnauthorized, 40100, "invalid authorization format")
			c.Abort()
			return
		}

		claims, err := tokenService.ValidateAccessToken(parts[1])
		if err != nil {
			response.Error(c, http.StatusUnauthorized, 40100, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyIsAdmin, claims.IsAdmin)
		c.Next()
	}
}

// AdminOnly returns middleware that requires the is_admin flag.
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get(ContextKeyIsAdmin)
		if !exists || !isAdmin.(bool) {
			response.Error(c, http.StatusForbidden, 40300, "admin access required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetUserID extracts the authenticated user ID from the gin context.
func GetUserID(c *gin.Context) int64 {
	id, _ := c.Get(ContextKeyUserID)
	return id.(int64)
}
