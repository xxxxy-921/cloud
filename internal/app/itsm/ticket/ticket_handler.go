package ticket

import (
	"encoding/json"
	"errors"
	"io"
	itsmdef "metis/internal/app/itsm/definition"
	. "metis/internal/app/itsm/domain"
	itsmsla "metis/internal/app/itsm/sla"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"metis/internal/app/itsm/engine"
	"metis/internal/handler"
)

type TicketHandler struct {
	svc         *TicketService
	timelineSvc *TimelineService
}

func NewTicketHandler(i do.Injector) (*TicketHandler, error) {
	svc := do.MustInvoke[*TicketService](i)
	timelineSvc := do.MustInvoke[*TimelineService](i)
	return &TicketHandler{svc: svc, timelineSvc: timelineSvc}, nil
}

func currentUserID(c *gin.Context) uint {
	userID, ok := c.Get("userId")
	if !ok {
		return 0
	}
	uid, _ := userID.(uint)
	return uid
}

func currentUserRole(c *gin.Context) string {
	role, _ := c.Get("userRole")
	roleCode, _ := role.(string)
	return roleCode
}

func respondTicketAccessError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrTicketNotFound):
		handler.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrTicketForbidden):
		handler.Fail(c, http.StatusForbidden, err.Error())
	default:
		handler.Fail(c, http.StatusInternalServerError, err.Error())
	}
}

func (h *TicketHandler) respondTicket(c *gin.Context, ticket *Ticket) {
	resp, err := h.svc.BuildResponse(ticket, currentUserID(c))
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	handler.OK(c, resp)
}

func (h *TicketHandler) respondTicketList(c *gin.Context, items []Ticket, total int64) {
	result, err := h.svc.BuildResponses(items, currentUserID(c))
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	handler.OK(c, gin.H{"items": result, "total": total})
}

type CreateTicketRequest struct {
	Title       string          `json:"title" binding:"required"`
	Description string          `json:"description"`
	ServiceID   uint            `json:"serviceId" binding:"required"`
	PriorityID  uint            `json:"priorityId" binding:"required"`
	FormData    json.RawMessage `json:"formData"`
}

func (h *TicketHandler) Create(c *gin.Context) {
	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.FormData) == 0 {
		req.FormData = json.RawMessage(`{}`)
	}

	operatorID := currentUserID(c)
	c.Set("audit_action", "itsm.ticket.create")
	c.Set("audit_resource", "ticket")

	ticket, err := h.svc.CreateCatalog(CreateTicketInput{
		Title:       req.Title,
		Description: req.Description,
		ServiceID:   req.ServiceID,
		PriorityID:  req.PriorityID,
		FormData:    JSONField(req.FormData),
		Source:      TicketSourceCatalog,
	}, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, itsmdef.ErrServiceDefNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrCatalogSubmissionClassic), errors.Is(err, ErrServiceNotActive),
			errors.Is(err, itsmsla.ErrPriorityNotFound), errors.Is(err, itsmsla.ErrSLATemplateNotFound):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_resource_id", strconv.FormatUint(uint64(ticket.ID), 10))
	c.Set("audit_summary", "created ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

func (h *TicketHandler) buildTimelineResponses(items []TicketTimeline) ([]TicketTimelineResponse, error) {
	result := make([]TicketTimelineResponse, len(items))
	userIDs := make(map[uint]struct{})
	for i, t := range items {
		result[i] = t.ToResponse()
		if t.OperatorID > 0 {
			userIDs[t.OperatorID] = struct{}{}
		}
	}
	userNames := make(map[uint]string)
	if ids := keysOf(userIDs); len(ids) > 0 {
		var rows []struct {
			ID       uint
			Username string
		}
		if err := h.svc.ticketRepo.DB().Table("users").Where("id IN ?", ids).Select("id, username").Scan(&rows).Error; err != nil {
			return result, err
		}
		for _, r := range rows {
			userNames[r.ID] = r.Username
		}
	}
	for i := range result {
		if result[i].OperatorID == 0 {
			result[i].OperatorName = "系统"
			continue
		}
		result[i].OperatorName = userNames[result[i].OperatorID]
	}
	return result, nil
}

func (h *TicketHandler) List(c *gin.Context) {
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

	params := TicketListParams{
		Keyword:    c.Query("keyword"),
		Status:     c.Query("status"),
		EngineType: c.Query("engineType"),
		Page:       page,
		PageSize:   pageSize,
	}
	var ok bool
	params.Status, ok = normalizeTicketQueryStatus(params.Status)
	if !ok {
		handler.Fail(c, http.StatusBadRequest, "invalid status")
		return
	}
	params.EngineType, ok = normalizeTicketQueryEngineType(params.EngineType)
	if !ok {
		handler.Fail(c, http.StatusBadRequest, "invalid engineType")
		return
	}

	if v := c.Query("priorityId"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			handler.Fail(c, http.StatusBadRequest, "invalid priorityId")
			return
		}
		uid := uint(id)
		params.PriorityID = &uid
	}
	if v := c.Query("serviceId"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			handler.Fail(c, http.StatusBadRequest, "invalid serviceId")
			return
		}
		uid := uint(id)
		params.ServiceID = &uid
	}
	if v := c.Query("assigneeId"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			handler.Fail(c, http.StatusBadRequest, "invalid assigneeId")
			return
		}
		uid := uint(id)
		params.AssigneeID = &uid
	}
	if v := c.Query("requesterId"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			handler.Fail(c, http.StatusBadRequest, "invalid requesterId")
			return
		}
		uid := uint(id)
		params.RequesterID = &uid
	}

	items, total, err := h.svc.List(params)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.respondTicketList(c, items, total)
}

func (h *TicketHandler) Monitor(c *gin.Context) {
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

	params := TicketMonitorParams{
		Keyword:    c.Query("keyword"),
		Status:     c.Query("status"),
		EngineType: c.Query("engineType"),
		RiskLevel:  c.Query("riskLevel"),
		MetricCode: c.Query("metricCode"),
		Page:       page,
		PageSize:   pageSize,
		OperatorID: currentUserID(c),
	}
	var ok bool
	params.Status, ok = normalizeTicketQueryStatus(params.Status)
	if !ok {
		handler.Fail(c, http.StatusBadRequest, "invalid status")
		return
	}
	params.EngineType, ok = normalizeTicketQueryEngineType(params.EngineType)
	if !ok {
		handler.Fail(c, http.StatusBadRequest, "invalid engineType")
		return
	}
	if scope, ok := c.Get("deptScope"); ok {
		if deptScope, ok := scope.(*[]uint); ok {
			params.DeptScope = deptScope
		}
	}

	if v := c.Query("priorityId"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			handler.Fail(c, http.StatusBadRequest, "invalid priorityId")
			return
		}
		uid := uint(id)
		params.PriorityID = &uid
	}
	if v := c.Query("serviceId"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			handler.Fail(c, http.StatusBadRequest, "invalid serviceId")
			return
		}
		uid := uint(id)
		params.ServiceID = &uid
	}
	if !isValidTicketMonitorRiskLevel(params.RiskLevel) {
		handler.Fail(c, http.StatusBadRequest, "invalid riskLevel")
		return
	}
	if !isValidTicketMonitorMetricCode(params.MetricCode) {
		handler.Fail(c, http.StatusBadRequest, "invalid metricCode")
		return
	}

	resp, err := h.svc.Monitor(params, currentUserID(c))
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	handler.OK(c, resp)
}

func isValidTicketMonitorRiskLevel(riskLevel string) bool {
	switch strings.TrimSpace(riskLevel) {
	case "", "all", "stuck", "blocked", "risk":
		return true
	default:
		return false
	}
}

func isValidTicketMonitorMetricCode(metricCode string) bool {
	switch strings.TrimSpace(metricCode) {
	case "", "all",
		"active_total",
		"blocked_total",
		"stuck_total",
		"risk_total",
		"sla_risk_total",
		"ai_incident_total",
		"completed_today_total",
		"smart_active_total",
		"classic_active_total":
		return true
	default:
		return false
	}
}

func (h *TicketHandler) DecisionQuality(c *gin.Context) {
	windowDays, err := strconv.Atoi(c.DefaultQuery("windowDays", "30"))
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid windowDays")
		return
	}
	if windowDays <= 0 || windowDays > 180 {
		handler.Fail(c, http.StatusBadRequest, "invalid windowDays")
		return
	}
	dimension := c.DefaultQuery("dimension", "service")
	if !isValidDecisionQualityDimension(dimension) {
		handler.Fail(c, http.StatusBadRequest, "invalid dimension")
		return
	}

	var serviceID *uint
	if v := c.Query("serviceId"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			handler.Fail(c, http.StatusBadRequest, "invalid serviceId")
			return
		}
		uid := uint(id)
		serviceID = &uid
	}
	var departmentID *uint
	if v := c.Query("departmentId"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			handler.Fail(c, http.StatusBadRequest, "invalid departmentId")
			return
		}
		uid := uint(id)
		departmentID = &uid
	}

	resp, err := h.svc.DecisionQuality(windowDays, dimension, serviceID, departmentID)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	handler.OK(c, resp)
}

func isValidDecisionQualityDimension(dimension string) bool {
	switch strings.TrimSpace(strings.ToLower(dimension)) {
	case "", "service", "department":
		return true
	default:
		return false
	}
}

func normalizeTicketQueryStatus(status string) (string, bool) {
	normalized := strings.TrimSpace(strings.ToLower(status))
	switch normalized {
	case "", "all",
		"active",
		"terminal",
		TicketStatusDecisioning,
		TicketStatusSubmitted,
		TicketStatusWaitingHuman,
		TicketStatusApprovedDecisioning,
		TicketStatusRejectedDecisioning,
		TicketStatusExecutingAction,
		TicketStatusCompleted,
		TicketStatusRejected,
		TicketStatusWithdrawn,
		TicketStatusCancelled,
		TicketStatusFailed:
		if normalized == "all" {
			return "", true
		}
		return normalized, true
	default:
		return "", false
	}
}

func normalizeTicketQueryEngineType(engineType string) (string, bool) {
	normalized := strings.TrimSpace(strings.ToLower(engineType))
	switch normalized {
	case "", "all", "classic", "smart", "manual":
		if normalized == "all" {
			return "", true
		}
		return normalized, true
	default:
		return "", false
	}
}

func (h *TicketHandler) Get(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	ticket, err := h.svc.GetVisible(id, currentUserID(c), currentUserRole(c))
	if err != nil {
		respondTicketAccessError(c, err)
		return
	}
	h.respondTicket(c, ticket)
}

type AssignTicketRequest struct {
	AssigneeID uint `json:"assigneeId" binding:"required"`
}

func (h *TicketHandler) Assign(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req AssignTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.assign")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.Assign(id, req.AssigneeID, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrInvalidActivityType):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNoActiveAssignment):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "assigned ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

func (h *TicketHandler) Cancel(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req CancelTicketInput
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.cancel")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.Cancel(id, req.Reason, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrInvalidActivityType):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNoActiveAssignment):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "cancelled ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

type WithdrawTicketInput struct {
	Reason string `json:"reason"`
}

func (h *TicketHandler) Withdraw(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req WithdrawTicketInput
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.withdraw")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.Withdraw(id, req.Reason, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNotRequester):
			handler.Fail(c, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrTicketClaimed):
			handler.Fail(c, http.StatusConflict, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "withdrew ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

func (h *TicketHandler) Mine(c *gin.Context) {
	userID, _ := c.Get("userId")
	requesterID := userID.(uint)

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
	keyword := c.Query("keyword")
	status := c.Query("status")
	status, ok := normalizeTicketQueryStatus(status)
	if !ok {
		handler.Fail(c, http.StatusBadRequest, "invalid status")
		return
	}

	var startDate *time.Time
	if v := c.Query("startDate"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			handler.Fail(c, http.StatusBadRequest, "invalid startDate")
			return
		}
		startDate = &t
	}
	var endDate *time.Time
	if v := c.Query("endDate"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			handler.Fail(c, http.StatusBadRequest, "invalid endDate")
			return
		}
		end := t.Add(24*time.Hour - time.Nanosecond)
		endDate = &end
	}

	items, total, err := h.svc.Mine(requesterID, keyword, status, startDate, endDate, page, pageSize)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.respondTicketList(c, items, total)
}

func (h *TicketHandler) PendingApprovals(c *gin.Context) {
	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

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
	items, total, err := h.svc.PendingApprovals(operatorID, c.Query("keyword"), page, pageSize)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.respondTicketList(c, items, total)
}

func (h *TicketHandler) ApprovalHistory(c *gin.Context) {
	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

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
	items, total, err := h.svc.ApprovalHistory(operatorID, c.Query("keyword"), page, pageSize)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.respondTicketList(c, items, total)
}

func (h *TicketHandler) Timeline(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.EnsureCanViewTicket(id, currentUserID(c), currentUserRole(c)); err != nil {
		respondTicketAccessError(c, err)
		return
	}

	items, err := h.timelineSvc.ListByTicket(id)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	result, err := h.buildTimelineResponses(items)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	handler.OK(c, result)
}

type ProgressTicketRequest struct {
	ActivityID uint            `json:"activityId" binding:"required"`
	Outcome    string          `json:"outcome" binding:"required"`
	Opinion    string          `json:"opinion"`
	Result     json.RawMessage `json:"result"`
}

func (h *TicketHandler) Progress(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req ProgressTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.progress")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.Progress(id, req.ActivityID, req.Outcome, req.Opinion, req.Result, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrInvalidProgressOutcome):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNoActiveAssignment):
			handler.Fail(c, http.StatusForbidden, err.Error())
		case errors.Is(err, engine.ErrActivityNotFound), errors.Is(err, engine.ErrActivityNotActive):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "progressed ticket: "+ticket.Code+" outcome="+req.Outcome)
	h.respondTicket(c, ticket)
}

type SignalTicketRequest struct {
	ActivityID uint            `json:"activityId" binding:"required"`
	Outcome    string          `json:"outcome" binding:"required"`
	Data       json.RawMessage `json:"data"`
}

func (h *TicketHandler) Signal(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req SignalTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.signal")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.Signal(id, req.ActivityID, req.Outcome, req.Data, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrActivityNotWait), errors.Is(err, engine.ErrActivityNotFound), errors.Is(err, engine.ErrActivityNotActive):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "signalled ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

func (h *TicketHandler) Activities(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	if err := h.svc.EnsureCanViewTicket(id, operatorID, currentUserRole(c)); err != nil {
		respondTicketAccessError(c, err)
		return
	}

	activities, err := h.svc.GetActivities(id, operatorID)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	handler.OK(c, activities)
}

// --- Smart engine override handlers ---

type OverrideJumpRequest struct {
	ActivityType string `json:"activityType" binding:"required"`
	AssigneeID   *uint  `json:"assigneeId"`
	Reason       string `json:"reason" binding:"required"`
}

func (h *TicketHandler) OverrideJump(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req OverrideJumpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.override_jump")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.OverrideJump(id, req.ActivityType, req.AssigneeID, req.Reason, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrInvalidActivityType):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNoActiveAssignment):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "override jump for ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

type OverrideReassignRequest struct {
	ActivityID    uint   `json:"activityId" binding:"required"`
	NewAssigneeID uint   `json:"newAssigneeId" binding:"required"`
	Reason        string `json:"reason" binding:"required"`
}

func (h *TicketHandler) OverrideReassign(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req OverrideReassignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.override_reassign")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.OverrideReassign(id, req.ActivityID, req.NewAssigneeID, req.Reason, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNoActiveAssignment):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "override reassign for ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

type RecoveryActionRequest struct {
	Action string `json:"action" binding:"required"`
	Reason string `json:"reason"`
}

func (h *TicketHandler) Recover(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req RecoveryActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.recovery")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.Recover(id, req.Action, req.Reason, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal), errors.Is(err, ErrInvalidRecoveryAction),
			errors.Is(err, ErrRetryAIOnlyForSmart), errors.Is(err, ErrHandoffHumanOnlyForSmart),
			errors.Is(err, ErrRecoveryOperatorRequired):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNotRequester):
			handler.Fail(c, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrTicketClaimed), errors.Is(err, ErrRecoveryActionTooFrequent):
			handler.Fail(c, http.StatusConflict, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "recovery action "+req.Action+" for ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

func (h *TicketHandler) RetryAI(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.retry_ai")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.RetryAI(id, req.Reason, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal), errors.Is(err, ErrRetryAIOnlyForSmart), errors.Is(err, ErrRecoveryOperatorRequired):
			handler.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrRecoveryActionTooFrequent):
			handler.Fail(c, http.StatusConflict, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "retry AI for ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

// SLAPause handles PUT /api/v1/itsm/tickets/:id/sla/pause
func (h *TicketHandler) SLAPause(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.sla_pause")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.SLAPause(id, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal):
			handler.Fail(c, http.StatusConflict, err.Error())
		case errors.Is(err, ErrSLAAlreadyPaused):
			handler.Fail(c, http.StatusConflict, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "paused SLA for ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

// SLAResume handles PUT /api/v1/itsm/tickets/:id/sla/resume
func (h *TicketHandler) SLAResume(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.sla_resume")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.SLAResume(id, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal):
			handler.Fail(c, http.StatusConflict, err.Error())
		case errors.Is(err, ErrSLANotPaused):
			handler.Fail(c, http.StatusConflict, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "resumed SLA for ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

// Transfer handles POST /api/v1/itsm/tickets/:id/transfer
func (h *TicketHandler) Transfer(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req TransferInput
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.transfer")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.Transfer(id, req.ActivityID, req.TargetUserID, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal):
			handler.Fail(c, http.StatusConflict, err.Error())
		case errors.Is(err, ErrNoActiveAssignment):
			handler.Fail(c, http.StatusForbidden, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "transferred task for ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

// Delegate handles POST /api/v1/itsm/tickets/:id/delegate
func (h *TicketHandler) Delegate(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req DelegateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.delegate")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.Delegate(id, req.ActivityID, req.TargetUserID, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal):
			handler.Fail(c, http.StatusConflict, err.Error())
		case errors.Is(err, ErrNoActiveAssignment):
			handler.Fail(c, http.StatusForbidden, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "delegated task for ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}

// Claim handles POST /api/v1/itsm/tickets/:id/claim
func (h *TicketHandler) Claim(c *gin.Context) {
	id, err := ParseID(c)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req ClaimInput
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("userId")
	operatorID := userID.(uint)

	c.Set("audit_action", "itsm.ticket.claim")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", c.Param("id"))

	ticket, err := h.svc.Claim(id, req.ActivityID, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTicketNotFound):
			handler.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketTerminal):
			handler.Fail(c, http.StatusConflict, err.Error())
		case errors.Is(err, ErrNoActiveAssignment):
			handler.Fail(c, http.StatusForbidden, err.Error())
		default:
			handler.Fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Set("audit_summary", "claimed task for ticket: "+ticket.Code)
	h.respondTicket(c, ticket)
}
