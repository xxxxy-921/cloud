package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"metis/internal/service"
)

type CaptchaHandler struct {
	captchaSvc  *service.CaptchaService
	settingsSvc *service.SettingsService
}

func (h *CaptchaHandler) Generate(c *gin.Context) {
	if h.settingsSvc.GetCaptchaProvider() == "none" {
		OK(c, gin.H{"enabled": false})
		return
	}

	result, err := h.captchaSvc.Generate()
	if err != nil {
		Fail(c, http.StatusInternalServerError, "failed to generate captcha")
		return
	}

	OK(c, gin.H{
		"enabled": true,
		"id":      result.ID,
		"image":   result.Image,
	})
}
