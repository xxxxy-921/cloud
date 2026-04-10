package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"metis/internal/service"
)

type AuthProviderHandler struct {
	svc *service.AuthProviderService
}

func (h *AuthProviderHandler) ListAll(c *gin.Context) {
	providers, err := h.svc.ListAll()
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	var resp []any
	for _, p := range providers {
		resp = append(resp, p.ToResponse())
	}
	if resp == nil {
		resp = []any{}
	}
	OK(c, resp)
}

func (h *AuthProviderHandler) Update(c *gin.Context) {
	key := c.Param("key")

	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := h.svc.Update(key, body)
	if err != nil {
		Fail(c, http.StatusNotFound, "provider not found")
		return
	}

	c.Set("audit_action", "auth_provider.update")
	c.Set("audit_resource", "auth_provider")
	c.Set("audit_resource_id", key)
	c.Set("audit_summary", "更新认证源配置")
	OK(c, p.ToResponse())
}

func (h *AuthProviderHandler) Toggle(c *gin.Context) {
	key := c.Param("key")

	p, err := h.svc.Toggle(key)
	if err != nil {
		Fail(c, http.StatusNotFound, "provider not found")
		return
	}

	c.Set("audit_action", "auth_provider.toggle")
	c.Set("audit_resource", "auth_provider")
	c.Set("audit_resource_id", key)
	c.Set("audit_summary", "切换认证源状态")
	OK(c, p.ToResponse())
}
