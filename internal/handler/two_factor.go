package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"metis/internal/pkg/token"
	"metis/internal/service"
)

type TwoFactorHandler struct {
	tfSvc     *service.TwoFactorService
	authSvc   *service.AuthService
	jwtSecret []byte
}

// Setup initiates 2FA setup for the current user.
func (h *TwoFactorHandler) Setup(c *gin.Context) {
	userID := c.GetUint("userId")
	result, err := h.tfSvc.Setup(userID)
	if err != nil {
		if errors.Is(err, service.ErrTwoFactorAlreadyEnabled) {
			Fail(c, http.StatusBadRequest, "2FA already enabled")
			return
		}
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	OK(c, result)
}

type confirmReq struct {
	Code string `json:"code" binding:"required"`
}

// Confirm verifies the TOTP code and enables 2FA.
func (h *TwoFactorHandler) Confirm(c *gin.Context) {
	userID := c.GetUint("userId")
	var req confirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.tfSvc.Confirm(userID, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTwoFactorNotSetup):
			Fail(c, http.StatusBadRequest, "2FA not set up, call setup first")
		case errors.Is(err, service.ErrTwoFactorInvalidCode):
			Fail(c, http.StatusBadRequest, "invalid verification code")
		default:
			Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_action", "2fa.enable")
	c.Set("audit_resource", "user")
	c.Set("audit_summary", "启用两步验证")
	OK(c, result)
}

// Disable removes 2FA for the current user.
func (h *TwoFactorHandler) Disable(c *gin.Context) {
	userID := c.GetUint("userId")
	if err := h.tfSvc.Disable(userID); err != nil {
		if errors.Is(err, service.ErrTwoFactorNotSetup) {
			Fail(c, http.StatusBadRequest, "2FA not enabled")
			return
		}
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "2fa.disable")
	c.Set("audit_resource", "user")
	c.Set("audit_summary", "关闭两步验证")
	OK(c, nil)
}

type twoFactorLoginReq struct {
	TwoFactorToken string `json:"twoFactorToken" binding:"required"`
	Code           string `json:"code" binding:"required"`
}

// Login completes authentication with a 2FA code after receiving a twoFactorToken.
func (h *TwoFactorHandler) Login(c *gin.Context) {
	var req twoFactorLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// Parse and validate the 2FA token
	claims, err := token.ParseToken(req.TwoFactorToken, h.jwtSecret)
	if err != nil {
		Fail(c, http.StatusUnauthorized, "invalid or expired 2FA token")
		return
	}
	if claims.Purpose != "2fa" {
		Fail(c, http.StatusUnauthorized, "invalid token purpose")
		return
	}

	// Verify the TOTP/backup code
	valid, err := h.tfSvc.Verify(claims.UserID, req.Code)
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if !valid {
		Fail(c, http.StatusUnauthorized, "invalid 2FA code")
		return
	}

	// Issue full token pair
	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	pair, err := h.authSvc.GenerateTokenPairByID(claims.UserID, ip, ua)
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	OK(c, pair)
}
