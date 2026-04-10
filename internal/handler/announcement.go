package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"metis/internal/repository"
	"metis/internal/service"
)

type AnnouncementHandler struct {
	notifSvc *service.NotificationService
}

func (h *AnnouncementHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")

	items, total, err := h.notifSvc.ListAnnouncements(repository.ListParams{
		Keyword:  keyword,
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

type createAnnouncementReq struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content"`
}

func (h *AnnouncementHandler) Create(c *gin.Context) {
	var req createAnnouncementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	createdBy := c.GetUint("userId")
	n, err := h.notifSvc.CreateAnnouncement(req.Title, req.Content, createdBy)
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "announcement.create")
	c.Set("audit_resource", "announcement")
	c.Set("audit_resource_id", strconv.FormatUint(uint64(n.ID), 10))
	c.Set("audit_summary", "创建公告")
	OK(c, gin.H{"id": n.ID})
}

type updateAnnouncementReq struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content"`
}

func (h *AnnouncementHandler) Update(c *gin.Context) {
	id, err := parseIDParam(c)
	if err != nil {
		return
	}

	var req updateAnnouncementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	n, err := h.notifSvc.UpdateAnnouncement(id, req.Title, req.Content)
	if err != nil {
		if errors.Is(err, service.ErrNotificationNotFound) {
			Fail(c, http.StatusNotFound, "announcement not found")
			return
		}
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "announcement.update")
	c.Set("audit_resource", "announcement")
	c.Set("audit_resource_id", strconv.FormatUint(uint64(id), 10))
	c.Set("audit_summary", "更新公告")
	OK(c, gin.H{"id": n.ID})
}

func (h *AnnouncementHandler) Delete(c *gin.Context) {
	id, err := parseIDParam(c)
	if err != nil {
		return
	}

	if err := h.notifSvc.DeleteAnnouncement(id); err != nil {
		if errors.Is(err, service.ErrNotificationNotFound) {
			Fail(c, http.StatusNotFound, "announcement not found")
			return
		}
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "announcement.delete")
	c.Set("audit_resource", "announcement")
	c.Set("audit_resource_id", strconv.FormatUint(uint64(id), 10))
	c.Set("audit_summary", "删除公告")
	OK(c, nil)
}
