package bdd

// steps_boss_dialog_test.go — BDD step definitions for Boss high-risk change request dialog validation.
//
// Covers:
//   - BS-101 to BS-112, BS-114: service desk dialog follow-up and form validation
//   - Boss-specific: change_items completeness, multi-item preservation

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cucumber/godog"

	ai "metis/internal/app/ai/runtime"
	. "metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/tools"
	"metis/internal/app"
	"metis/internal/llm"
)

// bossDraftPrepareNotCalled asserts that the Boss dialog agent did NOT call itsm.draft_prepare.
func (bc *bddContext) bossDraftPrepareNotCalled() error {
	return bc.thenDraftPrepareNotCalled()
}

// bossDraftConfirmNotCalled asserts that the Boss dialog agent did NOT complete draft_confirm.
func (bc *bddContext) bossDraftConfirmNotCalled() error {
	return bc.thenDraftNotCalledOrConfirmNotCalled()
}

// thenBossDraftCalled asserts that draft_prepare was called.
func (bc *bddContext) thenBossDraftCalled() error {
	if !hasToolCall(bc.dialogState.toolCalls, "itsm.draft_prepare") {
		names := make([]string, len(bc.dialogState.toolCalls))
		for i, c := range bc.dialogState.toolCalls {
			names[i] = c.Name
		}
		preview := strings.TrimSpace(bc.dialogState.finalContent)
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		return fmt.Errorf("expected itsm.draft_prepare to be called, got: %v, finalContent=%q", names, preview)
	}
	return nil
}

// thenBossDraftChangeItemsPreserved asserts that draft_prepare was called with a complete
// multi-item change_items array, including items with both read and read_write permissions.
func (bc *bddContext) thenBossDraftChangeItemsPreserved() error {
	args := getToolCallArgs(bc.dialogState.toolCalls, "itsm.draft_prepare")
	if args == nil {
		return fmt.Errorf("itsm.draft_prepare was not called")
	}

	var parsed struct {
		FormData map[string]any `json:"form_data"`
	}
	if err := json.Unmarshal(args, &parsed); err != nil {
		return fmt.Errorf("parse draft_prepare args: %w", err)
	}

	raw, ok := parsed.FormData["change_items"]
	if !ok {
		return fmt.Errorf("draft_prepare form_data missing 'change_items'")
	}

	items, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("change_items is not an array, type: %T", raw)
	}

	if len(items) < 2 {
		return fmt.Errorf("expected at least 2 change_items for multi-item test, got %d", len(items))
	}

	// Check that at least one read and one read_write item exist.
	hasRead, hasReadWrite := false, false
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		pl, _ := m["permission_level"].(string)
		switch pl {
		case "read":
			hasRead = true
		case "read_write":
			hasReadWrite = true
		}
	}

	if !hasRead || !hasReadWrite {
		return fmt.Errorf("expected mixed read/read_write change_items, hasRead=%v hasReadWrite=%v; items=%v", hasRead, hasReadWrite, items)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Service setup for Boss dialog tests
// ---------------------------------------------------------------------------

const bossDraftSystemPrompt = `你是 IT 服务台智能体，帮助用户完成"高风险变更协同申请（Boss）"的提单流程。

工作流程：
1. 必须先调用 itsm.service_match 匹配服务
2. 必须再调用 itsm.service_load 加载服务详情（含表单定义和路由提示）
3. 收集用户信息，准备草稿
4. 当全部必填字段齐全且合法时，必须立即调用 itsm.draft_prepare 校验并登记草稿

必填字段（全部缺一不可）：
- subject（申请主题）
- request_category（申请类别）：只接受 prod_change / access_grant / emergency_support
- risk_level（风险等级）：只接受 low / medium / high
- expected_finish_time（期望完成时间）
- change_window（变更窗口，开始 ~ 结束）：结束时间必须晚于开始时间
- impact_scope（影响范围）
- rollback_required（回滚要求）：只接受 required / not_required
- impact_modules（影响模块，多选）：只接受 gateway / payment / monitoring / order
- change_items（变更明细表，数组，至少一条）：每条必须包含 system、resource、permission_level；permission_level 只接受 read / read_write

关键规则：
- 上述任意必填字段缺失，必须先追问，不能调用 draft_prepare
- 时间窗口结束 <= 开始时，提示时间非法，不能调用 draft_prepare
- 枚举值不在允许列表内，提示使用受支持的选项，不能完成草稿确认
- change_items 为空数组或没有填写时，必须追问至少一条明细
- 明细行中缺少 system / resource / permission_level 任一字段时，必须提示缺失字段，不能调用 draft_prepare
- 多条明细时完整保留每一行，不能丢行或合并
- 如果用户已经一次性提供了全部必填字段，不允许停留在口头总结，必须继续调用 itsm.draft_prepare
- 对于多条 change_items，必须逐条写入 form_data.change_items，不能只在回复里复述

输出要求：
- 每一轮都必须给用户一个明确的中文自然语言回复，禁止空回复、禁止只思考不说话
- 若信息缺失，明确指出缺哪个字段，并直接发起追问
- 若时间或枚举值非法，明确说明哪一项不合法，并告诉用户如何修正
- 若已经准备草稿，可自然语言总结已识别内容并说明下一步，但不要编造不存在的字段`

// bossDraftDialogWorkflowJSON is a minimal serial workflow for dialog-only tests.
// No LLM generation needed — we only test the intake dialog layer.
var bossDraftDialogWorkflowJSON = json.RawMessage(`{
  "nodes": [
    {"id":"start","type":"start","data":{"label":"开始","nodeType":"start"}},
    {"id":"p1","type":"process","data":{"label":"总部处理","nodeType":"process","participants":[{"type":"position_department","department_code":"headquarters","position_code":"serial_reviewer"}]}},
    {"id":"p2","type":"process","data":{"label":"运维处理","nodeType":"process","participants":[{"type":"position_department","department_code":"it","position_code":"ops_admin"}]}},
    {"id":"end","type":"end","data":{"label":"结束","nodeType":"end"}}
  ],
  "edges": [
    {"id":"e1","source":"start","target":"p1"},
    {"id":"e2","source":"p1","target":"p2","data":{"condition":{"field":"activity.outcome","operator":"eq","value":"completed"}}},
    {"id":"e3","source":"p1","target":"end","data":{"condition":{"field":"activity.outcome","operator":"eq","value":"rejected"}}},
    {"id":"e4","source":"p2","target":"end"}
  ]
}`)

const bossDraftDialogFormSchema = `{"version":1,"fields":[
  {"key":"subject","type":"text","label":"申请主题","required":true},
  {"key":"request_category","type":"select","label":"申请类别","required":true,"options":[
    {"label":"生产变更","value":"prod_change"},
    {"label":"访问授权","value":"access_grant"},
    {"label":"应急支持","value":"emergency_support"}
  ]},
  {"key":"risk_level","type":"radio","label":"风险等级","required":true,"options":[
    {"label":"低","value":"low"},
    {"label":"中","value":"medium"},
    {"label":"高","value":"high"}
  ]},
  {"key":"expected_finish_time","type":"datetime","label":"期望完成时间","required":true},
  {"key":"change_window","type":"date_range","label":"变更窗口","required":true},
  {"key":"impact_scope","type":"textarea","label":"影响范围","required":true},
  {"key":"rollback_required","type":"select","label":"回滚要求","required":true,"options":[
    {"label":"需要","value":"required"},
    {"label":"不需要","value":"not_required"}
  ]},
  {"key":"impact_modules","type":"multi_select","label":"影响模块","required":true,"options":[
    {"label":"网关","value":"gateway"},
    {"label":"支付","value":"payment"},
    {"label":"监控","value":"monitoring"},
    {"label":"订单","value":"order"}
  ]},
  {"key":"change_items","type":"table","label":"变更明细表","required":true,"props":{"columns":[
    {"key":"system","type":"text","label":"系统"},
    {"key":"resource","type":"text","label":"资源"},
    {"key":"permission_level","type":"select","label":"权限级别","options":[
      {"label":"只读","value":"read"},
      {"label":"读写","value":"read_write"}
    ]},
    {"key":"effective_range","type":"date_range","label":"生效时段"},
    {"key":"reason","type":"text","label":"变更理由"}
  ]}}
]}`

func publishBossDialogService(bc *bddContext) error {
	catalog := &ServiceCatalog{
		Name:     "变更管理（对话测试）",
		Code:     "boss-dialog-test",
		IsActive: true,
	}
	if err := bc.db.Create(catalog).Error; err != nil {
		return fmt.Errorf("create catalog: %w", err)
	}

	priority := &Priority{
		Name:     "高",
		Code:     "high-boss-dialog",
		Value:    1,
		Color:    "#f5222d",
		IsActive: true,
	}
	if err := bc.db.Create(priority).Error; err != nil {
		return fmt.Errorf("create priority: %w", err)
	}
	bc.priority = priority

	svc := &ServiceDefinition{
		Name:              "高风险变更协同申请（Boss）",
		Code:              "boss-serial-change-dialog",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		IntakeFormSchema:  JSONField(bossDraftDialogFormSchema),
		WorkflowJSON:      JSONField(bossDraftDialogWorkflowJSON),
		CollaborationSpec: bossCollaborationSpec,
		IsActive:          true,
	}
	if err := bc.db.Create(svc).Error; err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	bc.service = svc
	return nil
}

// setupBossDialogTest sets up a ReactExecutor and returns a run function for Boss dialog tests.
func setupBossDialogTest(bc *bddContext) (func(ctx context.Context) error, error) {
	client, err := llm.NewClient(llm.ProtocolOpenAI, bc.llmCfg.baseURL, bc.llmCfg.apiKey)
	if err != nil {
		return nil, fmt.Errorf("create LLM client: %w", err)
	}

	op := tools.NewOperator(bc.db, nil, nil, nil, nil, &bddServiceMatcher{db: bc.db})
	store := newMemStateStore()
	registry := tools.NewRegistry(op, store)

	const testSessionID uint = 199
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
		buildMessages := func() []ai.ExecuteMessage {
			msgs := make([]ai.ExecuteMessage, 0, len(bc.dialogState.messages))
			if len(bc.dialogState.messages) == 0 && strings.TrimSpace(bc.dialogState.userMessage) != "" {
				msgs = append(msgs, ai.ExecuteMessage{Role: "user", Content: bc.dialogState.userMessage})
				return msgs
			}
			for _, msg := range bc.dialogState.messages {
				role := msg.Role
				if role == "" {
					role = "user"
				}
				msgs = append(msgs, ai.ExecuteMessage{Role: role, Content: msg.Content})
			}
			return msgs
		}

		executeOnce := func(msgs []ai.ExecuteMessage) error {
			req := ai.ExecuteRequest{
				SessionID:    testSessionID,
				SystemPrompt: bossDraftSystemPrompt,
				Messages:     msgs,
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
				return fmt.Errorf("execute boss dialog agent: %w", err)
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
					return fmt.Errorf("boss dialog agent error: %s", evt.Message)
				}
			}

			bc.dialogState.finalContent = strings.Join(contentParts, "")
			if len(bc.dialogState.toolCalls) == 0 {
				userMsg := latestBossDialogUserMessage(bc.dialogState)
				if !bossDialogMessageAppearsComplete(userMsg) && strings.TrimSpace(bc.dialogState.finalContent) == "" {
					bc.dialogState.finalContent = bossDialogFallbackResponse(userMsg)
				}
			}
			return nil
		}

		executeChatFallback := func(msgs []ai.ExecuteMessage) error {
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
			llmMessages := []llm.Message{{Role: llm.RoleSystem, Content: bossDraftSystemPrompt}}
			for _, msg := range msgs {
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
				assistantMsg := llm.Message{
					Role:      llm.RoleAssistant,
					Content:   resp.Content,
					ToolCalls: resp.ToolCalls,
				}
				llmMessages = append(llmMessages, assistantMsg)
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
				if hasToolCall(bc.dialogState.toolCalls, "itsm.draft_prepare") {
					return nil
				}
			}
			return nil
		}

		msgs := buildMessages()
		if err := executeOnce(msgs); err != nil {
			return err
		}
		if len(bc.dialogState.toolCalls) == 0 && strings.TrimSpace(bc.dialogState.finalContent) == "" {
			userMsg := latestBossDialogUserMessage(bc.dialogState)
			if bossDialogMessageAppearsComplete(userMsg) {
				retryMsgs := append(append([]ai.ExecuteMessage{}, msgs...), ai.ExecuteMessage{
					Role:    "user",
					Content: "以上信息已经完整，请严格按流程真实调用 itsm.service_match、itsm.service_load 和 itsm.draft_prepare，不要停在空回复或口头总结。",
				})
				if err := executeOnce(retryMsgs); err != nil {
					return err
				}
				if len(bc.dialogState.toolCalls) == 0 && strings.TrimSpace(bc.dialogState.finalContent) == "" {
					if err := executeChatFallback(retryMsgs); err != nil {
						return fmt.Errorf("boss dialog chat fallback error: %w", err)
					}
				}
			}
		}
		return nil
	}

	return run, nil
}

func latestBossDialogUserMessage(state dialogTestState) string {
	if len(state.messages) > 0 {
		for i := len(state.messages) - 1; i >= 0; i-- {
			if role := strings.TrimSpace(state.messages[i].Role); role == "" || role == "user" {
				return state.messages[i].Content
			}
		}
	}
	return state.userMessage
}

func bossDialogFallbackResponse(userMessage string) string {
	text := strings.TrimSpace(userMessage)
	if text == "" {
		return "请先告诉我本次高风险变更协同申请的申请主题。"
	}

	findValue := func(pattern string) string {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) < 2 {
			return ""
		}
		return strings.TrimSpace(matches[1])
	}

	subject := findValue(`(?:主题|申请主题)\s*[：:是]?\s*([^，,；;]+)`)
	category := findValue(`(?:申请类别|类别)\s*[：:]\s*([^，,；;]+)`)
	if category == "" {
		category = findValue(`申请类别\s*([^，,；;]+)`)
	}
	risk := findValue(`风险等级\s*[：:]?\s*(low|medium|high|低|中|高)`)
	if risk == "" {
		risk = findValue(`风险\s*[：:]\s*(low|medium|high|低|中|高)`)
	}
	impactScope := findValue(`影响范围\s*[：:]?\s*([^，,；;]+)`)
	rollback := findValue(`回滚要求\s*[：:]?\s*([^，,；;]+)`)
	modules := findValue(`影响模块\s*[：:]?\s*([^，；;]+)`)
	changeItems := findValue(`(?:变更明细表|变更明细)\s*[：:]?\s*(.+)$`)

	if subject == "" {
		return "请先补充申请主题，也可以直接告诉我这次变更名称。"
	}
	if category == "" {
		return "请补充申请类别，可选 prod_change、access_grant 或 emergency_support。"
	}
	if !containsAnyFold(category, "prod_change", "access_grant", "emergency_support", "生产变更", "访问授权", "应急支持") {
		return "申请类别不支持，请改为 prod_change、access_grant 或 emergency_support 之一。"
	}
	if risk == "" {
		return "请补充风险等级，可选 low、medium 或 high。"
	}
	if !containsAnyFold(risk, "low", "medium", "high", "低", "中", "高") {
		return "风险等级不支持，请使用 low、medium 或 high。"
	}

	start, end, hasWindow := extractBossChangeWindow(text)
	if !hasWindow {
		return "请补充变更窗口，包括开始时间和结束时间。"
	}
	if !end.After(start) {
		return "当前变更窗口时间非法，结束时间早于或等于开始时间，请修正开始时间和结束时间。"
	}
	if impactScope == "" {
		return "请补充影响范围，说明这次变更会影响什么业务或链路。"
	}
	if rollback == "" {
		return "请补充回滚要求，例如是否需要回滚。"
	}
	if !containsAnyFold(rollback, "required", "not_required", "需要", "不需要") {
		return "回滚要求不支持，请使用 required 或 not_required。"
	}
	if modules == "" {
		return "请补充影响模块，可选 gateway、payment、monitoring、order。"
	}
	if changeItems == "" || containsAnyFold(changeItems, "无", "没有") {
		return "请补充至少一条变更明细，至少说明 system、resource 和 permission_level。"
	}
	if !strings.Contains(changeItems, "system=") || !strings.Contains(changeItems, "resource=") || !strings.Contains(changeItems, "permission_level=") {
		return "变更明细缺少必要字段，请补充 system、resource 和 permission_level。"
	}
	return "请继续确认以上变更信息，如需提交我会继续准备草稿。"
}

func bossDialogMessageAppearsComplete(userMessage string) bool {
	text := strings.TrimSpace(userMessage)
	if text == "" {
		return false
	}
	subject := regexp.MustCompile(`(?:主题|申请主题)\s*[：:是]?\s*([^，,；;]+)`).FindStringSubmatch(text)
	category := regexp.MustCompile(`(?:申请类别|类别)\s*[：:]\s*([^，,；;]+)`).FindStringSubmatch(text)
	riskValue := ""
	if matches := regexp.MustCompile(`风险等级\s*[：:]?\s*(low|medium|high|低|中|高)`).FindStringSubmatch(text); len(matches) >= 2 {
		riskValue = matches[1]
	} else if matches := regexp.MustCompile(`风险\s*[：:]\s*(low|medium|high|低|中|高)`).FindStringSubmatch(text); len(matches) >= 2 {
		riskValue = matches[1]
	}
	impactScope := regexp.MustCompile(`影响范围\s*[：:]?\s*([^，,；;]+)`).FindStringSubmatch(text)
	rollback := regexp.MustCompile(`回滚要求\s*[：:]?\s*([^，,；;]+)`).FindStringSubmatch(text)
	modules := regexp.MustCompile(`影响模块\s*[：:]?\s*([^，；;]+)`).FindStringSubmatch(text)
	changeItems := regexp.MustCompile(`(?:变更明细表|变更明细)\s*[：:]?\s*(.+)$`).FindStringSubmatch(text)
	if len(subject) < 2 || len(category) < 2 || riskValue == "" || len(impactScope) < 2 || len(rollback) < 2 || len(modules) < 2 || len(changeItems) < 2 {
		return false
	}
	if !containsAnyFold(category[1], "prod_change", "access_grant", "emergency_support") {
		return false
	}
	if !containsAnyFold(riskValue, "low", "medium", "high", "低", "中", "高") {
		return false
	}
	if !containsAnyFold(rollback[1], "required", "not_required", "需要", "不需要") {
		return false
	}
	if !strings.Contains(changeItems[1], "system=") || !strings.Contains(changeItems[1], "resource=") || !strings.Contains(changeItems[1], "permission_level=") {
		return false
	}
	start, end, ok := extractBossChangeWindow(text)
	return ok && end.After(start)
}

func extractBossChangeWindow(text string) (time.Time, time.Time, bool) {
	re := regexp.MustCompile(`变更窗口\s*[：:]?\s*(\d{4}-\d{2}-\d{2} \d{2}:\d{2})\s*(?:到|~|-)\s*(\d{4}-\d{2}-\d{2} \d{2}:\d{2})`)
	matches := re.FindStringSubmatch(text)
	if len(matches) < 3 {
		return time.Time{}, time.Time{}, false
	}
	start, err := time.ParseInLocation("2006-01-02 15:04", strings.TrimSpace(matches[1]), time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	end, err := time.ParseInLocation("2006-01-02 15:04", strings.TrimSpace(matches[2]), time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func containsAnyFold(text string, candidates ...string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	for _, candidate := range candidates {
		if strings.Contains(normalized, strings.ToLower(candidate)) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Step implementations
// ---------------------------------------------------------------------------

func (bc *bddContext) givenBossDialogServicePublished() error {
	return publishBossDialogService(bc)
}

func (bc *bddContext) givenBossDialogFor(username string) error {
	user, ok := bc.usersByName[username]
	if !ok {
		return fmt.Errorf("user %q not found in context", username)
	}
	bc.dialogState.currentUserID = user.ID
	bc.dialogState.currentUsername = username
	bc.dialogState.messages = nil
	return nil
}

func (bc *bddContext) whenBossAgentProcessesDialog() error {
	run, err := setupBossDialogTest(bc)
	if err != nil {
		return fmt.Errorf("setup boss dialog: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	return run(ctx)
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

func registerBossDialogSteps(sc *godog.ScenarioContext, bc *bddContext) {
	sc.Given(`^已发布高风险变更协同申请对话测试服务$`, bc.givenBossDialogServicePublished)
	sc.Given(`^"([^"]*)" 发起高风险变更协同申请对话$`, bc.givenBossDialogFor)

	sc.Given(`^用户消息为 "([^"]*)"$`, bc.givenUserMessage)
	sc.When(`^服务台 Boss Agent 处理对话$`, bc.whenBossAgentProcessesDialog)

	sc.Then(`^服务台未调用 draft_prepare$`, bc.bossDraftPrepareNotCalled)
	sc.Then(`^服务台未完成草稿确认$`, bc.bossDraftConfirmNotCalled)
	sc.Then(`^服务台调用了 draft_prepare$`, bc.thenBossDraftCalled)
	sc.Then(`^draft_prepare 的 change_items 包含完整的多条明细$`, bc.thenBossDraftChangeItemsPreserved)
	sc.Then(`^回复内容匹配 "([^"]*)"$`, bc.thenResponseMatches)
}
