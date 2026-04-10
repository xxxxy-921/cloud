package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"metis/internal/model"
	"metis/internal/repository"
	"metis/internal/service"
)

// IdentitySourceHandler handles admin CRUD for identity sources.
type IdentitySourceHandler struct {
	svc *service.IdentitySourceService
}

func (h *IdentitySourceHandler) List(c *gin.Context) {
	sources, err := h.svc.List()
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	OK(c, sources)
}

type identitySourceCreateRequest struct {
	Name             string          `json:"name" binding:"required"`
	Type             string          `json:"type" binding:"required"`
	Domains          string          `json:"domains"`
	ForceSso         bool            `json:"forceSso"`
	DefaultRoleID    uint            `json:"defaultRoleId"`
	ConflictStrategy string          `json:"conflictStrategy"`
	Config           json.RawMessage `json:"config" binding:"required"`
	SortOrder        int             `json:"sortOrder"`
}

func (h *IdentitySourceHandler) Create(c *gin.Context) {
	var req identitySourceCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.ConflictStrategy == "" {
		req.ConflictStrategy = "fail"
	}

	source := &model.IdentitySource{
		Name:             req.Name,
		Type:             req.Type,
		Domains:          req.Domains,
		ForceSso:         req.ForceSso,
		DefaultRoleID:    req.DefaultRoleID,
		ConflictStrategy: req.ConflictStrategy,
		SortOrder:        req.SortOrder,
	}

	if err := h.svc.Create(source, req.Config); err != nil {
		switch err {
		case service.ErrUnsupportedType:
			Fail(c, http.StatusBadRequest, err.Error())
		case repository.ErrDomainConflict:
			Fail(c, http.StatusConflict, err.Error())
		default:
			Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_action", "identity_source.create")
	c.Set("audit_resource", "identity_source")
	c.Set("audit_resource_id", strconv.FormatUint(uint64(source.ID), 10))
	c.Set("audit_summary", "创建身份源: "+source.Name)

	OK(c, source.ToResponse())
}

type identitySourceUpdateRequest struct {
	Name             string          `json:"name" binding:"required"`
	Domains          string          `json:"domains"`
	ForceSso         bool            `json:"forceSso"`
	DefaultRoleID    uint            `json:"defaultRoleId"`
	ConflictStrategy string          `json:"conflictStrategy"`
	Config           json.RawMessage `json:"config" binding:"required"`
	SortOrder        int             `json:"sortOrder"`
}

func (h *IdentitySourceHandler) Update(c *gin.Context) {
	id, err := parseIdentityID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req identitySourceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	source := &model.IdentitySource{
		Name:             req.Name,
		Domains:          req.Domains,
		ForceSso:         req.ForceSso,
		DefaultRoleID:    req.DefaultRoleID,
		ConflictStrategy: req.ConflictStrategy,
		SortOrder:        req.SortOrder,
	}

	resp, err := h.svc.Update(id, source, req.Config)
	if err != nil {
		switch err {
		case service.ErrSourceNotFound:
			Fail(c, http.StatusNotFound, err.Error())
		case repository.ErrDomainConflict:
			Fail(c, http.StatusConflict, err.Error())
		default:
			Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_action", "identity_source.update")
	c.Set("audit_resource", "identity_source")
	c.Set("audit_resource_id", strconv.FormatUint(uint64(id), 10))
	c.Set("audit_summary", "更新身份源: "+req.Name)

	OK(c, resp)
}

func (h *IdentitySourceHandler) Delete(c *gin.Context) {
	id, err := parseIdentityID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		if err == service.ErrSourceNotFound {
			Fail(c, http.StatusNotFound, err.Error())
			return
		}
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "identity_source.delete")
	c.Set("audit_resource", "identity_source")
	c.Set("audit_resource_id", strconv.FormatUint(uint64(id), 10))
	c.Set("audit_summary", "删除身份源")

	OK(c, nil)
}

func (h *IdentitySourceHandler) Toggle(c *gin.Context) {
	id, err := parseIdentityID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	resp, err := h.svc.Toggle(id)
	if err != nil {
		if err == service.ErrSourceNotFound {
			Fail(c, http.StatusNotFound, err.Error())
			return
		}
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "identity_source.toggle")
	c.Set("audit_resource", "identity_source")
	c.Set("audit_resource_id", strconv.FormatUint(uint64(id), 10))

	OK(c, resp)
}

func (h *IdentitySourceHandler) TestConnection(c *gin.Context) {
	id, err := parseIdentityID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	success, msg := h.svc.TestConnection(id)
	OK(c, gin.H{"success": success, "message": msg})
}

func parseIdentityID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	return uint(id), err
}
