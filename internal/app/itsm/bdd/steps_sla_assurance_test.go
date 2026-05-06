package bdd

import (
	"context"
	"encoding/json"
	"fmt"
	"metis/internal/app"
	. "metis/internal/app/itsm/domain"
	"strings"
	"time"

	"github.com/cucumber/godog"

	ai "metis/internal/app/ai/runtime"
	"metis/internal/app/itsm/engine"
)

const bddSLAAssuranceAgentPrompt = `你是 SLA 保障岗，负责监督智能 ITSM 工单的 SLA 风险并在规则命中时触发升级动作。

操作必须按顺序执行：
1. 调用 sla.risk_queue 读取风险队列。
2. 对候选工单调用 sla.ticket_context 读取上下文。
3. 调用 sla.escalation_rules 读取已命中升级规则。
4. 规则允许时调用 sla.trigger_escalation 触发升级动作。

不得跳过工具调用直接回答。`

func registerSLAAssuranceSteps(sc *godog.ScenarioContext, bc *bddContext) {
	sc.Given(`^已发布带 SLA 的智能服务和 SLA 保障岗$`, bc.givenPublishedSLAServiceAndAssuranceAgent)
	sc.Given(`^存在响应 SLA 已超时且命中 "([^"]*)" 升级规则的工单$`, bc.givenResponseSLABreachedTicketWithRule)
	sc.Given(`^存在响应 SLA 尚未超时但配置 "([^"]*)" 升级规则的工单$`, bc.givenResponseSLANotBreachedTicketWithRule)
	sc.Given(`^存在响应 SLA 已超时但 "([^"]*)" 升级规则需等待 (\d+) 分钟的工单$`, bc.givenResponseSLABreachedTicketWithDelayedRule)
	sc.Given(`^已记录当前 SLA 升级规则的 "([^"]*)" 时间线$`, bc.givenCurrentSLAEscalationRuleRecorded)
	sc.Given(`^SLA 保障岗未绑定$`, bc.givenSLAAssurancePostUnbound)
	sc.When(`^执行 SLA 保障扫描$`, bc.whenRunSLAAssuranceScan)
	sc.When(`^执行 SLA 保障扫描，使用 "([^"]*)" 模拟智能体$`, bc.whenRunSLAAssuranceScanWithMode)
	sc.When(`^执行 SLA 保障扫描，AI 执行器不可用$`, bc.whenRunSLAAssuranceScanWithoutExecutor)
	sc.Then(`^SLA 保障岗已调用工具 "([^"]*)"$`, bc.thenSLAAssuranceToolCalled)
	sc.Then(`^SLA 保障岗未调用工具 "([^"]*)"$`, bc.thenSLAAssuranceToolNotCalled)
	sc.Then(`^SLA 保障工具 "([^"]*)" 返回错误包含 "([^"]*)"$`, bc.thenSLAAssuranceToolErrorContains)
	sc.Then(`^工单已转派给 "([^"]*)"$`, bc.thenTicketAssignedToUser)
	sc.Then(`^工单处理人为 "([^"]*)"$`, bc.thenTicketAssignedToUser)
	sc.Then(`^工单优先级为 "([^"]*)"$`, bc.thenTicketPriorityIs)
	sc.Then(`^工单 SLA 状态为 "([^"]*)"$`, bc.thenTicketSLAStatusIs)
	sc.Then(`^时间线中 "([^"]*)" 类型事件数量为 (\d+)$`, bc.thenTimelineEventCountIs)
	sc.Then(`^最新 "([^"]*)" 时间线原因包含 "([^"]*)"$`, bc.thenLatestTimelineReasoningContains)
	sc.Then(`^最新 "([^"]*)" 时间线详情包含当前 SLA 规则$`, bc.thenLatestTimelineDetailsContainCurrentSLARule)
}

func (bc *bddContext) givenPublishedSLAServiceAndAssuranceAgent() error {
	catalog := &ServiceCatalog{Name: "SLA 保障测试目录", Code: "sla-assurance", IsActive: true}
	if err := bc.db.Create(catalog).Error; err != nil {
		return fmt.Errorf("create catalog: %w", err)
	}

	normal := &Priority{Name: "普通", Code: "normal", Value: 3, Color: "#52c41a", IsActive: true}
	if err := bc.db.Create(normal).Error; err != nil {
		return fmt.Errorf("create normal priority: %w", err)
	}
	bc.priority = normal

	urgent := &Priority{Name: "紧急", Code: "urgent", Value: 1, Color: "#f5222d", IsActive: true}
	if err := bc.db.Create(urgent).Error; err != nil {
		return fmt.Errorf("create urgent priority: %w", err)
	}

	sla := &SLATemplate{
		Name:              "BDD 响应 SLA",
		Code:              "bdd-response-sla",
		ResponseMinutes:   1,
		ResolutionMinutes: 60,
		IsActive:          true,
	}
	if err := bc.db.Create(sla).Error; err != nil {
		return fmt.Errorf("create sla template: %w", err)
	}

	decisionAgent := &ai.Agent{
		Name:         "BDD 流程决策智能体",
		Type:         ai.AgentTypeAssistant,
		IsActive:     true,
		Visibility:   "private",
		Strategy:     ai.AgentStrategyReact,
		SystemPrompt: decisionAgentSystemPrompt,
		Temperature:  0.2,
		MaxTokens:    4096,
		MaxTurns:     8,
	}
	if err := bc.db.Create(decisionAgent).Error; err != nil {
		return fmt.Errorf("create decision agent: %w", err)
	}

	slaAgent := &ai.Agent{
		Name:         "BDD SLA 保障智能体",
		Type:         ai.AgentTypeAssistant,
		IsActive:     true,
		Visibility:   "private",
		Strategy:     ai.AgentStrategyReact,
		SystemPrompt: bddSLAAssuranceAgentPrompt,
		Temperature:  0.1,
		MaxTokens:    4096,
		MaxTurns:     8,
	}
	if err := bc.db.Create(slaAgent).Error; err != nil {
		return fmt.Errorf("create sla assurance agent: %w", err)
	}
	bc.slaAssuranceAgentID = slaAgent.ID

	svc := &ServiceDefinition{
		Name:              "SLA 保障智能服务",
		Code:              "sla-assurance-smart-service",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		SLAID:             &sla.ID,
		CollaborationSpec: "SLA 保障 BDD 使用的智能服务，工单由 SLA 保障岗扫描并按规则升级。",
		AgentID:           &decisionAgent.ID,
		IsActive:          true,
	}
	if err := bc.db.Create(svc).Error; err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	bc.service = svc
	return nil
}

func (bc *bddContext) givenResponseSLABreachedTicketWithRule(actionType string) error {
	return bc.createResponseSLATicketWithRule(actionType, 0, -5*time.Minute)
}

func (bc *bddContext) givenResponseSLANotBreachedTicketWithRule(actionType string) error {
	return bc.createResponseSLATicketWithRule(actionType, 0, 5*time.Minute)
}

func (bc *bddContext) givenResponseSLABreachedTicketWithDelayedRule(actionType string, waitMinutes int) error {
	return bc.createResponseSLATicketWithRule(actionType, waitMinutes, -5*time.Minute)
}

func (bc *bddContext) createResponseSLATicketWithRule(actionType string, waitMinutes int, responseDeadlineOffset time.Duration) error {
	if bc.service == nil || bc.service.SLAID == nil {
		return fmt.Errorf("SLA service is not prepared")
	}
	requester, ok := bc.users["申请人"]
	if !ok {
		return fmt.Errorf("申请人 not found")
	}
	current, ok := bc.users["当前处理人"]
	if !ok {
		return fmt.Errorf("当前处理人 not found")
	}

	targetConfig, err := bc.slaTargetConfig(actionType)
	if err != nil {
		return err
	}
	rule := &EscalationRule{
		SLAID:        *bc.service.SLAID,
		TriggerType:  "response_timeout",
		Level:        1,
		WaitMinutes:  waitMinutes,
		ActionType:   actionType,
		TargetConfig: JSONField(targetConfig),
		IsActive:     true,
	}
	if err := bc.db.Create(rule).Error; err != nil {
		return fmt.Errorf("create escalation rule: %w", err)
	}

	now := time.Now()
	responseDeadline := now.Add(responseDeadlineOffset)
	resolutionDeadline := now.Add(55 * time.Minute)
	ticket := &Ticket{
		Code:                  fmt.Sprintf("SLA-BDD-%d", now.UnixNano()),
		Title:                 "生产访问异常待处理",
		Description:           "SLA 保障岗 BDD 风险工单",
		ServiceID:             bc.service.ID,
		EngineType:            "smart",
		Status:                TicketStatusWaitingHuman,
		PriorityID:            bc.priority.ID,
		RequesterID:           requester.ID,
		AssigneeID:            &current.ID,
		Source:                TicketSourceAgent,
		FormData:              JSONField(`{"impact":"production"}`),
		SLAResponseDeadline:   &responseDeadline,
		SLAResolutionDeadline: &resolutionDeadline,
		SLAStatus:             SLAStatusOnTrack,
	}
	if err := bc.db.Create(ticket).Error; err != nil {
		return fmt.Errorf("create ticket: %w", err)
	}
	bc.ticket = ticket
	return nil
}

func (bc *bddContext) givenCurrentSLAEscalationRuleRecorded(eventType string) error {
	if bc.ticket == nil {
		return fmt.Errorf("no ticket in context")
	}
	rule, err := bc.currentSLAEscalationRule()
	if err != nil {
		return err
	}
	details, err := json.Marshal(map[string]any{
		"rule_id":      rule.ID,
		"sla_id":       rule.SLAID,
		"trigger_type": rule.TriggerType,
		"level":        rule.Level,
		"action_type":  rule.ActionType,
		"agent_id":     bc.slaAssuranceAgentID,
		"agent_name":   "BDD SLA 保障智能体",
	})
	if err != nil {
		return err
	}
	return bc.db.Create(&TicketTimeline{
		TicketID:   bc.ticket.ID,
		OperatorID: 0,
		EventType:  eventType,
		Message:    "SLA 升级：已由既有记录处理",
		Details:    JSONField(details),
		Reasoning:  "BDD 预置既有 SLA 保障记录",
	}).Error
}

func (bc *bddContext) givenSLAAssurancePostUnbound() error {
	bc.slaAssuranceAgentID = 0
	return nil
}

func (bc *bddContext) slaTargetConfig(actionType string) ([]byte, error) {
	switch actionType {
	case "notify":
		current, ok := bc.users["当前处理人"]
		if !ok {
			return nil, fmt.Errorf("当前处理人 not found")
		}
		return json.Marshal(map[string]any{
			"recipients": []map[string]any{{"type": "user", "value": fmt.Sprintf("%d", current.ID)}},
			"channelId":  1,
		})
	case "reassign":
		target, ok := bc.users["升级处理人"]
		if !ok {
			return nil, fmt.Errorf("升级处理人 not found")
		}
		return json.Marshal(map[string]any{
			"assigneeCandidates": []map[string]any{{"type": "user", "value": fmt.Sprintf("%d", target.ID)}},
		})
	case "escalate_priority":
		var urgent Priority
		if err := bc.db.Where("code = ?", "urgent").First(&urgent).Error; err != nil {
			return nil, fmt.Errorf("load urgent priority: %w", err)
		}
		return json.Marshal(map[string]any{"priorityId": urgent.ID})
	default:
		return nil, fmt.Errorf("unsupported SLA action type %q", actionType)
	}
}

func (bc *bddContext) whenRunSLAAssuranceScan() error {
	bc.toolCalls = nil
	bc.toolResults = nil
	executor := &testDecisionExecutor{db: bc.db, llmCfg: bc.llmCfg, recordToolCall: bc.recordToolCall, recordToolResult: bc.recordToolResult}
	return bc.runSLAAssuranceScanWithExecutor(executor)
}

func (bc *bddContext) whenRunSLAAssuranceScanWithMode(mode string) error {
	bc.toolCalls = nil
	bc.toolResults = nil
	return bc.runSLAAssuranceScanWithExecutor(&deterministicSLAAssuranceExecutor{bc: bc, mode: mode})
}

func (bc *bddContext) whenRunSLAAssuranceScanWithoutExecutor() error {
	bc.toolCalls = nil
	bc.toolResults = nil
	return bc.runSLAAssuranceScanWithExecutor(nil)
}

func (bc *bddContext) runSLAAssuranceScanWithExecutor(executor app.AIDecisionExecutor) error {
	handler := engine.HandleSLACheck(bc.db, &bddConfigProvider{bc: bc}, executor, engine.NewParticipantResolver(nil), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	if err := handler(ctx, nil); err != nil {
		bc.lastErr = err
		return err
	}
	return nil
}

type deterministicSLAAssuranceExecutor struct {
	bc   *bddContext
	mode string
}

func (e *deterministicSLAAssuranceExecutor) Execute(_ context.Context, _ uint, req app.AIDecisionRequest) (*app.AIDecisionResponse, error) {
	if e.mode == "只说不做" {
		return &app.AIDecisionResponse{Content: "我已经声称已经处理该 SLA 升级，请放心。", Turns: 1}, nil
	}

	rule, err := e.bc.currentSLAEscalationRule()
	if err != nil {
		return nil, err
	}
	ticketID := e.bc.ticket.ID
	if e.mode == "错误工单" {
		ticketID += 999
	}
	ruleID := rule.ID
	if e.mode == "错误规则" {
		ruleID += 999
	}

	if err := e.callTool(req, "sla.risk_queue", json.RawMessage(`{}`)); err != nil {
		return nil, err
	}
	if err := e.callTool(req, "sla.ticket_context", mustMarshalJSON(map[string]any{"ticket_id": e.bc.ticket.ID})); err != nil {
		return nil, err
	}
	if err := e.callTool(req, "sla.escalation_rules", mustMarshalJSON(map[string]any{"ticket_id": e.bc.ticket.ID, "trigger_type": rule.TriggerType})); err != nil {
		return nil, err
	}
	if err := e.callTool(req, "sla.trigger_escalation", mustMarshalJSON(map[string]any{
		"ticket_id": ticketID,
		"rule_id":   ruleID,
		"reasoning": fmt.Sprintf("BDD 模拟智能体按 %s 模式触发 SLA 升级", e.mode),
	})); err != nil {
		return nil, err
	}
	return &app.AIDecisionResponse{Content: "done", Turns: 1}, nil
}

func (e *deterministicSLAAssuranceExecutor) callTool(req app.AIDecisionRequest, name string, args json.RawMessage) error {
	e.bc.recordToolCall(name, args)
	result, err := req.ToolHandler(name, args)
	if err != nil {
		payload := json.RawMessage(fmt.Sprintf(`{"error":%q}`, err.Error()))
		e.bc.recordToolResult(name, payload, true)
		return err
	}
	e.bc.recordToolResult(name, result, false)
	return nil
}

var _ app.AIDecisionExecutor = (*deterministicSLAAssuranceExecutor)(nil)

func mustMarshalJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}

func (bc *bddContext) thenSLAAssuranceToolCalled(name string) error {
	if bc.hasToolCall(name) {
		return nil
	}
	return fmt.Errorf("expected SLA assurance tool %q to be called, got %+v", name, bc.toolCalls)
}

func (bc *bddContext) thenSLAAssuranceToolNotCalled(name string) error {
	if !bc.hasToolCall(name) {
		return nil
	}
	return fmt.Errorf("expected SLA assurance tool %q not to be called, got %+v", name, bc.toolCalls)
}

func (bc *bddContext) thenSLAAssuranceToolErrorContains(name, expected string) error {
	for i := len(bc.toolResults) - 1; i >= 0; i-- {
		result := bc.toolResults[i]
		if result.Name != name {
			continue
		}
		if !result.IsError {
			return fmt.Errorf("expected SLA assurance tool %q to return error, got success: %s", name, result.Output)
		}
		if !strings.Contains(result.Output, expected) {
			return fmt.Errorf("expected SLA assurance tool %q error to contain %q, got %s", name, expected, result.Output)
		}
		return nil
	}
	return fmt.Errorf("SLA assurance tool result %q not found; results=%+v", name, bc.toolResults)
}

func (bc *bddContext) thenTicketAssignedToUser(username string) error {
	if bc.ticket == nil {
		return fmt.Errorf("no ticket in context")
	}
	user, ok := bc.usersByName[username]
	if !ok {
		return fmt.Errorf("user %q not found", username)
	}
	var ticket Ticket
	if err := bc.db.First(&ticket, bc.ticket.ID).Error; err != nil {
		return fmt.Errorf("refresh ticket: %w", err)
	}
	if ticket.AssigneeID == nil || *ticket.AssigneeID != user.ID {
		actual := uint(0)
		if ticket.AssigneeID != nil {
			actual = *ticket.AssigneeID
		}
		return fmt.Errorf("expected assignee %s(%d), got %d", username, user.ID, actual)
	}
	return nil
}

func (bc *bddContext) thenTicketSLAStatusIs(expected string) error {
	if bc.ticket == nil {
		return fmt.Errorf("no ticket in context")
	}
	var ticket Ticket
	if err := bc.db.First(&ticket, bc.ticket.ID).Error; err != nil {
		return fmt.Errorf("refresh ticket: %w", err)
	}
	if ticket.SLAStatus != expected {
		return fmt.Errorf("expected ticket SLA status %q, got %q", expected, ticket.SLAStatus)
	}
	return nil
}

func (bc *bddContext) thenTimelineEventCountIs(eventType string, expected int) error {
	count, err := bc.timelineEventCount(eventType)
	if err != nil {
		return err
	}
	if count != int64(expected) {
		return fmt.Errorf("expected %q timeline count %d, got %d", eventType, expected, count)
	}
	return nil
}

func (bc *bddContext) thenLatestTimelineReasoningContains(eventType, expected string) error {
	event, err := bc.latestTimelineByType(eventType)
	if err != nil {
		return err
	}
	if !strings.Contains(event.Reasoning, expected) {
		return fmt.Errorf("expected latest %q timeline reasoning to contain %q, got %q", eventType, expected, event.Reasoning)
	}
	return nil
}

func (bc *bddContext) thenLatestTimelineDetailsContainCurrentSLARule(eventType string) error {
	event, err := bc.latestTimelineByType(eventType)
	if err != nil {
		return err
	}
	rule, err := bc.currentSLAEscalationRule()
	if err != nil {
		return err
	}
	details := string(event.Details)
	expectedFragments := []string{
		fmt.Sprintf(`"rule_id":%d`, rule.ID),
		fmt.Sprintf(`"trigger_type":"%s"`, rule.TriggerType),
		fmt.Sprintf(`"action_type":"%s"`, rule.ActionType),
		fmt.Sprintf(`"agent_id":%d`, bc.slaAssuranceAgentID),
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(details, fragment) {
			return fmt.Errorf("expected latest %q timeline details to contain %s, got %s", eventType, fragment, details)
		}
	}
	return nil
}

func (bc *bddContext) latestTimelineByType(eventType string) (*TicketTimeline, error) {
	if bc.ticket == nil {
		return nil, fmt.Errorf("no ticket in context")
	}
	var event TicketTimeline
	if err := bc.db.Where("ticket_id = ? AND event_type = ?", bc.ticket.ID, eventType).
		Order("id DESC").
		First(&event).Error; err != nil {
		return nil, fmt.Errorf("load latest %q timeline: %w", eventType, err)
	}
	return &event, nil
}

func (bc *bddContext) currentSLAEscalationRule() (*EscalationRule, error) {
	if bc.service == nil || bc.service.SLAID == nil {
		return nil, fmt.Errorf("SLA service is not prepared")
	}
	var rule EscalationRule
	if err := bc.db.Where("sla_id = ?", *bc.service.SLAID).Order("id DESC").First(&rule).Error; err != nil {
		return nil, fmt.Errorf("load current SLA escalation rule: %w", err)
	}
	return &rule, nil
}

func (bc *bddContext) thenTicketPriorityIs(priorityCode string) error {
	if bc.ticket == nil {
		return fmt.Errorf("no ticket in context")
	}
	var priority Priority
	if err := bc.db.Table("itsm_priorities").
		Joins("JOIN itsm_tickets ON itsm_tickets.priority_id = itsm_priorities.id").
		Where("itsm_tickets.id = ?", bc.ticket.ID).
		First(&priority).Error; err != nil {
		return fmt.Errorf("load ticket priority: %w", err)
	}
	if priority.Code != priorityCode {
		return fmt.Errorf("expected ticket priority %q, got %q", priorityCode, priority.Code)
	}
	return nil
}
