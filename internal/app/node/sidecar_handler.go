package node

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"metis/internal/handler"
)

type AckCommandRequest struct {
	Success bool   `json:"success"`
	Result  string `json:"result"`
}

type SidecarHandler struct {
	nodeRepo   *NodeRepo
	sidecarSvc *SidecarService
	hub        *NodeHub
}

func NewSidecarHandler(i do.Injector) (*SidecarHandler, error) {
	return &SidecarHandler{
		nodeRepo:   do.MustInvoke[*NodeRepo](i),
		sidecarSvc: do.MustInvoke[*SidecarService](i),
		hub:        do.MustInvoke[*NodeHub](i),
	}, nil
}

// TokenAuth returns a middleware that authenticates via Node Token.
func (h *SidecarHandler) TokenAuth() gin.HandlerFunc {
	return NodeTokenMiddleware(h.nodeRepo)
}

func (h *SidecarHandler) Register(c *gin.Context) {
	nodeID := GetNodeID(c)

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.sidecarSvc.Register(nodeID, req); err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	handler.OK(c, gin.H{"nodeId": nodeID})
}

func (h *SidecarHandler) Heartbeat(c *gin.Context) {
	nodeID := GetNodeID(c)

	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.sidecarSvc.Heartbeat(nodeID, req); err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	handler.OK(c, nil)
}

// Stream establishes an SSE connection for real-time command delivery.
func (h *SidecarHandler) Stream(c *gin.Context) {
	nodeID := GetNodeID(c)

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Register this connection with NodeHub
	conn := h.hub.Register(nodeID)
	defer h.hub.Unregister(nodeID)

	// Push any pending commands first
	pendingCmds, _ := h.sidecarSvc.PollCommands(nodeID)
	for _, cmd := range pendingCmds {
		data, _ := json.Marshal(cmd.ToResponse())
		c.SSEvent("command", json.RawMessage(data))
		c.Writer.Flush()
	}

	// Ping ticker for keepalive
	pingTicker := time.NewTicker(15 * time.Second)
	defer pingTicker.Stop()

	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			return
		case <-conn.DoneCh:
			return
		case event := <-conn.EventCh:
			data, _ := json.Marshal(event.Data)
			c.SSEvent(event.Event, json.RawMessage(data))
			c.Writer.Flush()
		case <-pingTicker.C:
			c.SSEvent("ping", "{}")
			c.Writer.Flush()
		}
	}
}

func (h *SidecarHandler) PollCommands(c *gin.Context) {
	nodeID := GetNodeID(c)

	// Fallback: return pending commands immediately (no long-poll)
	cmds, err := h.sidecarSvc.PollCommands(nodeID)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	result := make([]NodeCommandResponse, len(cmds))
	for i, cmd := range cmds {
		result[i] = cmd.ToResponse()
	}
	handler.OK(c, result)
}

func (h *SidecarHandler) AckCommand(c *gin.Context) {
	nodeID := GetNodeID(c)

	cmdIDStr := c.Param("id")
	cmdID, err := strconv.ParseUint(cmdIDStr, 10, 64)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid command id")
		return
	}

	var req AckCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.sidecarSvc.AckCommand(uint(cmdID), nodeID, req.Success, req.Result); err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	handler.OK(c, nil)
}

func (h *SidecarHandler) DownloadConfig(c *gin.Context) {
	nodeID := GetNodeID(c)
	processName := c.Param("name")
	filename := c.Query("file")

	rendered, hash, err := h.sidecarSvc.RenderConfig(nodeID, processName, filename)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Header("X-Config-Hash", hash)
	c.String(http.StatusOK, rendered)
}
