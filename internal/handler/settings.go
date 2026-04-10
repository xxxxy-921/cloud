package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"metis/internal/service"
)

func (h *Handler) GetSecuritySettings(c *gin.Context) {
	settings := h.settingsSvc.GetSecuritySettings()
	OK(c, settings)
}

func (h *Handler) UpdateSecuritySettings(c *gin.Context) {
	var req service.SecuritySettings
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.MaxConcurrentSessions < 0 {
		Fail(c, http.StatusBadRequest, "maxConcurrentSessions must be >= 0")
		return
	}
	if err := h.settingsSvc.UpdateSecuritySettings(req); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Set("audit_action", "settings.update")
	c.Set("audit_resource", "settings")
	c.Set("audit_summary", "更新安全设置")
	OK(c, req)
}

func (h *Handler) GetSchedulerSettings(c *gin.Context) {
	settings := h.settingsSvc.GetSchedulerSettings()
	OK(c, settings)
}

func (h *Handler) UpdateSchedulerSettings(c *gin.Context) {
	var req service.SchedulerSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.HistoryRetentionDays < 0 {
		Fail(c, http.StatusBadRequest, "historyRetentionDays must be >= 0")
		return
	}
	if req.AuditRetentionDaysAuth < 0 {
		Fail(c, http.StatusBadRequest, "auditRetentionDaysAuth must be >= 0")
		return
	}
	if req.AuditRetentionDaysOperation < 0 {
		Fail(c, http.StatusBadRequest, "auditRetentionDaysOperation must be >= 0")
		return
	}
	if err := h.settingsSvc.UpdateSchedulerSettings(req); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Set("audit_action", "settings.update")
	c.Set("audit_resource", "settings")
	c.Set("audit_summary", "更新自动清理设置")
	OK(c, req)
}
