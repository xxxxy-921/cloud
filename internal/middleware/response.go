package middleware

import "github.com/gin-gonic/gin"

// abortJSON sends a JSON error response and aborts the middleware chain.
// Uses the same format as handler.R to keep responses consistent.
func abortJSON(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{
		"code":    -1,
		"message": msg,
	})
}
