package bdd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cucumber/godog"

	"metis/internal/app"
	. "metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/engine"
)

type scriptedDecisionToolCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type scriptedDecisionSpec struct {
	ToolCalls []scriptedDecisionToolCall `json:"tool_calls"`
	Plan      engine.DecisionPlan        `json:"plan"`
}

type scriptedDecisionExecutor struct {
	bc   *bddContext
	spec scriptedDecisionSpec
}

func (e *scriptedDecisionExecutor) Execute(_ context.Context, _ uint, req app.AIDecisionRequest) (*app.AIDecisionResponse, error) {
	for _, call := range e.spec.ToolCalls {
		args := call.Args
		if len(args) == 0 {
			args = json.RawMessage(`{}`)
		}
		e.bc.recordToolCall(call.Name, args)
		if req.ToolHandler == nil {
			continue
		}
		result, err := req.ToolHandler(call.Name, args)
		if err != nil {
			payload := json.RawMessage(fmt.Sprintf(`{"error":%q}`, err.Error()))
			e.bc.recordToolResult(call.Name, payload, true)
			continue
		}
		e.bc.recordToolResult(call.Name, result, false)
	}

	plan := e.spec.Plan
	if strings.TrimSpace(plan.ExecutionMode) == "" {
		plan.ExecutionMode = "single"
	}
	if plan.Confidence == 0 {
		plan.Confidence = 0.9
	}
	content, _ := json.Marshal(plan)
	return &app.AIDecisionResponse{Content: string(content), Turns: 1}, nil
}

var _ app.AIDecisionExecutor = (*scriptedDecisionExecutor)(nil)

func registerAgenticQualitySteps(sc *godog.ScenarioContext, bc *bddContext) {
	sc.Given(`^已启用脚本化智能决策器:$`, bc.givenScriptedDecisionExecutor)
	sc.Given(`^记录当前工单活动数$`, bc.givenMarkCurrentActivityCount)
	sc.Given(`^当前工单状态强制设为 "([^"]*)"$`, bc.givenForceCurrentTicketStatus)

	sc.When(`^智能引擎执行脚本化决策循环$`, bc.whenScriptedDecisionCycle)
	sc.When(`^执行确定性并行处理决策，岗位为:$`, bc.whenDeterministicParallelProcessPlan)
	sc.When(`^完成一个并行活动$`, bc.whenCompleteOneParallelActivity)
	sc.When(`^完成剩余并行活动$`, bc.whenCompleteRemainingParallelActivities)

	sc.Then(`^决策工具调用顺序为:$`, bc.thenDecisionToolCallOrderIs)
	sc.Then(`^决策工具调用顺序以如下工具开始:$`, bc.thenDecisionToolCallOrderStartsWith)
	sc.Then(`^决策工具 "([^"]*)" 返回结果包含 "([^"]*)"$`, bc.thenDecisionToolResultContains)
	sc.Then(`^决策工具 "([^"]*)" 返回结果不包含 "([^"]*)"$`, bc.thenDecisionToolResultNotContains)
	sc.Then(`^所有决策工具均返回成功$`, bc.thenAllDecisionToolsSucceeded)
	sc.Then(`^工单活动数未变化$`, bc.thenActivityCountUnchanged)
	sc.Then(`^时间线不包含 "([^"]*)" 类型事件$`, bc.thenTimelineNotContainsEventType)
	sc.Then(`^最新决策解释包含字段:$`, bc.thenLatestDecisionExplanationHasFields)
	sc.Then(`^最新决策解释依据包含 "([^"]*)"$`, bc.thenLatestDecisionExplanationBasisContains)
	sc.Then(`^最新决策解释引用真实事实$`, bc.thenLatestDecisionExplanationUsesConcreteFacts)
	sc.Then(`^所有活跃活动共享同一并行组$`, bc.thenActiveActivitiesShareOneParallelGroup)
	sc.Then(`^当前仍有未完成并行活动$`, bc.thenThereArePendingParallelActivities)
	sc.Then(`^并行组已收敛且当前活动已清空$`, bc.thenParallelGroupConvergedAndCurrentCleared)
}

func (bc *bddContext) givenScriptedDecisionExecutor(doc *godog.DocString) error {
	if doc == nil {
		return fmt.Errorf("missing scripted decision JSON")
	}
	var spec scriptedDecisionSpec
	if err := json.Unmarshal([]byte(doc.Content), &spec); err != nil {
		return fmt.Errorf("parse scripted decision JSON: %w", err)
	}
	if strings.TrimSpace(spec.Plan.NextStepType) == "" {
		return fmt.Errorf("scripted decision missing plan.next_step_type")
	}

	orgSvc := &testOrgService{db: bc.db}
	resolver := engine.NewParticipantResolver(orgSvc)
	submitter := engine.TaskSubmitter(&noopSubmitter{})
	if bc.actionReceiver != nil {
		submitter = &syncActionSubmitter{db: bc.db, classicEngine: bc.engine}
	}
	bc.smartEngine = engine.NewSmartEngine(&scriptedDecisionExecutor{bc: bc, spec: spec}, nil, &testUserProvider{db: bc.db}, resolver, submitter, &bddConfigProvider{bc: bc})
	return nil
}

func (bc *bddContext) givenMarkCurrentActivityCount() error {
	if bc.ticket == nil {
		return fmt.Errorf("no ticket in context")
	}
	var count int64
	if err := bc.db.Model(&TicketActivity{}).Where("ticket_id = ?", bc.ticket.ID).Count(&count).Error; err != nil {
		return err
	}
	bc.activityCountMark = count
	return nil
}

func (bc *bddContext) givenForceCurrentTicketStatus(status string) error {
	if bc.ticket == nil {
		return fmt.Errorf("no ticket in context")
	}
	if err := bc.db.Model(&Ticket{}).Where("id = ?", bc.ticket.ID).Update("status", status).Error; err != nil {
		return fmt.Errorf("force ticket status: %w", err)
	}
	return bc.db.First(bc.ticket, bc.ticket.ID).Error
}

func (bc *bddContext) whenScriptedDecisionCycle() error {
	if bc.ticket == nil {
		return fmt.Errorf("no ticket in context")
	}
	bc.toolCalls = nil
	bc.toolResults = nil
	return bc.runSmartDecisionCycle(nil)
}

func (bc *bddContext) thenDecisionToolCallOrderIs(table *godog.Table) error {
	return bc.assertDecisionToolOrder(table, false)
}

func (bc *bddContext) thenDecisionToolCallOrderStartsWith(table *godog.Table) error {
	return bc.assertDecisionToolOrder(table, true)
}

func (bc *bddContext) assertDecisionToolOrder(table *godog.Table, prefixOnly bool) error {
	expected := singleColumnTableValues(table)
	actual := make([]string, 0, len(bc.toolCalls))
	for _, call := range bc.toolCalls {
		actual = append(actual, call.Name)
	}
	if prefixOnly {
		if len(actual) < len(expected) {
			return fmt.Errorf("expected tool call prefix %v, got %v", expected, actual)
		}
		actual = actual[:len(expected)]
	}
	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		return fmt.Errorf("expected tool order %v, got %v", expected, actual)
	}
	return nil
}

func singleColumnTableValues(table *godog.Table) []string {
	values := make([]string, 0, len(table.Rows))
	for _, row := range table.Rows {
		if len(row.Cells) == 0 {
			continue
		}
		value := strings.TrimSpace(row.Cells[0].Value)
		if value == "" || value == "工具" || value == "字段" {
			continue
		}
		values = append(values, value)
	}
	return values
}

func (bc *bddContext) thenDecisionToolResultContains(toolName, needle string) error {
	result, err := bc.latestDecisionToolResult(toolName)
	if err != nil {
		return err
	}
	if !strings.Contains(result.Output, needle) {
		return fmt.Errorf("expected %s result to contain %q, got %s", toolName, needle, result.Output)
	}
	return nil
}

func (bc *bddContext) thenDecisionToolResultNotContains(toolName, needle string) error {
	result, err := bc.latestDecisionToolResult(toolName)
	if err != nil {
		return err
	}
	if strings.Contains(result.Output, needle) {
		return fmt.Errorf("expected %s result not to contain %q, got %s", toolName, needle, result.Output)
	}
	return nil
}

func (bc *bddContext) latestDecisionToolResult(toolName string) (*bddToolResult, error) {
	for i := len(bc.toolResults) - 1; i >= 0; i-- {
		if bc.toolResults[i].Name == toolName {
			return &bc.toolResults[i], nil
		}
	}
	return nil, fmt.Errorf("tool result %q not found; results=%+v", toolName, bc.toolResults)
}

func (bc *bddContext) thenAllDecisionToolsSucceeded() error {
	for _, result := range bc.toolResults {
		if result.IsError || strings.Contains(result.Output, `"error"`) {
			return fmt.Errorf("tool %s returned error: %s", result.Name, result.Output)
		}
	}
	return nil
}

func (bc *bddContext) thenActivityCountUnchanged() error {
	if bc.ticket == nil {
		return fmt.Errorf("no ticket in context")
	}
	var count int64
	if err := bc.db.Model(&TicketActivity{}).Where("ticket_id = ?", bc.ticket.ID).Count(&count).Error; err != nil {
		return err
	}
	if count != bc.activityCountMark {
		return fmt.Errorf("expected activity count to remain %d, got %d", bc.activityCountMark, count)
	}
	return nil
}

func (bc *bddContext) thenTimelineNotContainsEventType(eventType string) error {
	if bc.ticket == nil {
		return fmt.Errorf("no ticket in context")
	}
	var count int64
	if err := bc.db.Model(&TicketTimeline{}).
		Where("ticket_id = ? AND event_type = ?", bc.ticket.ID, eventType).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("expected no %q timeline event, got %d", eventType, count)
	}
	return nil
}

func (bc *bddContext) latestDecisionExplanation() (map[string]any, error) {
	if bc.ticket == nil {
		return nil, fmt.Errorf("no ticket in context")
	}
	var events []TicketTimeline
	if err := bc.db.Where("ticket_id = ? AND details <> ''", bc.ticket.ID).
		Order("id DESC").Find(&events).Error; err != nil {
		return nil, err
	}
	for _, event := range events {
		var details map[string]any
		if err := json.Unmarshal([]byte(event.Details), &details); err != nil {
			continue
		}
		raw, ok := details["decision_explanation"]
		if !ok {
			continue
		}
		explanation, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		return explanation, nil
	}
	return nil, fmt.Errorf("decision explanation not found for ticket %d", bc.ticket.ID)
}

func (bc *bddContext) thenLatestDecisionExplanationHasFields(table *godog.Table) error {
	explanation, err := bc.latestDecisionExplanation()
	if err != nil {
		return err
	}
	for _, field := range singleColumnTableValues(table) {
		if _, ok := explanation[field]; !ok {
			return fmt.Errorf("decision explanation missing field %q: %+v", field, explanation)
		}
	}
	return nil
}

func (bc *bddContext) thenLatestDecisionExplanationBasisContains(needle string) error {
	explanation, err := bc.latestDecisionExplanation()
	if err != nil {
		return err
	}
	basis := fmt.Sprint(explanation["basis"])
	if !strings.Contains(basis, needle) {
		return fmt.Errorf("expected decision explanation basis to contain %q, got %q", needle, basis)
	}
	return nil
}

func (bc *bddContext) thenLatestDecisionExplanationUsesConcreteFacts() error {
	explanation, err := bc.latestDecisionExplanation()
	if err != nil {
		return err
	}
	basis := fmt.Sprint(explanation["basis"])
	if strings.TrimSpace(basis) == "" {
		return fmt.Errorf("decision explanation basis is empty: %+v", explanation)
	}
	for _, marker := range []string{
		"decision.ticket_context",
		"workflow_context",
		"request_kind",
		"completed_activity",
		"activity_history",
		"resolve_participant",
		"form.",
	} {
		if strings.Contains(basis, marker) {
			return nil
		}
	}
	return fmt.Errorf("decision explanation basis lacks concrete facts: %q", basis)
}

func (bc *bddContext) whenDeterministicParallelProcessPlan(table *godog.Table) error {
	if bc.ticket == nil {
		return fmt.Errorf("no ticket in context")
	}
	if err := bc.db.Model(&Ticket{}).Where("id = ?", bc.ticket.ID).Update("status", TicketStatusDecisioning).Error; err != nil {
		return fmt.Errorf("update ticket status: %w", err)
	}

	activities := make([]engine.DecisionActivity, 0)
	for _, row := range table.Rows {
		if len(row.Cells) < 2 {
			continue
		}
		deptCode := strings.TrimSpace(row.Cells[0].Value)
		positionCode := strings.TrimSpace(row.Cells[1].Value)
		if deptCode == "" || deptCode == "部门" || positionCode == "" || positionCode == "岗位" {
			continue
		}
		activities = append(activities, engine.DecisionActivity{
			Type:            engine.NodeProcess,
			ParticipantType: "position_department",
			DepartmentCode:  deptCode,
			PositionCode:    positionCode,
			Instructions:    fmt.Sprintf("并行处理：%s/%s", deptCode, positionCode),
		})
	}
	if len(activities) < 2 {
		return fmt.Errorf("parallel decision needs at least two activities, got %d", len(activities))
	}

	plan := &engine.DecisionPlan{
		NextStepType:  engine.NodeProcess,
		ExecutionMode: "parallel",
		Activities:    activities,
		Reasoning:     "decision.ticket_context.parallel_groups 为空；协作规范要求多角色并行处理。",
		Confidence:    0.92,
	}
	if err := bc.smartEngine.ExecuteDecisionPlan(bc.db, bc.ticket.ID, plan); err != nil {
		return fmt.Errorf("execute parallel plan: %w", err)
	}
	return bc.db.First(bc.ticket, bc.ticket.ID).Error
}

func (bc *bddContext) thenActiveActivitiesShareOneParallelGroup() error {
	activities, err := bc.activeParallelActivities()
	if err != nil {
		return err
	}
	if len(activities) < 2 {
		return fmt.Errorf("expected at least 2 active parallel activities, got %d", len(activities))
	}
	groupID := activities[0].ActivityGroupID
	if strings.TrimSpace(groupID) == "" {
		return fmt.Errorf("parallel activity %d has empty group id", activities[0].ID)
	}
	for _, activity := range activities[1:] {
		if activity.ActivityGroupID != groupID {
			return fmt.Errorf("parallel activities do not share group id: first=%q activity %d=%q", groupID, activity.ID, activity.ActivityGroupID)
		}
	}
	return nil
}

func (bc *bddContext) activeParallelActivities() ([]TicketActivity, error) {
	var activities []TicketActivity
	err := bc.db.Where("ticket_id = ? AND activity_group_id <> '' AND status IN ?",
		bc.ticket.ID, []string{engine.ActivityPending, engine.ActivityInProgress}).
		Order("id ASC").Find(&activities).Error
	return activities, err
}

func (bc *bddContext) whenCompleteOneParallelActivity() error {
	activities, err := bc.activeParallelActivities()
	if err != nil {
		return err
	}
	if len(activities) == 0 {
		return fmt.Errorf("no active parallel activities")
	}
	return bc.completeSpecificActivity(activities[0], engine.ActivityCompleted, "")
}

func (bc *bddContext) whenCompleteRemainingParallelActivities() error {
	activities, err := bc.activeParallelActivities()
	if err != nil {
		return err
	}
	for _, activity := range activities {
		if err := bc.completeSpecificActivity(activity, engine.ActivityCompleted, ""); err != nil {
			return err
		}
	}
	return bc.db.First(bc.ticket, bc.ticket.ID).Error
}

func (bc *bddContext) completeSpecificActivity(activity TicketActivity, outcome string, opinion string) error {
	var assignment TicketAssignment
	if err := bc.db.Where("activity_id = ?", activity.ID).First(&assignment).Error; err != nil {
		fallbackID := bc.findFallbackOperator()
		if fallbackID == 0 {
			return fmt.Errorf("no operator available for activity %d", activity.ID)
		}
		assignment = TicketAssignment{
			TicketID:        bc.ticket.ID,
			ActivityID:      activity.ID,
			ParticipantType: "user",
			UserID:          &fallbackID,
			AssigneeID:      &fallbackID,
			Status:          "claimed",
			IsCurrent:       true,
		}
		if err := bc.db.Create(&assignment).Error; err != nil {
			return fmt.Errorf("create fallback assignment: %w", err)
		}
	}
	operatorID := uint(0)
	if assignment.AssigneeID != nil {
		operatorID = *assignment.AssigneeID
	} else if assignment.UserID != nil {
		operatorID = *assignment.UserID
	} else {
		operatorID = bc.resolveOperatorFromAssignment(assignment)
	}
	if operatorID == 0 {
		return fmt.Errorf("could not resolve operator for activity %d", activity.ID)
	}
	if err := bc.db.Model(&TicketAssignment{}).Where("activity_id = ?", activity.ID).
		Updates(map[string]any{"assignee_id": operatorID, "status": "claimed"}).Error; err != nil {
		return fmt.Errorf("claim activity %d: %w", activity.ID, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := bc.smartEngine.Progress(ctx, bc.db, engine.ProgressParams{
		TicketID:   bc.ticket.ID,
		ActivityID: activity.ID,
		Outcome:    outcome,
		Opinion:    opinion,
		OperatorID: operatorID,
	}); err != nil {
		return fmt.Errorf("complete activity %d: %w", activity.ID, err)
	}
	return nil
}

func (bc *bddContext) thenThereArePendingParallelActivities() error {
	activities, err := bc.activeParallelActivities()
	if err != nil {
		return err
	}
	if len(activities) == 0 {
		return fmt.Errorf("expected pending parallel activities, got none")
	}
	return nil
}

func (bc *bddContext) thenParallelGroupConvergedAndCurrentCleared() error {
	activities, err := bc.activeParallelActivities()
	if err != nil {
		return err
	}
	if len(activities) > 0 {
		return fmt.Errorf("expected no active parallel activities after convergence, got %d", len(activities))
	}
	if err := bc.db.First(bc.ticket, bc.ticket.ID).Error; err != nil {
		return err
	}
	if bc.ticket.CurrentActivityID != nil {
		return fmt.Errorf("expected current_activity_id to be cleared, got %d", *bc.ticket.CurrentActivityID)
	}
	return nil
}
