package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"metis/internal/repository"
	"metis/internal/service"
)

type ChannelHandler struct {
	svc *service.MessageChannelService
}

func (h *ChannelHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")

	items, total, err := h.svc.List(repository.ListParams{
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

func (h *ChannelHandler) Get(c *gin.Context) {
	id, err := parseIDParam(c)
	if err != nil {
		return
	}

	ch, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			Fail(c, http.StatusNotFound, "channel not found")
			return
		}
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	OK(c, ch)
}

type createChannelReq struct {
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type" binding:"required"`
	Config string `json:"config" binding:"required"`
}

func (h *ChannelHandler) Create(c *gin.Context) {
	var req createChannelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	ch, err := h.svc.Create(req.Name, req.Type, req.Config)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Set("audit_action", "channel.create")
	c.Set("audit_resource", "channel")
	c.Set("audit_resource_id", strconv.FormatUint(uint64(ch.ID), 10))
	c.Set("audit_summary", "创建消息渠道")
	OK(c, gin.H{"id": ch.ID})
}

type updateChannelReq struct {
	Name   string `json:"name" binding:"required"`
	Config string `json:"config" binding:"required"`
}

func (h *ChannelHandler) Update(c *gin.Context) {
	id, err := parseIDParam(c)
	if err != nil {
		return
	}

	var req updateChannelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	ch, err := h.svc.Update(id, req.Name, req.Config)
	if err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			Fail(c, http.StatusNotFound, "channel not found")
			return
		}
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "channel.update")
	c.Set("audit_resource", "channel")
	c.Set("audit_resource_id", strconv.FormatUint(uint64(id), 10))
	c.Set("audit_summary", "更新消息渠道")
	OK(c, ch)
}

func (h *ChannelHandler) Delete(c *gin.Context) {
	id, err := parseIDParam(c)
	if err != nil {
		return
	}

	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			Fail(c, http.StatusNotFound, "channel not found")
			return
		}
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "channel.delete")
	c.Set("audit_resource", "channel")
	c.Set("audit_resource_id", strconv.FormatUint(uint64(id), 10))
	c.Set("audit_summary", "删除消息渠道")
	OK(c, nil)
}

func (h *ChannelHandler) Toggle(c *gin.Context) {
	id, err := parseIDParam(c)
	if err != nil {
		return
	}

	ch, err := h.svc.ToggleEnabled(id)
	if err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			Fail(c, http.StatusNotFound, "channel not found")
			return
		}
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "channel.toggle")
	c.Set("audit_resource", "channel")
	c.Set("audit_resource_id", strconv.FormatUint(uint64(id), 10))
	c.Set("audit_summary", "切换消息渠道状态")
	OK(c, ch)
}

func (h *ChannelHandler) Test(c *gin.Context) {
	id, err := parseIDParam(c)
	if err != nil {
		return
	}

	if err := h.svc.TestChannel(id); err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			Fail(c, http.StatusNotFound, "channel not found")
			return
		}
		OK(c, gin.H{"success": false, "error": err.Error()})
		return
	}

	OK(c, gin.H{"success": true})
}

type sendTestReq struct {
	To      string `json:"to" binding:"required"`
	Subject string `json:"subject" binding:"required"`
	Body    string `json:"body" binding:"required"`
}

func (h *ChannelHandler) SendTest(c *gin.Context) {
	id, err := parseIDParam(c)
	if err != nil {
		return
	}

	var req sendTestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.SendTest(id, []string{req.To}, req.Subject, req.Body); err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			Fail(c, http.StatusNotFound, "channel not found")
			return
		}
		OK(c, gin.H{"success": false, "error": err.Error()})
		return
	}

	OK(c, gin.H{"success": true})
}
