package bdd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"metis/internal/app"
	ai "metis/internal/app/ai/runtime"
	"metis/internal/app/itsm/tools"
	"metis/internal/llm"
)

const parallelApprovalDialogPrompt = `你是 IT 服务台智能体，帮助用户完成“多角色并签申请”的提单流程。

工作流程：
1. 必须先调用 itsm.service_match 匹配服务
2. 必须再调用 itsm.service_load 加载服务详情（含表单定义和审批协作规范）
3. 根据 service_load 返回的字段定义收集申请信息
4. 只有在 title、target_system、time_window、reason、expected_result 全部齐全且合法时，才允许调用 itsm.draft_prepare

关键规则：
- 用户提到“防火墙策略变更”“网络和安全团队同时审批”“多人审批”“并签”等语义时，应识别为多角色并签申请
- 缺少任一必填字段时，必须先追问缺失字段，不能跳过 service_match / service_load，也不能直接 draft_prepare
- time_window 必须是明确的开始和结束时间；如果用户没给完整时间窗口，必须追问
- 回复必须使用中文自然语言；禁止空回复、禁止只思考不说话
- 如果本轮信息已经足够，也要先真实调用 itsm.service_match 和 itsm.service_load，再决定是否 draft_prepare
`

func setupParallelApprovalDialogTest(bc *bddContext) (func(ctx context.Context) error, error) {
	client, err := llm.NewClient(llm.ProtocolOpenAI, bc.llmCfg.baseURL, bc.llmCfg.apiKey)
	if err != nil {
		return nil, fmt.Errorf("create LLM client: %w", err)
	}

	op := tools.NewOperator(bc.db, nil, nil, nil, nil, &bddServiceMatcher{db: bc.db})
	store := newMemStateStore()
	registry := tools.NewRegistry(op, store)

	const testSessionID uint = 209
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
				SystemPrompt: parallelApprovalDialogPrompt,
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
				return fmt.Errorf("execute parallel approval dialog agent: %w", err)
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
					return fmt.Errorf("parallel approval dialog agent error: %s", evt.Message)
				}
			}

			bc.dialogState.finalContent = strings.Join(contentParts, "")
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
			llmMessages := []llm.Message{{Role: llm.RoleSystem, Content: parallelApprovalDialogPrompt}}
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
				if hasToolCall(bc.dialogState.toolCalls, "itsm.service_load") {
					return nil
				}
			}
			return nil
		}

		msgs := buildMessages()
		if err := executeOnce(msgs); err != nil {
			return err
		}
		if len(bc.dialogState.toolCalls) == 0 {
			retryMsgs := append(append([]ai.ExecuteMessage{}, msgs...), ai.ExecuteMessage{
				Role:    "user",
				Content: "请严格真实调用 itsm.service_match 和 itsm.service_load；不要只做口头回复，也不要跳过工具。",
			})
			if err := executeOnce(retryMsgs); err != nil {
				return err
			}
			if len(bc.dialogState.toolCalls) == 0 {
				if err := executeChatFallback(retryMsgs); err != nil {
					return fmt.Errorf("parallel approval dialog chat fallback error: %w", err)
				}
			}
		}
		return nil
	}

	return run, nil
}

func (bc *bddContext) whenParallelApprovalAgentProcessesDialog() error {
	run, err := setupParallelApprovalDialogTest(bc)
	if err != nil {
		return fmt.Errorf("setup parallel approval dialog: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	return run(ctx)
}
