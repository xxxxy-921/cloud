package bdd

// steps_boss_test.go — step definitions for the Boss serial process BDD scenarios.

import (
	"encoding/json"
	"fmt"
	. "metis/internal/app/itsm/domain"
	"time"

	"github.com/cucumber/godog"

	"metis/internal/app/itsm/engine"
)

// registerBossSteps registers all Boss serial process step definitions.
func registerBossSteps(sc *godog.ScenarioContext, bc *bddContext) {
	sc.Given(`^已定义高风险变更协同申请协作规范$`, bc.givenBossCollaborationSpec)
	sc.Given(`^已基于协作规范发布高风险变更协同申请服务（智能引擎）$`, bc.givenBossSmartServicePublished)
	sc.Given(`^"([^"]*)" 已创建高风险变更工单，场景为 "([^"]*)"$`, bc.givenBossTicketCreated)
	sc.Given(`^"([^"]*)" 已创建高风险变更工单 "([^"]*)"，场景为 "([^"]*)"$`, bc.givenBossTicketCreatedWithAlias)
	sc.Given(`^"([^"]*)" 已创建高风险变更工单，表单数据为:$`, bc.givenBossTicketCreatedWithFormData)
	sc.Given(`^高风险变更工作流参考图错误地把首级岗位改成 "([^"]*)/([^"]*)"$`, bc.givenBossWorkflowFirstStepMislabeled)
	sc.Given(`^高风险变更工作流参考图错误地把首级岗位改成旧固定用户 "([^"]*)"$`, bc.givenBossWorkflowFirstStepLegacyUser)
	sc.Given(`^高风险变更工作流参考图错误地把驳回指向申请人补充表单$`, bc.givenBossWorkflowRejectedReturnsRequesterForm)
	sc.Given(`^高风险变更岗位 "([^"]*)/([^"]*)" 处理人已停用$`, bc.givenBossPositionInactive)

	sc.Then(`^工单的表单数据中包含完整的 change_items 明细表格$`, bc.thenFormDataContainsChangeItems)
	sc.Then(`^工单 "([^"]*)" 的处理记录与工单 "([^"]*)" 完全隔离$`, bc.thenProcessRecordsIsolated)
	sc.Then(`^当前处理任务分配到岗位部门 "([^"]*)/([^"]*)"$`, bc.thenCurrentProcessAssignedToDepartmentPosition)
	sc.Then(`^当前处理任务未分配到岗位部门 "([^"]*)/([^"]*)"$`, bc.thenCurrentProcessNotAssignedToDepartmentPosition)
	sc.Then(`^当前岗位部门 "([^"]*)/([^"]*)" 的活跃处理任务数为 (\d+)$`, bc.thenActiveProcessActivityCountForDepartmentPositionIs)
}

// --- Given steps ---

func (bc *bddContext) givenBossCollaborationSpec() error {
	bc.collaborationSpec = bossCollaborationSpec
	return nil
}

func (bc *bddContext) givenBossSmartServicePublished() error {
	return publishBossSmartService(bc)
}

func (bc *bddContext) givenBossTicketCreated(username, caseKey string) error {
	payload, ok := bossCasePayloads[caseKey]
	if !ok {
		return fmt.Errorf("unknown case key %q, expected one of: requester-1, requester-2", caseKey)
	}
	return bc.createBossTicket(username, fmt.Sprintf("BOSS-%s", caseKey), payload.Summary, payload.FormData, bc.service.WorkflowJSON)
}

func (bc *bddContext) givenBossTicketCreatedWithAlias(username, alias, caseKey string) error {
	payload, ok := bossCasePayloads[caseKey]
	if !ok {
		return fmt.Errorf("unknown case key %q", caseKey)
	}

	if err := bc.createBossTicket(username, fmt.Sprintf("BOSS-%s", alias), payload.Summary, payload.FormData, bc.service.WorkflowJSON); err != nil {
		return err
	}
	bc.tickets[alias] = bc.ticket
	return nil
}

func (bc *bddContext) givenBossTicketCreatedWithFormData(username string, doc *godog.DocString) error {
	if doc == nil {
		return fmt.Errorf("missing form data doc string")
	}
	var formData map[string]any
	if err := json.Unmarshal([]byte(doc.Content), &formData); err != nil {
		return fmt.Errorf("parse form data JSON: %w", err)
	}
	return bc.createBossTicket(username, "BOSS-CORNER", "高风险变更协同申请：corner case", formData, bc.service.WorkflowJSON)
}

func (bc *bddContext) createBossTicket(username, codePrefix, title string, formData map[string]any, workflowJSON JSONField) error {
	user, ok := bc.usersByName[username]
	if !ok {
		return fmt.Errorf("user %q not found in context", username)
	}

	formJSON, _ := json.Marshal(formData)

	ticket := &Ticket{
		Code:         fmt.Sprintf("%s-%d", codePrefix, time.Now().UnixNano()),
		Title:        title,
		ServiceID:    bc.service.ID,
		EngineType:   "smart",
		Status:       "pending",
		PriorityID:   bc.priority.ID,
		RequesterID:  user.ID,
		FormData:     JSONField(formJSON),
		WorkflowJSON: workflowJSON,
	}
	if err := bc.db.Create(ticket).Error; err != nil {
		return fmt.Errorf("create ticket: %w", err)
	}
	bc.ticket = ticket
	return nil
}

func (bc *bddContext) givenBossWorkflowFirstStepMislabeled(departmentCode, positionCode string) error {
	if bc.service == nil {
		return fmt.Errorf("no service in context")
	}
	corrupted, err := corruptBossWorkflowFirstParticipant(json.RawMessage(bc.service.WorkflowJSON), departmentCode, positionCode)
	if err != nil {
		return err
	}
	bc.service.WorkflowJSON = JSONField(corrupted)
	return bc.db.Save(bc.service).Error
}

func (bc *bddContext) givenBossWorkflowFirstStepLegacyUser(username string) error {
	if bc.service == nil {
		return fmt.Errorf("no service in context")
	}
	corrupted, err := corruptBossWorkflowFirstParticipantToLegacyUser(json.RawMessage(bc.service.WorkflowJSON), username)
	if err != nil {
		return err
	}
	bc.service.WorkflowJSON = JSONField(corrupted)
	return bc.db.Save(bc.service).Error
}

func (bc *bddContext) givenBossWorkflowRejectedReturnsRequesterForm() error {
	if bc.service == nil {
		return fmt.Errorf("no service in context")
	}
	corrupted, err := corruptBossWorkflowRejectedTarget(json.RawMessage(bc.service.WorkflowJSON))
	if err != nil {
		return err
	}
	bc.service.WorkflowJSON = JSONField(corrupted)
	return bc.db.Save(bc.service).Error
}

func (bc *bddContext) givenBossPositionInactive(departmentCode, positionCode string) error {
	var userIDs []uint
	err := bc.db.Table("users").
		Joins("JOIN user_positions ON user_positions.user_id = users.id").
		Joins("JOIN positions ON positions.id = user_positions.position_id").
		Joins("JOIN departments ON departments.id = user_positions.department_id").
		Where("departments.code = ? AND positions.code = ?", departmentCode, positionCode).
		Pluck("users.id", &userIDs).Error
	if err != nil {
		return fmt.Errorf("query users for %s/%s: %w", departmentCode, positionCode, err)
	}
	if len(userIDs) == 0 {
		return fmt.Errorf("no users found for %s/%s", departmentCode, positionCode)
	}
	return bc.db.Table("users").Where("id IN ?", userIDs).Update("is_active", false).Error
}

// --- Then steps ---

// thenFormDataContainsChangeItems asserts that the current ticket's form_data
// contains a complete change_items array with expected fields.
func (bc *bddContext) thenFormDataContainsChangeItems() error {
	if bc.ticket == nil {
		return fmt.Errorf("no ticket in context")
	}

	// Refresh ticket from DB
	if err := bc.db.First(bc.ticket, bc.ticket.ID).Error; err != nil {
		return fmt.Errorf("refresh ticket: %w", err)
	}

	var formData map[string]any
	if err := json.Unmarshal([]byte(bc.ticket.FormData), &formData); err != nil {
		return fmt.Errorf("parse form_data: %w", err)
	}

	itemsRaw, ok := formData["change_items"]
	if !ok {
		keys := make([]string, 0, len(formData))
		for k := range formData {
			keys = append(keys, k)
		}
		return fmt.Errorf("form_data missing 'change_items' key, got keys: %v", keys)
	}

	items, ok := itemsRaw.([]any)
	if !ok {
		return fmt.Errorf("change_items is not an array, type: %T", itemsRaw)
	}

	if len(items) == 0 {
		return fmt.Errorf("change_items array is empty")
	}

	// Find the original payload to compare
	var expectedItems []map[string]any
	for _, cp := range bossCasePayloads {
		fd := cp.FormData
		if ri, ok := fd["change_items"].([]map[string]any); ok {
			if len(ri) == len(items) {
				expectedItems = ri
				break
			}
		}
	}

	requiredFields := []string{"system", "resource", "permission_level", "reason"}

	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return fmt.Errorf("change_items[%d] is not an object, type: %T", i, item)
		}
		for _, field := range requiredFields {
			val, exists := m[field]
			if !exists {
				return fmt.Errorf("change_items[%d] missing field %q", i, field)
			}
			strVal, ok := val.(string)
			if !ok || strVal == "" {
				return fmt.Errorf("change_items[%d].%s is empty or non-string", i, field)
			}
			// If we have expected items, verify values match
			if expectedItems != nil && i < len(expectedItems) {
				if expected, ok := expectedItems[i][field].(string); ok && expected != strVal {
					return fmt.Errorf("change_items[%d].%s = %q, expected %q", i, field, strVal, expected)
				}
			}
		}
	}

	return nil
}

func (bc *bddContext) thenCurrentProcessAssignedToDepartmentPosition(departmentCode, positionCode string) error {
	activity, err := bc.getCurrentActivity()
	if err != nil {
		return err
	}
	if bc.activityAssignedToDepartmentPosition(activity.ID, departmentCode, positionCode) {
		return nil
	}
	return fmt.Errorf("current activity %d is not assigned to %s/%s", activity.ID, departmentCode, positionCode)
}

func (bc *bddContext) thenCurrentProcessNotAssignedToDepartmentPosition(departmentCode, positionCode string) error {
	if err := bc.thenCurrentProcessAssignedToDepartmentPosition(departmentCode, positionCode); err == nil {
		return fmt.Errorf("current process unexpectedly assigned to %s/%s", departmentCode, positionCode)
	}
	return nil
}

func (bc *bddContext) thenActiveProcessActivityCountForDepartmentPositionIs(departmentCode, positionCode string, expected int) error {
	var activities []TicketActivity
	if err := bc.db.Where("ticket_id = ? AND activity_type = ? AND status IN ?",
		bc.ticket.ID, engine.NodeProcess, []string{engine.ActivityPending, engine.ActivityInProgress}).
		Find(&activities).Error; err != nil {
		return err
	}

	actual := 0
	for _, activity := range activities {
		if bc.activityAssignedToDepartmentPosition(activity.ID, departmentCode, positionCode) {
			actual++
		}
	}
	if actual != expected {
		return fmt.Errorf("expected %d active process activities for %s/%s, got %d", expected, departmentCode, positionCode, actual)
	}
	return nil
}

func (bc *bddContext) activityAssignedToDepartmentPosition(activityID uint, departmentCode, positionCode string) bool {
	var assignments []TicketAssignment
	if err := bc.db.Where("activity_id = ?", activityID).Find(&assignments).Error; err != nil {
		return false
	}

	orgSvc := &testOrgService{db: bc.db}
	eligibleIDs, err := orgSvc.FindUsersByPositionAndDepartment(positionCode, departmentCode)
	if err != nil {
		eligibleIDs = nil
	}
	eligible := make(map[uint]struct{}, len(eligibleIDs))
	for _, id := range eligibleIDs {
		eligible[id] = struct{}{}
	}

	for _, assignment := range assignments {
		if assignment.PositionID != nil && assignment.DepartmentID != nil {
			pos, ok := bc.positions[positionCode]
			if !ok || pos.ID != *assignment.PositionID {
				continue
			}
			dept, ok := bc.departments[departmentCode]
			if ok && dept.ID == *assignment.DepartmentID {
				return true
			}
			continue
		}

		var userID uint
		if assignment.AssigneeID != nil {
			userID = *assignment.AssigneeID
		} else if assignment.UserID != nil {
			userID = *assignment.UserID
		}
		if userID > 0 {
			if _, ok := eligible[userID]; ok {
				return true
			}
		}
	}
	return false
}

func corruptBossWorkflowFirstParticipant(raw json.RawMessage, departmentCode, positionCode string) (json.RawMessage, error) {
	return corruptBossWorkflowFirstParticipantWith(raw, []any{
		map[string]any{
			"type":            "position_department",
			"department_code": departmentCode,
			"position_code":   positionCode,
		},
	})
}

func corruptBossWorkflowFirstParticipantToLegacyUser(raw json.RawMessage, username string) (json.RawMessage, error) {
	return corruptBossWorkflowFirstParticipantWith(raw, []any{
		map[string]any{
			"type":  "user",
			"value": username,
		},
	})
}

func corruptBossWorkflowFirstParticipantWith(raw json.RawMessage, participants []any) (json.RawMessage, error) {
	var wf vpnWorkflowDoc
	if err := json.Unmarshal(raw, &wf); err != nil {
		return nil, fmt.Errorf("parse workflow_json: %w", err)
	}

	target := -1
	for i := range wf.Nodes {
		if wf.Nodes[i].Type != engine.NodeProcess && wf.Nodes[i].Type != engine.NodeApprove {
			continue
		}
		if bossNodeTargetsPosition(wf.Nodes[i], "headquarters", "serial_reviewer") {
			target = i
			break
		}
		if target == -1 {
			target = i
		}
	}
	if target == -1 {
		return nil, fmt.Errorf("workflow_json missing boss first human node")
	}

	if wf.Nodes[target].Data == nil {
		wf.Nodes[target].Data = make(map[string]any)
	}
	wf.Nodes[target].Data["participants"] = participants
	corrupted, err := json.Marshal(wf)
	if err != nil {
		return nil, fmt.Errorf("marshal corrupted workflow_json: %w", err)
	}
	return corrupted, nil
}

func corruptBossWorkflowRejectedTarget(raw json.RawMessage) (json.RawMessage, error) {
	var wf vpnWorkflowDoc
	if err := json.Unmarshal(raw, &wf); err != nil {
		return nil, fmt.Errorf("parse workflow_json: %w", err)
	}

	formID := "boss_requester_supplement"
	wf.Nodes = append(wf.Nodes, vpnWorkflowNode{
		ID:   formID,
		Type: engine.NodeForm,
		Data: map[string]any{
			"label":    "申请人补充高风险变更信息",
			"nodeType": engine.NodeForm,
			"participants": []any{
				map[string]any{"type": "requester"},
			},
			"formSchema": map[string]any{
				"fields": []any{
					map[string]any{"key": "supplement_reason", "type": "textarea", "label": "补充说明"},
				},
			},
		},
	})

	changed := false
	for i := range wf.Edges {
		if edgeOutcome(wf.Edges[i].Data) == engine.ActivityRejected {
			wf.Edges[i].Target = formID
			changed = true
		}
	}
	if !changed {
		for _, node := range wf.Nodes {
			if node.Type != engine.NodeProcess && node.Type != engine.NodeApprove {
				continue
			}
			wf.Edges = append(wf.Edges, vpnWorkflowEdge{
				ID:     fmt.Sprintf("edge_%s_rejected_boss_supplement", node.ID),
				Source: node.ID,
				Target: formID,
				Data:   map[string]any{"outcome": engine.ActivityRejected},
			})
			changed = true
		}
	}
	if !changed {
		return nil, fmt.Errorf("workflow_json has no human nodes to corrupt rejected target")
	}

	if endID := firstEndNodeID(wf.Nodes); endID != "" {
		wf.Edges = append(wf.Edges, vpnWorkflowEdge{
			ID:     "edge_boss_requester_supplement_end",
			Source: formID,
			Target: endID,
		})
	}

	corrupted, err := json.Marshal(wf)
	if err != nil {
		return nil, fmt.Errorf("marshal corrupted workflow_json: %w", err)
	}
	return corrupted, nil
}

func bossNodeTargetsPosition(node vpnWorkflowNode, departmentCode, positionCode string) bool {
	rawParticipants, ok := node.Data["participants"].([]any)
	if !ok {
		return false
	}
	for _, rawParticipant := range rawParticipants {
		participant, ok := rawParticipant.(map[string]any)
		if !ok {
			continue
		}
		if fmt.Sprint(participant["type"]) == "position_department" &&
			fmt.Sprint(participant["department_code"]) == departmentCode &&
			fmt.Sprint(participant["position_code"]) == positionCode {
			return true
		}
	}
	return false
}

// thenProcessRecordsIsolated asserts that two tickets' TicketAssignment records are completely isolated.
func (bc *bddContext) thenProcessRecordsIsolated(ticketRefA, ticketRefB string) error {
	ticketA, ok := bc.tickets[ticketRefA]
	if !ok {
		return fmt.Errorf("ticket alias %q not found in context", ticketRefA)
	}
	ticketB, ok := bc.tickets[ticketRefB]
	if !ok {
		return fmt.Errorf("ticket alias %q not found in context", ticketRefB)
	}

	// Check A has assignment records
	var countA int64
	bc.db.Model(&TicketAssignment{}).Where("ticket_id = ?", ticketA.ID).Count(&countA)
	if countA == 0 {
		return fmt.Errorf("ticket %q has no assignment records", ticketRefA)
	}

	// Check B has assignment records
	var countB int64
	bc.db.Model(&TicketAssignment{}).Where("ticket_id = ?", ticketB.ID).Count(&countB)
	if countB == 0 {
		return fmt.Errorf("ticket %q has no assignment records", ticketRefB)
	}

	// Verify no cross-contamination via activity IDs
	var activitiesA []TicketActivity
	bc.db.Where("ticket_id = ?", ticketA.ID).Find(&activitiesA)
	activityIDsA := make(map[uint]bool)
	for _, a := range activitiesA {
		activityIDsA[a.ID] = true
	}

	var activitiesB []TicketActivity
	bc.db.Where("ticket_id = ?", ticketB.ID).Find(&activitiesB)
	activityIDsB := make(map[uint]bool)
	for _, a := range activitiesB {
		activityIDsB[a.ID] = true
	}

	// Verify A's assignments only reference A's activities
	var assignmentsA []TicketAssignment
	bc.db.Where("ticket_id = ?", ticketA.ID).Find(&assignmentsA)
	for _, asgn := range assignmentsA {
		if activityIDsB[asgn.ActivityID] {
			return fmt.Errorf("ticket %q's assignment references ticket %q's activity %d", ticketRefA, ticketRefB, asgn.ActivityID)
		}
	}

	// Verify B's assignments only reference B's activities
	var assignmentsB []TicketAssignment
	bc.db.Where("ticket_id = ?", ticketB.ID).Find(&assignmentsB)
	for _, asgn := range assignmentsB {
		if activityIDsA[asgn.ActivityID] {
			return fmt.Errorf("ticket %q's assignment references ticket %q's activity %d", ticketRefB, ticketRefA, asgn.ActivityID)
		}
	}

	return nil
}
