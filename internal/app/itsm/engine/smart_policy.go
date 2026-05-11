package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

type SmartDecisionPolicy interface {
	Apply(ctx context.Context, e *SmartEngine, tx *gorm.DB, ticketID uint, plan *DecisionPlan, svc *serviceModel) (bool, error)
}

type SmartDecisionPolicyFunc func(ctx context.Context, e *SmartEngine, tx *gorm.DB, ticketID uint, plan *DecisionPlan, svc *serviceModel) (bool, error)

func (f SmartDecisionPolicyFunc) Apply(ctx context.Context, e *SmartEngine, tx *gorm.DB, ticketID uint, plan *DecisionPlan, svc *serviceModel) (bool, error) {
	return f(ctx, e, tx, ticketID, plan, svc)
}

func builtInSmartDecisionPolicies() []SmartDecisionPolicy {
	return []SmartDecisionPolicy{
		SmartDecisionPolicyFunc(dbBackupWhitelistPolicy),
		SmartDecisionPolicyFunc(bossSerialChangePolicy),
		SmartDecisionPolicyFunc(rejectedBranchTerminalPolicy),
		SmartDecisionPolicyFunc(accessPurposeRoutePolicy),
	}
}

func dbBackupWhitelistPolicy(ctx context.Context, e *SmartEngine, tx *gorm.DB, ticketID uint, plan *DecisionPlan, svc *serviceModel) (bool, error) {
	if !looksLikeDBBackupWhitelistSpec(svc.CollaborationSpec) {
		return false, nil
	}
	return true, e.applyDBBackupWhitelistGuard(ctx, tx, ticketID, plan, svc)
}

func bossSerialChangePolicy(ctx context.Context, e *SmartEngine, tx *gorm.DB, ticketID uint, plan *DecisionPlan, svc *serviceModel) (bool, error) {
	if !looksLikeBossSerialChangeSpec(svc.CollaborationSpec) {
		return false, nil
	}
	return true, e.applyBossSerialChangeGuard(tx, ticketID, plan)
}

func rejectedBranchTerminalPolicy(ctx context.Context, e *SmartEngine, tx *gorm.DB, ticketID uint, plan *DecisionPlan, svc *serviceModel) (bool, error) {
	if svc == nil || svc.WorkflowJSON == "" {
		return false, nil
	}

	completed, formData, ok, err := latestRejectedHumanBranchContext(tx, ticketID)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	insights := buildBranchInsights(svc.WorkflowJSON, svc.CollaborationSpec, formData, "", "", completed)
	contract, ok := insights["completion_contract"].(map[string]any)
	if !ok {
		return false, nil
	}
	canComplete, _ := contract["can_complete_after_rejection"].(bool)
	selected, _ := insights["selected_branch"].(map[string]any)
	branchRejectedTerminal, _ := selected["branch_rejected_terminal"].(bool)
	if !canComplete && !branchRejectedTerminal {
		return false, nil
	}

	targetLabel := fmt.Sprint(contract["rejected_target_label"])
	if targetLabel == "" || targetLabel == "<nil>" {
		targetLabel = fmt.Sprint(selected["branch_label"])
	}
	if targetLabel == "" || targetLabel == "<nil>" {
		targetLabel = "当前分支驳回终态"
	}
	forceCompletePlan(plan, fmt.Sprintf("当前业务分支刚发生 rejected，分支闭环合同要求直接收敛到 %s。", targetLabel))
	return true, nil
}

func accessPurposeRoutePolicy(ctx context.Context, e *SmartEngine, tx *gorm.DB, ticketID uint, plan *DecisionPlan, svc *serviceModel) (bool, error) {
	expectedPosition, ok, err := collaborationSpecAccessPurposePosition(tx, ticketID, svc.CollaborationSpec)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if expectedPosition == "" {
		return true, fmt.Errorf("form.access_reason/form.operation_purpose 缺失、为空或未命中协作规范定义的访问原因分支；不得高置信结束或选择单一路由")
	}
	return true, e.applySingleHumanRouteGuard(tx, ticketID, plan, expectedPosition, "访问目的已命中协作规范岗位分支")
}

func latestRejectedHumanBranchContext(tx *gorm.DB, ticketID uint) (*activityModel, map[string]any, bool, error) {
	var ticket ticketModel
	if err := tx.Select("id, form_data").First(&ticket, ticketID).Error; err != nil {
		return nil, nil, false, err
	}

	var completed activityModel
	if err := tx.Where("ticket_id = ? AND status IN ?", ticketID, CompletedActivityStatuses()).
		Order("finished_at DESC, id DESC").
		First(&completed).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, false, nil
		}
		return nil, nil, false, err
	}
	if !isHumanActivityType(completed.ActivityType) || isPositiveActivityOutcome(completed.TransitionOutcome) {
		return nil, nil, false, nil
	}

	formData := map[string]any{}
	if strings := ticket.FormData; strings != "" {
		if err := json.Unmarshal([]byte(strings), &formData); err != nil {
			return nil, nil, false, fmt.Errorf("parse ticket form_data: %w", err)
		}
	}
	return &completed, formData, true, nil
}
