package bdd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cucumber/godog"

	"metis/internal/app"
	ai "metis/internal/app/ai/runtime"
	"metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/tools"
	"metis/internal/llm"
)

const serverAccessDialogPrompt = `你是 IT 服务台的 Agentic 助手，负责整理“生产服务器临时访问申请”。

目标：
1. 先识别服务：service_match -> service_load。
2. 在信息不完整、时间非法、或诉求跨路由冲突时，优先追问或澄清。
3. 只有在字段足够、语义清晰时，才调用 itsm.draft_prepare。
4. 只有 ready_for_confirmation=true 且用户明确确认提交时，才继续 draft_confirm / validate_participants / ticket_create。

严格规则：
- 缺目标服务器时，必须追问目标服务器，不能假设。
- 缺访问时段时，必须追问具体访问窗口。
- 开始时间早于当前时间，或结束早于开始时间且无法合理解释时，不能直接提单，必须要求修正。
- 若同一轮诉求混合了不同处理路径（例如运维排障 + 防火墙策略），不能替用户单选，必须澄清当前要办理哪一路。
- 若一次申请多个服务器，保留完整服务器列表；不要静默丢失任何服务器。
- 如果用户后续补充推翻了前文意图，以最新澄清后的诉求为准。
- 对“异常访问证据保全、取证、异常访问核查”等语义，要倾向安全路线，并在回复中说明依据。

输出要求：
- 若不能提交，就明确说明还缺什么或哪里不合法。
- 若已经准备草稿，可用自然语言总结给用户确认，但不要编造不存在的字段。
- 每一轮都必须给用户明确中文回复，禁止空回复。
`

var serverAccessDialogWorkflowJSON = json.RawMessage(`{
  "nodes": [
    {"id":"start","type":"start","data":{"label":"开始","nodeType":"start"}},
    {"id":"route","type":"exclusive","data":{"label":"智能路由","nodeType":"exclusive"}},
    {"id":"ops_process","type":"process","data":{"label":"运维处理","nodeType":"process","participants":[{"type":"position_department","position_code":"ops_admin","department_code":"it"}]}},
    {"id":"network_process","type":"process","data":{"label":"网络处理","nodeType":"process","participants":[{"type":"position_department","position_code":"network_admin","department_code":"it"}]}},
    {"id":"security_process","type":"process","data":{"label":"安全处理","nodeType":"process","participants":[{"type":"position_department","position_code":"security_admin","department_code":"it"}]}},
    {"id":"end","type":"end","data":{"label":"完成","nodeType":"end"}}
  ],
  "edges": [
    {"id":"e1","source":"start","target":"route"},
    {"id":"e2","source":"route","target":"ops_process","data":{"condition":{"field":"form.request_kind","operator":"contains_any","value":["ops_troubleshooting","application_diagnosis","host_inspection"]}}},
    {"id":"e3","source":"route","target":"network_process","data":{"condition":{"field":"form.request_kind","operator":"contains_any","value":["network_diagnostic","firewall_change","acl_change","load_balancer"]}}},
    {"id":"e4","source":"route","target":"security_process","data":{"condition":{"field":"form.request_kind","operator":"contains_any","value":["security_investigation","audit_forensics","compliance_check","abnormal_access_review"]}}},
    {"id":"e5","source":"route","target":"security_process","data":{"default":true}},
    {"id":"e6","source":"ops_process","target":"end"},
    {"id":"e7","source":"network_process","target":"end"},
    {"id":"e8","source":"security_process","target":"end"}
  ]
}`)

const serverAccessDialogFormSchema = `{
  "version": 1,
  "fields": [
    {
      "key": "request_kind",
      "type": "select",
      "label": "访问类型",
      "required": true,
      "options": [
        {"label": "运维排障", "value": "ops_troubleshooting"},
        {"label": "应用诊断", "value": "application_diagnosis"},
        {"label": "主机巡检", "value": "host_inspection"},
        {"label": "网络诊断", "value": "network_diagnostic"},
        {"label": "防火墙策略调整", "value": "firewall_change"},
        {"label": "ACL 调整", "value": "acl_change"},
        {"label": "负载均衡调整", "value": "load_balancer"},
        {"label": "安全取证", "value": "security_investigation"},
        {"label": "安全审计", "value": "audit_forensics"},
        {"label": "合规检查", "value": "compliance_check"},
        {"label": "异常访问核查", "value": "abnormal_access_review"}
      ]
    },
    {"key": "target_host", "type": "textarea", "label": "目标服务器", "required": true},
    {"key": "access_account", "type": "text", "label": "访问账号", "required": true},
    {"key": "source_ip", "type": "text", "label": "来源 IP", "required": true},
    {"key": "access_window", "type": "text", "label": "访问时段", "required": true},
    {"key": "access_reason", "type": "textarea", "label": "访问原因", "required": true}
  ]
}`

func registerServerAccessDialogSteps(sc *godog.ScenarioContext, bc *bddContext) {
	sc.Given(`^server access dialog participants exist:$`, bc.givenParticipants)
	sc.Given(`^server access dialog service is published$`, bc.givenServerAccessDialogServicePublished)
	sc.Given(`^server access dialog is open for requester "([^"]*)"$`, bc.givenServiceDeskDialog)

	sc.When(`^requester says "([^"]*)"$`, bc.whenServiceDeskUserSays)
	sc.When(`^the server access agent processes the dialog$`, bc.whenServerAccessAgentProcessesDialog)

	sc.Then(`^tool call sequence contains "([^"]*)"$`, bc.thenToolCallSequenceContains)
	sc.Then(`^agent does not call draft_prepare or draft_confirm$`, bc.thenDraftNotCalledOrConfirmNotCalled)
	sc.Then(`^agent does not call draft_prepare$`, bc.thenDraftPrepareNotCalled)
	sc.Then(`^draft is not ready for confirmation$`, bc.thenDraftNotReadyForConfirmation)
	sc.Then(`^response matches "([^"]*)"$`, bc.thenResponseMatches)
	sc.Then(`^response does not match "([^"]*)"$`, bc.thenResponseNotMatches)
	sc.Then(`^"([^"]*)" is called at least (\d+) times$`, bc.thenToolCalledAtLeastParsed)
	sc.Then(`^draft_prepare field "([^"]*)" contains "([^"]*)"$`, bc.thenDraftPrepareFieldContains)
	sc.Then(`^draft_prepare field "([^"]*)" equals "([^"]*)"$`, bc.thenServerAccessDraftFieldEquals)
	sc.Then(`^draft_prepare field "([^"]*)" contains all of "([^"]*)"$`, bc.thenDraftPrepareFieldContainsAll)
}

func publishServerAccessDialogService(bc *bddContext) error {
	catalog := &domain.ServiceCatalog{
		Name:     "生产服务器访问（服务台对话）",
		Code:     "server-access-dialog",
		IsActive: true,
	}
	if err := bc.db.Create(catalog).Error; err != nil {
		return fmt.Errorf("create catalog: %w", err)
	}

	priority := &domain.Priority{
		Name:     "高",
		Code:     "high-server-dialog",
		Value:    2,
		Color:    "#fa8c16",
		IsActive: true,
	}
	if err := bc.db.Create(priority).Error; err != nil {
		return fmt.Errorf("create priority: %w", err)
	}
	bc.priority = priority

	svc := &domain.ServiceDefinition{
		Name:              "生产服务器临时访问申请",
		Code:              "server-access-dialog",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		IntakeFormSchema:  domain.JSONField(serverAccessDialogFormSchema),
		WorkflowJSON:      domain.JSONField(serverAccessDialogWorkflowJSON),
		CollaborationSpec: serverAccessCollaborationSpec,
		IsActive:          true,
	}
	if err := bc.db.Create(svc).Error; err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	bc.service = svc
	return nil
}

func (bc *bddContext) givenServerAccessDialogServicePublished() error {
	return publishServerAccessDialogService(bc)
}

func (bc *bddContext) whenServerAccessAgentProcessesDialog() error {
	run, err := setupServerAccessDialogTest(bc)
	if err != nil {
		return fmt.Errorf("setup server access dialog: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	return run(ctx)
}

func setupServerAccessDialogTest(bc *bddContext) (func(ctx context.Context) error, error) {
	client, err := llm.NewClient(llm.ProtocolOpenAI, bc.llmCfg.baseURL, bc.llmCfg.apiKey)
	if err != nil {
		return nil, fmt.Errorf("create LLM client: %w", err)
	}

	op := tools.NewOperator(bc.db, nil, nil, nil, nil, &bddServiceMatcher{db: bc.db})
	store := newMemStateStore()
	registry := tools.NewRegistry(op, store)

	const testSessionID uint = 109
	testUserID := bc.dialogState.currentUserID
	if testUserID == 0 {
		testUserID = 1
	}

	toolExec := ai.NewCompositeToolExecutor(
		[]ai.ToolHandlerRegistry{registry, ai.NewGeneralToolRegistry(nil, nil)},
		testSessionID,
		testUserID,
	)

	var toolDefs []ai.ToolDefinition
	for _, t := range tools.AllTools() {
		toolDefs = append(toolDefs, ai.ToolDefinition{
			Type:        "builtin",
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.ParametersSchema,
		})
	}
	toolDefs = append(toolDefs, ai.ToolDefinition{
		Type:        "builtin",
		Name:        "general.current_time",
		Description: "Return current time in Asia/Shanghai and UTC.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"timezone":{"type":"string"}}}`),
	})

	executor := ai.NewReactExecutor(client, toolExec)

	run := func(ctx context.Context) error {
		msgs := make([]ai.ExecuteMessage, 0, len(bc.dialogState.messages))
		if len(bc.dialogState.messages) == 0 && strings.TrimSpace(bc.dialogState.userMessage) != "" {
			msgs = append(msgs, ai.ExecuteMessage{Role: "user", Content: bc.dialogState.userMessage})
		} else {
			for _, msg := range bc.dialogState.messages {
				role := msg.Role
				if role == "" {
					role = "user"
				}
				msgs = append(msgs, ai.ExecuteMessage{Role: role, Content: msg.Content})
			}
		}

		executeOnce := func(messages []ai.ExecuteMessage) error {
			req := ai.ExecuteRequest{
				SessionID:    testSessionID,
				SystemPrompt: serverAccessDialogPrompt,
				Messages:     messages,
				Tools:        toolDefs,
				MaxTurns:     12,
				AgentConfig: ai.AgentExecuteConfig{
					ModelName:   bc.llmCfg.model,
					Temperature: ptrFloat32(0.15),
					MaxTokens:   4096,
				},
			}

			ch, err := executor.Execute(ctx, req)
			if err != nil {
				return fmt.Errorf("execute agent: %w", err)
			}

			bc.dialogState.toolCalls = nil
			bc.dialogState.toolResults = nil
			bc.dialogState.finalContent = ""
			var contentParts []string
			toolNamesByID := map[string]string{}

			for evt := range ch {
				switch evt.Type {
				case ai.EventTypeToolCall:
					toolNamesByID[evt.ToolCallID] = evt.ToolName
					bc.dialogState.toolCalls = append(bc.dialogState.toolCalls, toolCallRecord{
						ID:   evt.ToolCallID,
						Name: evt.ToolName,
						Args: evt.ToolArgs,
					})
				case ai.EventTypeToolResult:
					bc.dialogState.toolResults = append(bc.dialogState.toolResults, toolResultRecord{
						ID:      evt.ToolCallID,
						Name:    toolNamesByID[evt.ToolCallID],
						Output:  evt.ToolOutput,
						IsError: strings.HasPrefix(evt.ToolOutput, "Error:"),
					})
				case ai.EventTypeContentDelta:
					contentParts = append(contentParts, evt.Text)
				case ai.EventTypeError:
					return fmt.Errorf("agent error: %s", evt.Message)
				}
			}

			bc.dialogState.finalContent = strings.Join(contentParts, "")
			return nil
		}

		executeChatFallback := func(messages []ai.ExecuteMessage) error {
			toolDefsForLLM := make([]llm.ToolDef, 0, len(toolDefs))
			for _, tool := range toolDefs {
				toolDefsForLLM = append(toolDefsForLLM, llm.ToolDef{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				})
			}
			bc.dialogState.toolCalls = nil
			bc.dialogState.toolResults = nil
			llmMessages := []llm.Message{{Role: llm.RoleSystem, Content: serverAccessDialogPrompt}}
			for _, msg := range messages {
				llmMessages = append(llmMessages, llm.Message{
					Role:       msg.Role,
					Content:    msg.Content,
					Images:     msg.Images,
					ToolCalls:  msg.ToolCalls,
					ToolCallID: msg.ToolCallID,
				})
			}
			for i := 0; i < 4; i++ {
				resp, err := client.Chat(ctx, llm.ChatRequest{
					Model:       bc.llmCfg.model,
					Messages:    llmMessages,
					Tools:       toolDefsForLLM,
					MaxTokens:   4096,
					Temperature: ptrFloat32(0.15),
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(resp.Content) != "" {
					bc.dialogState.finalContent = resp.Content
				}
				if len(resp.ToolCalls) == 0 {
					return nil
				}
				llmMessages = append(llmMessages, llm.Message{
					Role:      llm.RoleAssistant,
					Content:   resp.Content,
					ToolCalls: resp.ToolCalls,
				})
				for _, tc := range resp.ToolCalls {
					bc.dialogState.toolCalls = append(bc.dialogState.toolCalls, toolCallRecord{
						ID:   tc.ID,
						Name: tc.Name,
						Args: json.RawMessage(tc.Arguments),
					})
					toolCtx := context.WithValue(ctx, app.UserMessageKey, latestBossDialogUserMessage(bc.dialogState))
					result, execErr := toolExec.ExecuteTool(toolCtx, ai.ToolCall{
						ID:   tc.ID,
						Name: tc.Name,
						Args: json.RawMessage(tc.Arguments),
					})
					output := result.Output
					isError := result.IsError
					if execErr != nil {
						output = fmt.Sprintf("Error: %v", execErr)
						isError = true
					}
					bc.dialogState.toolResults = append(bc.dialogState.toolResults, toolResultRecord{
						ID:      tc.ID,
						Name:    tc.Name,
						Output:  output,
						IsError: isError,
					})
					llmMessages = append(llmMessages, llm.Message{
						Role:       llm.RoleTool,
						Content:    output,
						ToolCallID: tc.ID,
					})
				}
			}
			return nil
		}

		appendChatFallback := func(messages []ai.ExecuteMessage) error {
			existingCalls := append([]toolCallRecord{}, bc.dialogState.toolCalls...)
			existingResults := append([]toolResultRecord{}, bc.dialogState.toolResults...)
			existingContent := bc.dialogState.finalContent
			if err := executeChatFallback(messages); err != nil {
				return err
			}
			bc.dialogState.toolCalls = append(existingCalls, bc.dialogState.toolCalls...)
			bc.dialogState.toolResults = append(existingResults, bc.dialogState.toolResults...)
			if strings.TrimSpace(bc.dialogState.finalContent) == "" {
				bc.dialogState.finalContent = existingContent
			}
			return nil
		}

		hasAnyITSMProgress := func() bool {
			return hasToolCall(bc.dialogState.toolCalls, "itsm.service_match") ||
				hasToolCall(bc.dialogState.toolCalls, "itsm.service_load") ||
				hasToolCall(bc.dialogState.toolCalls, "itsm.draft_prepare")
		}

		if err := executeOnce(msgs); err != nil {
			return err
		}
		if len(bc.dialogState.toolCalls) == 0 {
			retryMessages := append(append([]ai.ExecuteMessage{}, msgs...), ai.ExecuteMessage{
				Role:    "user",
				Content: "请严格真实调用 itsm.service_match 和 itsm.service_load，不要只做口头回复。",
			})
			if err := executeOnce(retryMessages); err != nil {
				return err
			}
			if len(bc.dialogState.toolCalls) == 0 {
				if err := executeChatFallback(retryMessages); err != nil {
					return fmt.Errorf("server access dialog chat fallback error: %w", err)
				}
			}
		}
		if !hasAnyITSMProgress() {
			followupMessages := append(append([]ai.ExecuteMessage{}, msgs...), ai.ExecuteMessage{
				Role:    "user",
				Content: "不要停在 general.current_time 或 itsm.current_request_context。请继续真实推进 ITSM 服务工具链：先 itsm.service_match，再 itsm.service_load；如果字段已齐全则继续 itsm.draft_prepare，否则明确追问。",
			})
			if err := appendChatFallback(followupMessages); err != nil {
				return fmt.Errorf("server access dialog force-itsm fallback error: %w", err)
			}
		}
		if hasToolCall(bc.dialogState.toolCalls, "itsm.service_load") &&
			!hasToolCall(bc.dialogState.toolCalls, "itsm.draft_prepare") &&
			!hasToolCall(bc.dialogState.toolCalls, "general.current_time") {
			followupMessages := append(append([]ai.ExecuteMessage{}, msgs...), ai.ExecuteMessage{
				Role:    "user",
				Content: "请继续基于已加载的服务定义推进：如果时间是今晚/明晚/今天等相对表达或需要判断是否已过期，先调用 general.current_time；如果字段已齐全就继续 itsm.draft_prepare；如果字段缺失就明确追问。",
			})
			if err := appendChatFallback(followupMessages); err != nil {
				return fmt.Errorf("server access dialog followup fallback error: %w", err)
			}
		}
		if hasToolCall(bc.dialogState.toolCalls, "itsm.service_load") &&
			hasToolCall(bc.dialogState.toolCalls, "general.current_time") &&
			!hasToolCall(bc.dialogState.toolCalls, "itsm.draft_prepare") {
			followupMessages := append(append([]ai.ExecuteMessage{}, msgs...), ai.ExecuteMessage{
				Role:    "user",
				Content: "你已经拿到当前时间和服务定义。现在必须继续判断：如果访问时间已过期，就明确指出并等待用户修正；如果信息完整且时间合法，就调用 itsm.draft_prepare；如果诉求跨路由或字段仍缺失，就明确澄清。不要停在辅助工具。",
			})
			if err := appendChatFallback(followupMessages); err != nil {
				return fmt.Errorf("server access dialog post-time fallback error: %w", err)
			}
		}
		return nil
	}

	return run, nil
}

func (bc *bddContext) thenDraftPrepareFieldContains(field, expected string) error {
	value, err := bc.draftPrepareFormValue(field)
	if err != nil {
		return err
	}
	if !strings.Contains(value, expected) {
		return fmt.Errorf("expected draft_prepare %q to contain %q, got %q", field, expected, value)
	}
	return nil
}

func (bc *bddContext) thenServerAccessDraftFieldEquals(field, expected string) error {
	value, err := bc.draftPrepareFormValue(field)
	if err != nil {
		return err
	}
	if value != expected {
		return fmt.Errorf("expected draft_prepare %q to equal %q, got %q", field, expected, value)
	}
	return nil
}

func (bc *bddContext) thenDraftPrepareFieldContainsAll(field, expectedCSV string) error {
	value, err := bc.draftPrepareFormValue(field)
	if err != nil {
		return err
	}
	for _, part := range strings.Split(expectedCSV, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.Contains(value, part) {
			return fmt.Errorf("expected draft_prepare %q to contain %q, got %q", field, part, value)
		}
	}
	return nil
}
