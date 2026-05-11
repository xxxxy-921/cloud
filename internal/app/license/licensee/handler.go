package licensee

import (
	"errors"
	"metis/internal/app/license/domain"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"metis/internal/handler"
)

// --- domain.Licensee request types ---

type CreateLicenseeRequest struct {
	Name  string `json:"name" binding:"required,max=128"`
	Notes string `json:"notes"`
}

type UpdateLicenseeRequest struct {
	Name  *string `json:"name" binding:"omitempty,max=128"`
	Notes *string `json:"notes"`
}

type UpdateLicenseeStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// --- LicenseeHandler ---

type LicenseeHandler struct {
	svc *LicenseeService
}

func NewLicenseeHandler(i do.Injector) (*LicenseeHandler, error) {
	return &LicenseeHandler{
		svc: do.MustInvoke[*LicenseeService](i),
	}, nil
}

func (h *LicenseeHandler) Create(c *gin.Context) {
	var req CreateLicenseeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Set("audit_action", "licensee.create")
	c.Set("audit_resource", "licensee")

	licensee, err := h.svc.CreateLicensee(CreateLicenseeParams{
		Name:  req.Name,
		Notes: req.Notes,
	})
	if err != nil {
		if errors.Is(err, ErrLicenseeNameExists) || errors.Is(err, ErrInvalidLicenseeName) {
			handler.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_resource_id", strconv.Itoa(int(licensee.ID)))
	c.Set("audit_summary", "created licensee: "+licensee.Name)
	handler.OK(c, licensee.ToResponse())
}

func (h *LicenseeHandler) List(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid page")
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid pageSize")
		return
	}

	params := LicenseeListParams{
		Keyword:  c.Query("keyword"),
		Status:   normalizeLicenseeListStatus(c.Query("status")),
		Page:     page,
		PageSize: pageSize,
	}
	if !isValidLicenseeListStatus(params.Status) {
		handler.Fail(c, http.StatusBadRequest, "invalid status")
		return
	}

	items, total, err := h.svc.ListLicensees(params)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]domain.LicenseeResponse, len(items))
	for i, item := range items {
		result[i] = item.ToResponse()
	}

	handler.OK(c, gin.H{
		"items":    result,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func isValidLicenseeListStatus(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "", "all", domain.LicenseeStatusActive, domain.LicenseeStatusArchived:
		return true
	default:
		return false
	}
}

func normalizeLicenseeListStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "all" {
		return ""
	}
	return status
}

func (h *LicenseeHandler) Get(c *gin.Context) {
	id, err := domain.ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	licensee, err := h.svc.GetLicensee(id)
	if err != nil {
		if errors.Is(err, ErrLicenseeNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	handler.OK(c, licensee.ToResponse())
}

func (h *LicenseeHandler) Update(c *gin.Context) {
	id, err := domain.ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req UpdateLicenseeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Set("audit_action", "licensee.update")
	c.Set("audit_resource", "licensee")
	c.Set("audit_resource_id", c.Param("id"))

	licensee, err := h.svc.UpdateLicensee(id, UpdateLicenseeParams{
		Name:  req.Name,
		Notes: req.Notes,
	})
	if err != nil {
		if errors.Is(err, ErrLicenseeNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, ErrLicenseeNameExists) || errors.Is(err, ErrInvalidLicenseeName) {
			handler.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_summary", "updated licensee: "+licensee.Name)
	handler.OK(c, licensee.ToResponse())
}

func (h *LicenseeHandler) UpdateStatus(c *gin.Context) {
	id, err := domain.ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req UpdateLicenseeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	action := "licensee.archive"
	if req.Status == domain.LicenseeStatusActive {
		action = "licensee.unarchive"
	}
	c.Set("audit_action", action)
	c.Set("audit_resource", "licensee")
	c.Set("audit_resource_id", c.Param("id"))

	if err := h.svc.UpdateLicenseeStatus(id, req.Status); err != nil {
		if errors.Is(err, ErrLicenseeNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, ErrLicenseeInvalidStatus) {
			handler.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_summary", "changed licensee status to "+req.Status)
	handler.OK(c, nil)
}
