package bdd

// steps_parallel_approval_test.go — step definitions for multi-role parallel approval BDD scenarios.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	. "metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/tools"
	"time"

	"github.com/cucumber/godog"

	"metis/internal/app/itsm/engine"
)

// registerParallelApprovalSteps registers all parallel approval BDD step definitions.
func registerParallelApprovalSteps(sc *godog.ScenarioContext, bc *bddContext) {
	sc.Given(`^已定义多角色并签申请协作规范$`, bc.givenParallelApprovalCollaborationSpec)
	sc.Given(`^已基于协作规范发布多角色并签申请服务（智能引擎）$`, bc.givenParallelApprovalSmartServicePublished)
	sc.Given(`^"([^"]*)" 已创建并签申请工单，场景为 "([^"]*)"$`, bc.givenParallelApprovalTicketCreated)
	sc.Given(`^已发布多角色并签申请对话测试服务$`, bc.givenParallelApprovalDialogServicePublished)

	sc.When(`^并签审批组中岗位 "([^"]*)" 的审批人认领并审批通过$`, bc.whenParallelApprovalRoleApproves)
	sc.When(`^当前活动的被分配人认领并审批通过$`, bc.whenCurrentActivityApproved)

	sc.Then(`^应存在一个并签审批活动组，包含 (\d+) 个并行活动$`, bc.thenParallelApprovalGroupExists)
	sc.Then(`^并签审批组仍有未完成活动，不应触发下一步$`, bc.thenParallelApprovalGroupNotConverged)
	sc.Then(`^并签审批组全部完成，应触发下一轮决策$`, bc.thenParallelApprovalGroupConverged)
	sc.Then(`^不应存在分配给岗位 "([^"]*)" 的待处理审批活动$`, bc.thenNoApprovalActivityForPosition)
}

// --- Given steps ---

func (bc *bddContext) givenParallelApprovalCollaborationSpec() error {
	bc.collaborationSpec = parallelApprovalCollaborationSpec
	return nil
}

func (bc *bddContext) givenParallelApprovalSmartServicePublished() error {
	return publishParallelApprovalSmartService(bc)
}

func (bc *bddContext) givenParallelApprovalTicketCreated(username, caseKey string) error {
	user, ok := bc.usersByName[username]
	if !ok {
		return fmt.Errorf("user %q not found in context", username)
	}

	payload, ok := parallelApprovalCasePayloads[caseKey]
	if !ok {
		return fmt.Errorf("unknown parallel approval case key %q", caseKey)
	}

	formJSON, _ := json.Marshal(payload.FormData)
	op := tools.NewOperator(bc.db, nil, nil, nil, nil, nil)
	detail, err := op.LoadService(bc.service.ID)
	if err != nil {
		return fmt.Errorf("load service snapshot: %w", err)
	}

	ticket := &Ticket{
		Code:             fmt.Sprintf("PA-%s-%d", caseKey, time.Now().UnixNano()),
		Title:            payload.Summary,
		ServiceID:        bc.service.ID,
		ServiceVersionID: uintPtr(detail.ServiceVersionID),
		EngineType:       "smart",
		Status:           "pending",
		PriorityID:       bc.priority.ID,
		RequesterID:      user.ID,
		FormData:         JSONField(formJSON),
		WorkflowJSON:     bc.service.WorkflowJSON,
	}
	if err := bc.db.Create(ticket).Error; err != nil {
		return fmt.Errorf("create ticket: %w", err)
	}
	bc.ticket = ticket
	return nil
}

func (bc *bddContext) givenParallelApprovalDialogServicePublished() error {
	return publishParallelApprovalDialogService(bc)
}

// --- When steps ---

// whenParallelApprovalRoleApproves finds the parallel approve activity assigned to a specific
// position code and approves it via SmartEngine.Progress() with outcome="approved".
func (bc *bddContext) whenParallelApprovalRoleApproves(positionCode string) error {
	if bc.ticket == nil {
		return fmt.Errorf("no ticket in context")
	}

	// Find pending parallel activities.
	var activities []TicketActivity
	bc.db.Where("ticket_id = ? AND activity_group_id != '' AND status IN ?",
		bc.ticket.ID, []string{"pending", "in_progress"}).
		Find(&activities)

	if len(activities) == 0 {
		return fmt.Errorf("no pending parallel activities found for ticket %d", bc.ticket.ID)
	}

	// Find the activity assigned to the target position.
	var targetActivity *TicketActivity
	var targetAssignment TicketAssignment
	orgSvc := &testOrgService{db: bc.db}

	for i := range activities {
		var assignment TicketAssignment
		if err := bc.db.Where("activity_id = ?", activities[i].ID).First(&assignment).Error; err != nil {
			continue
		}

		// Match by direct PositionID.
		if assignment.PositionID != nil {
			for code, pos := range bc.positions {
				if pos.ID == *assignment.PositionID && code == positionCode {
					targetActivity = &activities[i]
					targetAssignment = assignment
					break
				}
			}
		}

		// Match by assignee belonging to the position.
		if targetActivity == nil {
			var userID uint
			if assignment.AssigneeID != nil {
				userID = *assignment.AssigneeID
			} else if assignment.UserID != nil {
				userID = *assignment.UserID
			}
			if userID > 0 {
				for _, dept := range bc.departments {
					userIDs, _ := orgSvc.FindUsersByPositionAndDepartment(positionCode, dept.Code)
					for _, uid := range userIDs {
						if uid == userID {
							targetActivity = &activities[i]
							targetAssignment = assignment
							break
						}
					}
					if targetActivity != nil {
						break
					}
				}
			}
		}

		if targetActivity != nil {
			break
		}
	}

	if targetActivity == nil {
		return fmt.Errorf("no pending parallel activity found for position %q in ticket %d", positionCode, bc.ticket.ID)
	}

	// Resolve operator from assignment.
	var operatorID uint
	if targetAssignment.AssigneeID != nil {
		operatorID = *targetAssignment.AssigneeID
	} else if targetAssignment.UserID != nil {
		operatorID = *targetAssignment.UserID
	}
	if operatorID == 0 {
		operatorID = bc.findFallbackOperator()
	}

	// Claim the activity.
	bc.db.Model(&TicketAssignment{}).
		Where("activity_id = ?", targetActivity.ID).
		Updates(map[string]any{"assignee_id": operatorID, "status": "claimed"})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Progress with outcome="approved" (approve-type activity semantic).
	err := bc.smartEngine.Progress(ctx, bc.db, engine.ProgressParams{
		TicketID:   bc.ticket.ID,
		ActivityID: targetActivity.ID,
		Outcome:    "approved",
		OperatorID: operatorID,
	})
	if err != nil {
		bc.lastErr = err
		log.Printf("parallel approval progress for %s: %v", positionCode, err)
	}

	bc.db.First(bc.ticket, bc.ticket.ID)
	return nil
}

// whenCurrentActivityApproved approves the current single-sign activity via SmartEngine.Progress().
func (bc *bddContext) whenCurrentActivityApproved() error {
	return bc.progressCurrentActivity("approved", "")
}

// --- Then steps ---

func (bc *bddContext) thenParallelApprovalGroupExists(expectedCount int) error {
	var activities []TicketActivity
	bc.db.Where("ticket_id = ? AND activity_group_id != ''", bc.ticket.ID).
		Find(&activities)

	if len(activities) == 0 {
		return fmt.Errorf("no parallel activities found for ticket %d", bc.ticket.ID)
	}

	groupID := activities[0].ActivityGroupID
	for _, a := range activities {
		if a.ActivityGroupID != groupID {
			return fmt.Errorf("activities have different group IDs: %q vs %q", groupID, a.ActivityGroupID)
		}
	}

	if len(activities) != expectedCount {
		return fmt.Errorf("expected %d parallel activities, got %d", expectedCount, len(activities))
	}

	return nil
}

func (bc *bddContext) thenParallelApprovalGroupNotConverged() error {
	var pendingCount int64
	bc.db.Model(&TicketActivity{}).
		Where("ticket_id = ? AND activity_group_id != '' AND status NOT IN ?",
			bc.ticket.ID, engine.CompletedActivityStatuses()).
		Count(&pendingCount)

	if pendingCount == 0 {
		return fmt.Errorf("expected pending parallel activities but all are completed")
	}

	// Verify no non-parallel activities were prematurely created.
	var nonParallelPending int64
	bc.db.Model(&TicketActivity{}).
		Where("ticket_id = ? AND (activity_group_id = '' OR activity_group_id IS NULL) AND status IN ?",
			bc.ticket.ID, []string{"pending", "in_progress"}).
		Count(&nonParallelPending)

	if nonParallelPending > 0 {
		return fmt.Errorf("found %d non-parallel pending activities — premature next step triggered", nonParallelPending)
	}

	return nil
}

func (bc *bddContext) thenParallelApprovalGroupConverged() error {
	var pendingCount int64
	bc.db.Model(&TicketActivity{}).
		Where("ticket_id = ? AND activity_group_id != '' AND status NOT IN ?",
			bc.ticket.ID, engine.CompletedActivityStatuses()).
		Count(&pendingCount)

	if pendingCount > 0 {
		return fmt.Errorf("expected all parallel activities completed, but %d still pending", pendingCount)
	}

	return nil
}

func (bc *bddContext) thenNoApprovalActivityForPosition(positionCode string) error {
	pos, ok := bc.positions[positionCode]
	if !ok {
		return fmt.Errorf("position %q not in context", positionCode)
	}

	var count int64
	bc.db.Model(&TicketAssignment{}).
		Joins("JOIN itsm_ticket_activities ON itsm_ticket_activities.id = itsm_ticket_assignments.activity_id").
		Where("itsm_ticket_assignments.ticket_id = ? AND itsm_ticket_assignments.position_id = ? AND itsm_ticket_activities.status IN ? AND (itsm_ticket_activities.activity_group_id = '' OR itsm_ticket_activities.activity_group_id IS NULL)",
			bc.ticket.ID, pos.ID, []string{"pending", "in_progress"}).
		Count(&count)

	if count > 0 {
		return fmt.Errorf("found %d pending activities for position %q — premature creation", count, positionCode)
	}

	return nil
}
