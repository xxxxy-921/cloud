## Purpose

ITSM SmartEngine 轻量 ReAct 循环 -- 在引擎内部实现独立的 Reason-Act-Observe 循环，直接调用 LLM 并携带决策域工具，不依赖 Agent Runtime。

## Requirements

### Requirement: SmartEngine 内置轻量 ReAct 循环
SmartEngine SHALL 在 `smart_react.go` 中实现一个轻量级 ReAct（Reason-Act-Observe）循环，直接调用 `llm.Client.Chat()` 并携带 `Tools` 参数。该循环不依赖 Agent Runtime（Gateway/ReactExecutor/Session），独立运行于引擎内部。

#### Scenario: ReAct 循环基本流程
- **WHEN** SmartEngine 需要做出决策
- **THEN** 系统 SHALL 执行以下循环：(1) 将 messages + tools 发送给 LLM (2) 如果返回 tool_calls，执行工具并将结果追加到 messages (3) 重复直到 LLM 返回无 tool_calls 的最终输出 (4) 解析最终输出为 DecisionPlan JSON

#### Scenario: 多轮工具调用的消息累积
- **WHEN** ReAct 循环第 N 轮 LLM 返回了 tool_calls
- **THEN** 系统 SHALL 将 assistant 消息（含 ToolCalls）追加到 messages 数组，然后对每个 tool_call 执行工具并将 `llm.Message{Role: RoleTool, Content: result, ToolCallID: tc.ID}` 追加到 messages，随后进入第 N+1 轮

#### Scenario: 无工具调用时终止循环
- **WHEN** 第 N 轮 LLM 返回的 `ChatResponse.ToolCalls` 为空
- **THEN** 系统 SHALL 终止循环，将 `ChatResponse.Content` 传入 `parseDecisionPlan()` 解析为 DecisionPlan

### Requirement: ReAct 循环最大轮数限制
ReAct 循环 SHALL 有最大轮数限制，防止无限循环消耗 token。默认值为 `DecisionToolMaxTurns = 8`。

#### Scenario: 达到最大轮数
- **WHEN** ReAct 循环已执行 8 轮且 LLM 仍在返回 tool_calls
- **THEN** 系统 SHALL 终止循环，视为决策失败，调用 `handleDecisionFailure()` 处理

#### Scenario: 正常决策在限制内完成
- **WHEN** Agent 在第 4 轮停止调用工具并输出 DecisionPlan
- **THEN** 循环 SHALL 正常终止，返回解析后的 DecisionPlan

### Requirement: ReAct 循环超时控制
ReAct 循环 SHALL 遵守 `runDecisionCycle()` 传入的 `context.WithTimeout`，整个多轮循环共享同一个超时上下文。

#### Scenario: 超时中断循环
- **WHEN** ReAct 循环执行到第 3 轮时 context 超时
- **THEN** `llm.Client.Chat()` SHALL 返回 context 错误，循环终止，视为决策超时

### Requirement: ReAct 循环复用 LLM 基础设施
ReAct 循环 SHALL 完全复用现有 `internal/llm` 包的能力，不引入新的 LLM 客户端实现。

#### Scenario: 使用 AgentProvider 获取配置
- **WHEN** ReAct 循环初始化
- **THEN** 系统 SHALL 通过 `AgentProvider.GetAgentConfig(agentID)` 获取模型、协议、BaseURL、APIKey、Temperature 等配置

#### Scenario: 使用 llm.NewClient 创建客户端
- **WHEN** ReAct 循环创建 LLM 客户端
- **THEN** 系统 SHALL 调用 `llm.NewClient(protocol, baseURL, apiKey)` 创建客户端，与现有 SmartEngine 使用的方式一致

#### Scenario: 工具定义使用 llm.ToolDef 格式
- **WHEN** ReAct 循环构建 ChatRequest
- **THEN** 决策工具 SHALL 被转换为 `llm.ToolDef` 格式，通过 `ChatRequest.Tools` 字段传入

### Requirement: 工具分发机制
ReAct 循环 SHALL 通过简单的 `map[string]toolHandler` 分发工具调用，不使用 Agent Runtime 的 CompositeToolExecutor。

#### Scenario: 工具名匹配分发
- **WHEN** LLM 返回 tool_call 且 name 为 "decision.resolve_participant"
- **THEN** 系统 SHALL 在 map 中查找对应的 handler 并执行，将结果序列化为 JSON 字符串

#### Scenario: 未知工具名
- **WHEN** LLM 返回 tool_call 且 name 不在已注册工具 map 中
- **THEN** 系统 SHALL 返回 tool result `{"error": true, "message": "未知工具: xxx"}`，循环继续

### Requirement: 初始 seed 构建
ReAct 循环的初始消息 SHALL 包含精简的工单 seed 信息和策略约束，而非全量上下文。

#### Scenario: system message 构建
- **WHEN** 构建初始 messages
- **THEN** system message SHALL 依次包含：(1) 协作规范（如有）(2) Agent system_prompt (3) 可用工具说明和使用指引 (4) 最终输出格式要求（DecisionPlan JSON）

#### Scenario: user message 构建
- **WHEN** 构建初始 messages
- **THEN** user message SHALL 包含：(1) 工单基本信息（code, title, status, priority, service_name）(2) allowed_step_types 列表 (3) 当前状态和允许的状态转换 (4) 明确提示 Agent 可通过工具获取更多信息

#### Scenario: user message 不包含全量用户列表
- **WHEN** 构建初始 user message
- **THEN** message 中 SHALL NOT 包含 participant_candidates 全量用户列表
