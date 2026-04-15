package itsm

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"metis/internal/handler"
)

type FormDefHandler struct {
	svc *FormDefService
}

func NewFormDefHandler(i do.Injector) (*FormDefHandler, error) {
	svc := do.MustInvoke[*FormDefService](i)
	return &FormDefHandler{svc: svc}, nil
}

type CreateFormDefRequest struct {
	Name        string `json:"name" binding:"required,max=128"`
	Code        string `json:"code" binding:"required,max=64"`
	Description string `json:"description" binding:"max=512"`
	Schema      string `json:"schema" binding:"required"`
	Scope       string `json:"scope" binding:"omitempty,oneof=global service"`
	ServiceID   *uint  `json:"serviceId"`
}

func (h *FormDefHandler) Create(c *gin.Context) {
	var req CreateFormDefRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Set("audit_action", "itsm.form.create")
	c.Set("audit_resource", "form_definition")

	fd := &FormDefinition{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Schema:      req.Schema,
		Scope:       req.Scope,
		ServiceID:   req.ServiceID,
	}

	result, err := h.svc.Create(fd)
	if err != nil {
		switch {
		case errors.Is(err, ErrFormDefCodeExists):
			handler.Fail(c, http.StatusConflict, err.Error())
		case errors.Is(err, ErrFormSchemaInvalid):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_resource_id", strconv.Itoa(int(result.ID)))
	c.Set("audit_summary", "created form: "+result.Name)
	handler.OK(c, result.ToResponse())
}

func (h *FormDefHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var isActive *bool
	if v := c.Query("isActive"); v != "" {
		b := v == "true"
		isActive = &b
	}

	items, total, err := h.svc.List(FormDefListParams{
		Keyword:  c.Query("keyword"),
		IsActive: isActive,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]FormDefinitionResponse, len(items))
	for i, fd := range items {
		result[i] = fd.ToResponse()
	}
	handler.OK(c, gin.H{"items": result, "total": total})
}

func (h *FormDefHandler) Get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	fd, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, ErrFormDefNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	handler.OK(c, fd.ToResponse())
}

type UpdateFormDefRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=128"`
	Code        *string `json:"code" binding:"omitempty,max=64"`
	Description *string `json:"description" binding:"omitempty,max=512"`
	Schema      *string `json:"schema"`
	Scope       *string `json:"scope" binding:"omitempty,oneof=global service"`
	ServiceID   *uint   `json:"serviceId"`
	IsActive    *bool   `json:"isActive"`
}

func (h *FormDefHandler) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req UpdateFormDefRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Set("audit_action", "itsm.form.update")
	c.Set("audit_resource", "form_definition")
	c.Set("audit_resource_id", c.Param("id"))

	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Code != nil {
		updates["code"] = *req.Code
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Schema != nil {
		updates["schema"] = *req.Schema
	}
	if req.Scope != nil {
		updates["scope"] = *req.Scope
	}
	if req.ServiceID != nil {
		updates["service_id"] = *req.ServiceID
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	result, err := h.svc.Update(id, updates)
	if err != nil {
		switch {
		case errors.Is(err, ErrFormDefNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrFormDefCodeExists):
			handler.Fail(c, http.StatusConflict, err.Error())
		case errors.Is(err, ErrFormSchemaInvalid):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "updated form: "+result.Name)
	handler.OK(c, result.ToResponse())
}

func (h *FormDefHandler) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	c.Set("audit_action", "itsm.form.delete")
	c.Set("audit_resource", "form_definition")
	c.Set("audit_resource_id", c.Param("id"))

	if err := h.svc.Delete(id); err != nil {
		switch {
		case errors.Is(err, ErrFormDefNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrFormDefInUse):
			handler.Fail(c, http.StatusConflict, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "deleted form definition")
	handler.OK(c, nil)
}
