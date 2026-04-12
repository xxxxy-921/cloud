package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"metis/internal/pkg/token"
)

// JWTAuth returns a Gin middleware that validates JWT access tokens
// and checks the token blacklist for force-terminated sessions.
// Supports both Authorization header and query parameter "token" (for SSE).
func JWTAuth(secret []byte, blacklist *token.TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		// Try Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				abortJSON(c, http.StatusUnauthorized, "invalid authorization format")
				return
			}
			tokenStr = parts[1]
		} else {
			// Fallback to query parameter (for SSE/EventSource)
			tokenStr = c.Query("token")
			if tokenStr == "" {
				abortJSON(c, http.StatusUnauthorized, "missing authorization")
				return
			}
		}

		claims, err := token.ParseToken(tokenStr, secret)
		if err != nil {
			msg := "invalid token"
			if errors.Is(err, jwt.ErrTokenExpired) {
				msg = "token expired"
			}
			abortJSON(c, http.StatusUnauthorized, msg)
			return
		}

		// Check blacklist for force-terminated sessions
		if blacklist.IsBlocked(claims.ID) {
			abortJSON(c, http.StatusUnauthorized, "session terminated")
			return
		}

		c.Set("userId", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Set("tokenJTI", claims.ID)
		c.Set("passwordChangedAt", claims.PasswordChangedAt)
		c.Set("forcePasswordReset", claims.ForcePasswordReset)
		c.Next()
	}
}
