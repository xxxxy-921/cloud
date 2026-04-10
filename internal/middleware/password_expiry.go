package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// passwordExpiryWhitelist routes that bypass the password expiry check.
var passwordExpiryWhitelist = map[string]bool{
	"/api/v1/auth/password": true,
	"/api/v1/auth/logout":   true,
	"/api/v1/auth/refresh":  true,
}

// PasswordExpiryCheckFunc reads the password expiry days from config.
type PasswordExpiryCheckFunc func() int

// PasswordExpiry returns a middleware that checks if the user's password has expired.
// It reads passwordChangedAt and forcePasswordReset from JWT claims (set by JWTAuth).
// Returns 409 if password is expired or force-reset is required.
func PasswordExpiry(getExpiryDays PasswordExpiryCheckFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip whitelisted routes
		if passwordExpiryWhitelist[path] {
			c.Next()
			return
		}
		// Skip non-API routes
		if !strings.HasPrefix(path, "/api/") {
			c.Next()
			return
		}

		// Check force reset flag
		forceReset, _ := c.Get("forcePasswordReset")
		if fr, ok := forceReset.(bool); ok && fr {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"code":    -1,
				"message": "password expired",
			})
			return
		}

		// Check password expiry
		expiryDays := getExpiryDays()
		if expiryDays <= 0 {
			c.Next()
			return
		}

		changedAt, _ := c.Get("passwordChangedAt")
		ts, ok := changedAt.(int64)
		if !ok || ts == 0 {
			// No password change timestamp (OAuth user or legacy) — skip
			c.Next()
			return
		}

		changedTime := time.Unix(ts, 0)
		if time.Since(changedTime) > time.Duration(expiryDays)*24*time.Hour {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"code":    -1,
				"message": "password expired",
			})
			return
		}

		c.Next()
	}
}
