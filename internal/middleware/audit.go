package middleware

import (
	"github.com/gin-gonic/gin"

	"metis/internal/model"
	"metis/internal/service"
)

// Audit returns a Gin middleware that captures operation audit logs.
// It runs after the handler. If the handler set "audit_action" in the context
// and the response status is 2xx, an audit log is written asynchronously.
func Audit(auditSvc *service.AuditLogService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only record if handler declared audit metadata
		action, exists := c.Get("audit_action")
		if !exists {
			return
		}

		// Only record successful operations
		if c.Writer.Status() >= 300 {
			return
		}

		actionStr, _ := action.(string)
		resource, _ := c.Get("audit_resource")
		resourceStr, _ := resource.(string)
		resourceID, _ := c.Get("audit_resource_id")
		resourceIDStr, _ := resourceID.(string)
		summary, _ := c.Get("audit_summary")
		summaryStr, _ := summary.(string)

		userID := c.GetUint("userId")
		username, _ := c.Get("userName")
		usernameStr, _ := username.(string)

		entry := model.AuditLog{
			Category:   model.AuditCategoryOperation,
			Action:     actionStr,
			Resource:   resourceStr,
			ResourceID: resourceIDStr,
			Summary:    summaryStr,
			Level:      model.AuditLevelInfo,
			IPAddress:  c.ClientIP(),
			UserAgent:  c.GetHeader("User-Agent"),
		}

		if userID > 0 {
			entry.UserID = &userID
		}
		entry.Username = usernameStr

		auditSvc.Log(entry)
	}
}
