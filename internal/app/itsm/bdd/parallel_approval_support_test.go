package bdd

// parallel_approval_support_test.go — Multi-role parallel approval workflow LLM generation
// and service publish helpers for BDD tests.

import (
	"context"
	"encoding/json"
	"fmt"
	. "metis/internal/app/itsm/definition"
	. "metis/internal/app/itsm/domain"
	"time"

	ai "metis/internal/app/ai/runtime"
	"metis/internal/app/itsm/engine"
	"metis/internal/llm"
)

// parallelApprovalCollaborationSpec mirrors the seed.go built-in service spec.
// Matches natural language style; generation context provides technical mapping hints.
const parallelApprovalCollaborationSpec = `员工在 IT 服务台提交多角色并签申请时，服务台需要确认申请标题、目标系统、时间窗口、申请原因和期望结果。
申请提交后，由信息部网络管理员和安全管理员同时完成审批；两人均完成审批后，工单汇聚到信息部运维管理员进行最终审批。
最终审批通过后流程结束。`

// parallelApprovalGenerationContext provides org structure and workflow hints for LLM generation.
const parallelApprovalGenerationContext = `
## 已有申请表单契约
该服务已经配置申请确认表单。生成参考路径时必须复用这些字段 key、类型和选项值；引用表单字段时必须使用 form.<key>。

- 申请标题: key=` + "`title`" + `, type=` + "`text`" + `
- 目标系统: key=` + "`target_system`" + `, type=` + "`text`" + `
- 时间窗口: key=` + "`time_window`" + `, type=` + "`date_range`" + `
- 申请原因: key=` + "`reason`" + `, type=` + "`textarea`" + `
- 期望结果: key=` + "`expected_result`" + `, type=` + "`textarea`" + `

## 按需查询到的组织上下文
以下组织结构映射来自本次按需工具查询。生成人工审批节点参与人时，特定部门中的特定岗位使用 position_department，并填入 department_code 与 position_code；不要输出具体用户。

- 参与人解析：department_hint=` + "`信息部`" + `, position_hint=` + "`网络管理员`" + `
  - 候选：type=` + "`position_department`" + `, department_code=` + "`it`" + `（信息部）, position_code=` + "`network_admin`" + `（网络管理员）, candidate_count=1
- 参与人解析：department_hint=` + "`信息部`" + `, position_hint=` + "`安全管理员`" + `
  - 候选：type=` + "`position_department`" + `, department_code=` + "`it`" + `（信息部）, position_code=` + "`security_admin`" + `（安全管理员）, candidate_count=1
- 参与人解析：department_hint=` + "`信息部`" + `, position_hint=` + "`运维管理员`" + `
  - 候选：type=` + "`position_department`" + `, department_code=` + "`it`" + `（信息部）, position_code=` + "`ops_admin`" + `（运维管理员）, candidate_count=1

## 多角色并签审批约束
协作规范要求并行审批：先由网络管理员和安全管理员同时（并行）审批，全部通过后汇聚到运维管理员进行最终审批。
- 并行拆分必须使用 type="parallel"，data.gateway_direction="fork"；并行汇聚必须使用 type="parallel"，data.gateway_direction="join"；不要使用 exclusive 作为汇聚节点。
- 所有人工审批节点必须使用 type="approve"；禁止写成 type="process" 或 type="action"。
- 并行审批时必须使用 execution_mode: "parallel"，在 activities 中同时列出网络管理员和安全管理员，participant_type 使用 position_department。
- 协作规范没有定义驳回后返工路径；每个审批节点的 rejected 出边应指向公共驳回结束节点，不能退回申请人补充。
- 最终审批通过后直接结束流程，不需要额外生成取消分支。
`

// parallelApprovalStaticWorkflowJSON is the reference path used for dialog (non-LLM) tests.
const parallelApprovalStaticWorkflowJSON = `{"nodes":[{"id":"start","type":"start","position":{"x":400,"y":50},"data":{"label":"开始","nodeType":"start"}},{"id":"intake","type":"form","position":{"x":400,"y":200},"data":{"label":"填写并签申请","nodeType":"form","participants":[{"type":"requester"}],"formSchema":{"fields":[{"key":"title","type":"text","label":"申请标题"},{"key":"target_system","type":"text","label":"目标系统"},{"key":"time_window","type":"date_range","label":"时间窗口"},{"key":"reason","type":"textarea","label":"申请原因"},{"key":"expected_result","type":"textarea","label":"期望结果"}]}}},{"id":"parallel_fork","type":"parallel","position":{"x":400,"y":400},"data":{"label":"并签拆分","nodeType":"parallel","gateway_direction":"fork"}},{"id":"approve_network","type":"approve","position":{"x":180,"y":600},"data":{"label":"网络管理员审批","nodeType":"approve","participants":[{"type":"position_department","department_code":"it","position_code":"network_admin"}]}},{"id":"approve_security","type":"approve","position":{"x":620,"y":600},"data":{"label":"安全管理员审批","nodeType":"approve","participants":[{"type":"position_department","department_code":"it","position_code":"security_admin"}]}},{"id":"parallel_join","type":"parallel","position":{"x":400,"y":800},"data":{"label":"并签汇聚","nodeType":"parallel","gateway_direction":"join"}},{"id":"approve_ops","type":"approve","position":{"x":400,"y":1000},"data":{"label":"运维管理员最终审批","nodeType":"approve","participants":[{"type":"position_department","department_code":"it","position_code":"ops_admin"}]}},{"id":"end_completed","type":"end","position":{"x":400,"y":1200},"data":{"label":"审批完成","nodeType":"end"}},{"id":"end_rejected","type":"end","position":{"x":700,"y":900},"data":{"label":"审批驳回","nodeType":"end"}}],"edges":[{"id":"e1","source":"start","target":"intake"},{"id":"e2","source":"intake","target":"parallel_fork"},{"id":"e3","source":"parallel_fork","target":"approve_network"},{"id":"e4","source":"parallel_fork","target":"approve_security"},{"id":"e5","source":"approve_network","target":"parallel_join","data":{"outcome":"approved"}},{"id":"e6","source":"approve_network","target":"end_rejected","data":{"outcome":"rejected"}},{"id":"e7","source":"approve_security","target":"parallel_join","data":{"outcome":"approved"}},{"id":"e8","source":"approve_security","target":"end_rejected","data":{"outcome":"rejected"}},{"id":"e9","source":"parallel_join","target":"approve_ops"},{"id":"e10","source":"approve_ops","target":"end_completed","data":{"outcome":"approved"}},{"id":"e11","source":"approve_ops","target":"end_rejected","data":{"outcome":"rejected"}}]}`

// parallelApprovalCasePayload defines test data for a parallel approval BDD scenario.
type parallelApprovalCasePayload struct {
	Summary  string
	FormData map[string]any
}

// parallelApprovalCasePayloads provides test data for parallel approval BDD scenarios.
var parallelApprovalCasePayloads = map[string]parallelApprovalCasePayload{
	"standard": {
		Summary: "多角色并签申请：防火墙策略变更，需要网络管理员和安全管理员同时审批。",
		FormData: map[string]any{
			"title":           "防火墙策略变更申请",
			"target_system":   "prod-firewall-01",
			"time_window":     []string{"2026-05-10 22:00", "2026-05-10 23:00"},
			"reason":          "需要调整防火墙策略以支持新的微服务通信，涉及网络和安全双重审批。",
			"expected_result": "允许 10.0.1.0/24 网段访问 10.0.2.0/24 的 8443 端口",
		},
	},
}

// generateParallelApprovalWorkflow calls the LLM to generate a parallel approval workflow JSON.
// Validates with ValidateWorkflow and retries on blocking errors.
func generateParallelApprovalWorkflow(cfg llmConfig) (json.RawMessage, error) {
	client, err := llm.NewClient(llm.ProtocolOpenAI, cfg.baseURL, cfg.apiKey)
	if err != nil {
		return nil, fmt.Errorf("create LLM client: %w", err)
	}

	svc := &WorkflowGenerateService{}
	maxRetries := 3

	var lastErrors []engine.ValidationError

	for attempt := 0; attempt <= maxRetries; attempt++ {
		userMsg := svc.BuildUserMessage(parallelApprovalCollaborationSpec, parallelApprovalGenerationContext, lastErrors)

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		resp, err := client.Chat(ctx, llm.ChatRequest{
			Model: cfg.model,
			Messages: []llm.Message{
				{Role: llm.RoleSystem, Content: PathBuilderSystemPrompt},
				{Role: llm.RoleUser, Content: userMsg},
			},
			MaxTokens: 4096,
		})
		cancel()

		if err != nil {
			if attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("LLM call failed after %d attempts: %w", attempt+1, err)
		}

		workflowJSON, extractErr := ExtractJSON(resp.Content)
		if extractErr != nil {
			lastErrors = []engine.ValidationError{
				{Message: fmt.Sprintf("输出不是有效 JSON: %v", extractErr)},
			}
			if attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("JSON extraction failed after %d attempts: %w", attempt+1, extractErr)
		}

		validationErrors := engine.ValidateWorkflow(workflowJSON)
		var blockingErrors []engine.ValidationError
		for _, e := range validationErrors {
			if !e.IsWarning() {
				blockingErrors = append(blockingErrors, e)
			}
		}

		if len(blockingErrors) == 0 {
			return workflowJSON, nil
		}

		lastErrors = blockingErrors
		if attempt < maxRetries {
			continue
		}

		return nil, fmt.Errorf("workflow validation failed after %d attempts: %v", attempt+1, blockingErrors)
	}

	return nil, fmt.Errorf("workflow generation failed")
}

// publishParallelApprovalSmartService creates the full service for parallel approval BDD lifecycle tests.
// Uses LLM to generate workflow JSON from the collaboration spec.
func publishParallelApprovalSmartService(bc *bddContext) error {
	// 1. Generate workflow via LLM (tests: spec→参考路径, 健康校验可发布)
	workflowJSON, err := generateParallelApprovalWorkflow(bc.llmCfg)
	if err != nil {
		return fmt.Errorf("generate parallel approval workflow: %w", err)
	}

	// 2. ServiceCatalog
	catalog := &ServiceCatalog{
		Name:     "安全与合规服务",
		Code:     "security-compliance-pa",
		IsActive: true,
	}
	if err := bc.db.Create(catalog).Error; err != nil {
		return fmt.Errorf("create catalog: %w", err)
	}

	// 3. Priority
	priority := &Priority{
		Name:     "普通",
		Code:     "normal-pa",
		Value:    3,
		Color:    "#52c41a",
		IsActive: true,
	}
	if err := bc.db.Create(priority).Error; err != nil {
		return fmt.Errorf("create priority: %w", err)
	}
	bc.priority = priority

	// 4. Decision agent
	agent := &ai.Agent{
		Name:         "流程决策智能体",
		Type:         "assistant",
		IsActive:     true,
		Visibility:   "private",
		Strategy:     "react",
		SystemPrompt: decisionAgentSystemPrompt,
		MaxTokens:    2048,
		MaxTurns:     1,
		CreatedBy:    1,
	}
	if err := bc.db.Create(agent).Error; err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	bc.db.Model(agent).Update("temperature", 0)

	// 5. ServiceDefinition (smart engine)
	svc := &ServiceDefinition{
		Name:              "多角色并签申请",
		Code:              "multi-role-parallel-approval-bdd",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		WorkflowJSON:      JSONField(workflowJSON),
		CollaborationSpec: parallelApprovalCollaborationSpec,
		AgentID:           &agent.ID,
		IsActive:          true,
	}
	if err := bc.db.Create(svc).Error; err != nil {
		return fmt.Errorf("create service definition: %w", err)
	}
	bc.service = svc

	return nil
}

// publishParallelApprovalDialogService creates the service with a static workflow for dialog tests.
// Does not require LLM — tests service matching via the service desk agent.
func publishParallelApprovalDialogService(bc *bddContext) error {
	catalog := &ServiceCatalog{
		Name:     "安全与合规服务（对话测试）",
		Code:     "security-compliance-pa-dialog",
		IsActive: true,
	}
	if err := bc.db.Create(catalog).Error; err != nil {
		return fmt.Errorf("create catalog: %w", err)
	}

	priority := &Priority{
		Name:     "普通",
		Code:     "normal-pa-dialog",
		Value:    3,
		Color:    "#52c41a",
		IsActive: true,
	}
	if err := bc.db.Create(priority).Error; err != nil {
		return fmt.Errorf("create priority: %w", err)
	}
	bc.priority = priority

	svc := &ServiceDefinition{
		Name:              "多角色并签申请",
		Code:              "multi-role-parallel-approval-dialog",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		WorkflowJSON:      JSONField([]byte(parallelApprovalStaticWorkflowJSON)),
		CollaborationSpec: parallelApprovalCollaborationSpec,
		IsActive:          true,
	}
	if err := bc.db.Create(svc).Error; err != nil {
		return fmt.Errorf("create service definition: %w", err)
	}
	bc.service = svc

	return nil
}
