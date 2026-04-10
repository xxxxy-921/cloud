package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"metis/internal/model"
	"metis/internal/pkg/identity"
	"metis/internal/repository"
	"metis/internal/service"
)

// SSOHandler handles public SSO endpoints (check-domain, initiate, callback).
type SSOHandler struct {
	svc      *service.IdentitySourceService
	authSvc  *service.AuthService
	userRepo *repository.UserRepo
	connRepo *repository.UserConnectionRepo
	roleRepo *repository.RoleRepo
	stateMgr *identity.SSOStateManager
}

// CheckDomain checks if an email domain is bound to an identity source.
// GET /api/v1/auth/check-domain?email=user@acme.com
func (h *SSOHandler) CheckDomain(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		Fail(c, http.StatusBadRequest, "email is required")
		return
	}

	domain := service.ExtractDomain(email)
	if domain == "" {
		Fail(c, http.StatusBadRequest, "invalid email format")
		return
	}

	source, err := h.svc.FindByDomain(domain)
	if err != nil {
		Fail(c, http.StatusNotFound, "no identity source for this domain")
		return
	}

	OK(c, gin.H{
		"id":       source.ID,
		"name":     source.Name,
		"type":     source.Type,
		"forceSso": source.ForceSso,
	})
}

// InitiateSSO starts the OIDC SSO flow for the given identity source.
// GET /api/v1/auth/sso/:id/authorize
func (h *SSOHandler) InitiateSSO(c *gin.Context) {
	id, err := parseIdentityID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	source, cfg, err := h.svc.GetDecryptedConfig(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Fail(c, http.StatusNotFound, "identity source not found")
		} else {
			Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	if !source.Enabled {
		Fail(c, http.StatusBadRequest, "identity source not available")
		return
	}

	if source.Type != "oidc" {
		Fail(c, http.StatusBadRequest, "SSO initiation is only supported for OIDC sources")
		return
	}

	oidcCfg, ok := cfg.(*model.OIDCConfig)
	if !ok || oidcCfg == nil {
		Fail(c, http.StatusInternalServerError, "invalid OIDC config")
		return
	}

	ctx := context.Background()
	provider, err := identity.GetOIDCProvider(ctx, source.ID, oidcCfg)
	if err != nil {
		Fail(c, http.StatusBadGateway, "OIDC discovery failed: "+err.Error())
		return
	}

	var pkce *identity.PKCEParams
	codeVerifier := ""
	if oidcCfg.UsePKCE {
		pkce, err = identity.GeneratePKCE()
		if err != nil {
			Fail(c, http.StatusInternalServerError, "PKCE generation failed")
			return
		}
		codeVerifier = pkce.Verifier
	}

	state, err := h.stateMgr.Generate(source.ID, codeVerifier)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "state generation failed")
		return
	}

	authURL := provider.AuthURL(state, pkce)

	OK(c, gin.H{
		"authUrl": authURL,
		"state":   state,
	})
}

type ssoCallbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// SSOCallback handles the OIDC callback after user authentication.
// POST /api/v1/auth/sso/callback
func (h *SSOHandler) SSOCallback(c *gin.Context) {
	var req ssoCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	stateMeta, err := h.stateMgr.Validate(req.State)
	if err != nil {
		Fail(c, http.StatusBadRequest, "invalid or expired state")
		return
	}

	source, cfg, err := h.svc.GetDecryptedConfig(stateMeta.SourceID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "identity source not found")
		return
	}

	if source.Type != "oidc" {
		Fail(c, http.StatusBadRequest, "invalid source type for SSO callback")
		return
	}

	oidcCfg, ok := cfg.(*model.OIDCConfig)
	if !ok {
		Fail(c, http.StatusInternalServerError, "invalid OIDC config")
		return
	}

	ctx := context.Background()

	provider, err := identity.GetOIDCProvider(ctx, source.ID, oidcCfg)
	if err != nil {
		Fail(c, http.StatusBadGateway, "OIDC discovery failed")
		return
	}

	token, err := provider.ExchangeCode(ctx, req.Code, stateMeta.CodeVerifier)
	if err != nil {
		Fail(c, http.StatusBadGateway, "code exchange failed: "+err.Error())
		return
	}

	idToken, err := provider.VerifyIDToken(ctx, token)
	if err != nil {
		Fail(c, http.StatusBadGateway, "ID token verification failed: "+err.Error())
		return
	}

	claims, err := identity.ExtractClaims(idToken)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "failed to extract claims")
		return
	}

	providerName := fmt.Sprintf("oidc_%d", source.ID)
	user, err := h.jitProvision(source, providerName, claims.Sub, claims.Email, claims.Name, claims.Picture)
	if err != nil {
		if errors.Is(err, service.ErrEmailConflict) {
			Fail(c, http.StatusConflict, err.Error())
		} else {
			slog.Error("SSO JIT provision failed", "error", err)
			Fail(c, http.StatusInternalServerError, "user provisioning failed")
		}
		return
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	tokenPair, err := h.authSvc.GenerateTokenPair(user, ip, ua)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "token generation failed")
		return
	}

	OK(c, tokenPair)
}

// jitProvision handles JIT user provisioning for SSO flows.
func (h *SSOHandler) jitProvision(source *model.IdentitySource, providerName, externalID, email, name, avatar string) (*model.User, error) {
	conn, err := h.connRepo.FindByProviderAndExternalID(providerName, externalID)
	if err == nil {
		user, err := h.userRepo.FindByID(conn.UserID)
		if err != nil {
			return nil, err
		}
		if !user.IsActive {
			return nil, service.ErrAccountDisabled
		}

		changed := false
		if conn.ExternalName != name {
			conn.ExternalName = name
			changed = true
		}
		if conn.ExternalEmail != email {
			conn.ExternalEmail = email
			changed = true
		}
		if conn.AvatarURL != avatar {
			conn.AvatarURL = avatar
			changed = true
		}
		if changed {
			_ = h.connRepo.Update(conn)
		}
		return user, nil
	}

	if email != "" {
		existing, err := h.userRepo.FindByEmail(email)
		if err == nil && existing != nil {
			if source.ConflictStrategy == "link" {
				newConn := &model.UserConnection{
					UserID:        existing.ID,
					Provider:      providerName,
					ExternalID:    externalID,
					ExternalName:  name,
					ExternalEmail: email,
					AvatarURL:     avatar,
				}
				if err := h.connRepo.Create(newConn); err != nil {
					return nil, fmt.Errorf("create connection: %w", err)
				}
				return existing, nil
			}
			return nil, service.ErrEmailConflict
		}
	}

	roleID := source.DefaultRoleID
	if roleID == 0 {
		role, err := h.roleRepo.FindByCode(model.RoleUser)
		if err != nil {
			return nil, fmt.Errorf("find default role: %w", err)
		}
		roleID = role.ID
	}

	username := fmt.Sprintf("%s_%s", providerName, externalID)
	if name != "" {
		username = name
	}
	if _, err := h.userRepo.FindByUsername(username); err == nil {
		username = fmt.Sprintf("%s_%s", providerName, externalID)
	}

	user := &model.User{
		Username: username,
		Email:    email,
		Avatar:   avatar,
		RoleID:   roleID,
		IsActive: true,
	}
	if err := h.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	user, err = h.userRepo.FindByID(user.ID)
	if err != nil {
		return nil, err
	}

	newConn := &model.UserConnection{
		UserID:        user.ID,
		Provider:      providerName,
		ExternalID:    externalID,
		ExternalName:  name,
		ExternalEmail: email,
		AvatarURL:     avatar,
	}
	if err := h.connRepo.Create(newConn); err != nil {
		return nil, fmt.Errorf("create connection: %w", err)
	}

	return user, nil
}

func extractDomain(email string) string {
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return ""
	}
	return strings.ToLower(email[at+1:])
}
