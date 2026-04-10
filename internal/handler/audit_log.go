package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"metis/internal/model"
	"metis/internal/repository"
	"metis/internal/service"
)

type AuditLogHandler struct {
	auditSvc *service.AuditLogService
}

func (h *AuditLogHandler) List(c *gin.Context) {
	category := c.Query("category")
	if category == "" {
		Fail(c, http.StatusBadRequest, "category is required")
		return
	}

	// Validate category
	cat := model.AuditCategory(category)
	switch cat {
	case model.AuditCategoryAuth, model.AuditCategoryOperation, model.AuditCategoryApplication:
		// valid
	default:
		Fail(c, http.StatusBadRequest, "invalid category")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	params := repository.AuditLogListParams{
		Category: cat,
		Keyword:  c.Query("keyword"),
		Action:   c.Query("action"),
		Resource: c.Query("resource"),
		Page:     page,
		PageSize: pageSize,
	}

	if df := c.Query("dateFrom"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			params.DateFrom = &t
		}
	}
	if dt := c.Query("dateTo"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			end := t.Add(24*time.Hour - time.Nanosecond) // end of day
			params.DateTo = &end
		}
	}

	result, err := h.auditSvc.List(params)
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]model.AuditLogResponse, len(result.Items))
	for i, item := range result.Items {
		items[i] = item.ToResponse()
	}

	OK(c, gin.H{
		"items":    items,
		"total":    result.Total,
		"page":     page,
		"pageSize": pageSize,
	})
}
