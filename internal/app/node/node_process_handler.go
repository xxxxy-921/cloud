package node

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"metis/internal/handler"
)

type BindProcessRequest struct {
	ProcessDefID uint            `json:"processDefId" binding:"required"`
	OverrideVars json.RawMessage `json:"overrideVars"`
}

type NodeProcessHandler struct {
	nodeProcessSvc *NodeProcessService
}

func NewNodeProcessHandler(i do.Injector) (*NodeProcessHandler, error) {
	return &NodeProcessHandler{
		nodeProcessSvc: do.MustInvoke[*NodeProcessService](i),
	}, nil
}

func (h *NodeProcessHandler) Bind(c *gin.Context) {
	nodeID, err := parseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid node id")
		return
	}

	var req BindProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Set("audit_action", "bind_process")
	c.Set("audit_resource", "node_process")
	c.Set("audit_resource_id", c.Param("id"))

	np, err := h.nodeProcessSvc.Bind(nodeID, req.ProcessDefID, JSONMap(req.OverrideVars))
	if err != nil {
		if errors.Is(err, ErrNodeNotFound) || errors.Is(err, ErrProcessDefNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, ErrNodeProcessExists) {
			handler.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_summary", "bound process to node")
	handler.OK(c, np.ToResponse())
}

func (h *NodeProcessHandler) List(c *gin.Context) {
	nodeID, err := parseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid node id")
		return
	}

	items, err := h.nodeProcessSvc.ListByNodeID(nodeID)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]NodeProcessResponse, len(items))
	for i, item := range items {
		resp := item.NodeProcess.ToResponse()
		resp.ProcessName = item.ProcessName
		resp.DisplayName = item.DisplayName
		result[i] = resp
	}

	handler.OK(c, result)
}

func (h *NodeProcessHandler) Unbind(c *gin.Context) {
	nodeID, err := parseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid node id")
		return
	}

	processIDStr := c.Param("processId")
	processDefID, err := strconv.ParseUint(processIDStr, 10, 64)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid process id")
		return
	}

	c.Set("audit_action", "unbind_process")
	c.Set("audit_resource", "node_process")
	c.Set("audit_resource_id", c.Param("id"))

	if err := h.nodeProcessSvc.Unbind(nodeID, uint(processDefID)); err != nil {
		if errors.Is(err, ErrNodeProcessNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_summary", "unbound process from node")
	handler.OK(c, nil)
}

func (h *NodeProcessHandler) Start(c *gin.Context) {
	nodeID, err := parseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid node id")
		return
	}

	processIDStr := c.Param("processId")
	processDefID, err := strconv.ParseUint(processIDStr, 10, 64)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid process id")
		return
	}

	c.Set("audit_action", "start_process")
	c.Set("audit_resource", "node_process")
	c.Set("audit_resource_id", c.Param("id"))

	if err := h.nodeProcessSvc.Start(nodeID, uint(processDefID)); err != nil {
		if errors.Is(err, ErrNodeProcessNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_summary", "started process on node")
	handler.OK(c, nil)
}

func (h *NodeProcessHandler) Stop(c *gin.Context) {
	nodeID, err := parseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid node id")
		return
	}

	processIDStr := c.Param("processId")
	processDefID, err := strconv.ParseUint(processIDStr, 10, 64)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid process id")
		return
	}

	c.Set("audit_action", "stop_process")
	c.Set("audit_resource", "node_process")
	c.Set("audit_resource_id", c.Param("id"))

	if err := h.nodeProcessSvc.Stop(nodeID, uint(processDefID)); err != nil {
		if errors.Is(err, ErrNodeProcessNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_summary", "stopped process on node")
	handler.OK(c, nil)
}

func (h *NodeProcessHandler) Restart(c *gin.Context) {
	nodeID, err := parseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid node id")
		return
	}

	processIDStr := c.Param("processId")
	processDefID, err := strconv.ParseUint(processIDStr, 10, 64)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid process id")
		return
	}

	c.Set("audit_action", "restart_process")
	c.Set("audit_resource", "node_process")
	c.Set("audit_resource_id", c.Param("id"))

	if err := h.nodeProcessSvc.Restart(nodeID, uint(processDefID)); err != nil {
		if errors.Is(err, ErrNodeProcessNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_summary", "restarted process on node")
	handler.OK(c, nil)
}
