package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"metis/internal/repository"
	"metis/internal/service"
)

type NotificationHandler struct {
	notifSvc *service.NotificationService
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID := c.GetUint("userId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	items, total, err := h.notifSvc.ListForUser(userID, repository.ListParams{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	OK(c, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := c.GetUint("userId")

	count, err := h.notifSvc.GetUnreadCount(userID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	OK(c, gin.H{"count": count})
}

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := parseIDParam(c)
	if err != nil {
		return
	}

	if err := h.notifSvc.MarkAsRead(id, userID); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	OK(c, nil)
}

func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.GetUint("userId")

	if err := h.notifSvc.MarkAllAsRead(userID); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	OK(c, nil)
}
