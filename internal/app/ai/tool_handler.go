package ai

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"metis/internal/handler"
)

type ToolHandler struct {
	svc *ToolService
}

func NewToolHandler(i do.Injector) (*ToolHandler, error) {
	return &ToolHandler{
		svc: do.MustInvoke[*ToolService](i),
	}, nil
}

type toolkitGroup struct {
	Toolkit string         `json:"toolkit"`
	Tools   []ToolResponse `json:"tools"`
}

func (h *ToolHandler) List(c *gin.Context) {
	tools, err := h.svc.List()
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Group by toolkit, preserve insertion order
	orderMap := map[string]int{}
	var groups []toolkitGroup
	for _, t := range tools {
		resp := t.ToResponse()
		idx, ok := orderMap[t.Toolkit]
		if !ok {
			idx = len(groups)
			orderMap[t.Toolkit] = idx
			groups = append(groups, toolkitGroup{Toolkit: t.Toolkit})
		}
		groups[idx].Tools = append(groups[idx].Tools, resp)
	}
	handler.OK(c, gin.H{"items": groups})
}

type toggleToolReq struct {
	IsActive bool `json:"isActive"`
}

func (h *ToolHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req toggleToolReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	t, err := h.svc.ToggleActive(uint(id), req.IsActive)
	if err != nil {
		if errors.Is(err, ErrToolNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "update")
	c.Set("audit_resource", "ai_tool")
	c.Set("audit_resource_id", strconv.Itoa(int(t.ID)))
	c.Set("audit_summary", "Toggled tool: "+t.Name)

	handler.OK(c, t.ToResponse())
}
