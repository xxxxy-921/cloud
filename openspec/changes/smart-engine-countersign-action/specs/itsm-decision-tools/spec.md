## MODIFIED Requirements

### Requirement: 决策域工具集定义
SmartEngine SHALL 提供一组决策域工具，供决策 Agent 在 ReAct 循环中按需调用。工具定义（名称、描述、参数 JSON Schema）在 `smart_tools.go` 中硬编码注册，不通过 `ai_agent_tools` 表动态绑定。每个工具 SHALL 返回 JSON 格式结果。

#### Scenario: 工具集包含 8 个决策工具
- **WHEN** SmartEngine 初始化构建工具列表
- **THEN** 工具集 SHALL 包含以下 8 个工具：`decision.ticket_context`、`decision.knowledge_search`、`decision.resolve_participant`、`decision.user_workload`、`decision.similar_history`、`decision.sla_status`、`decision.list_actions`、`decision.execute_action`

#### Scenario: 工具定义转换为 llm.ToolDef
- **WHEN** ReAct 循环构建 `llm.ChatRequest`
- **THEN** 每个决策工具 SHALL 被转换为 `llm.ToolDef{Name, Description, Parameters}` 格式传入 `ChatRequest.Tools`

### Requirement: decision.ticket_context 工具
该工具 SHALL 返回工单的完整上下文信息，包括表单数据、SLA 状态、活动历史和并签组状态。这是初始 seed 的补充，Agent 需要详细信息时调用。

参数：无（工具执行时从 ReAct 循环上下文获取 ticketID）

返回字段：
- `form_data`: 完整表单 JSON
- `description`: 工单详细描述
- `sla_status`: SLA 剩余时间（response_remaining_seconds, resolution_remaining_seconds），无 SLA 时为 null
- `activity_history`: 已完成活动列表（type, name, outcome, completed_at, ai_reasoning）
- `current_assignment`: 当前指派信息（assignee_id, assignee_name），无指派时为 null
- `executed_actions`: 已成功执行的动作名称列表
- `all_actions_completed`: 布尔值，所有服务动作是否全部执行完毕
- `parallel_groups`: 当前活跃的并签组状态（group_id, total, completed, pending_activities）

#### Scenario: 查询含并签组的工单上下文
- **WHEN** Agent 调用 `decision.ticket_context` 且工单有一个活跃的并签组（2 活动，1 已完成）
- **THEN** 返回结果 SHALL 包含 `parallel_groups` 字段，其中 `total=2, completed=1, pending_activities` 列出未完成活动

#### Scenario: 查询无并签组的工单上下文
- **WHEN** Agent 调用 `decision.ticket_context` 且工单无活跃并签组
- **THEN** 返回结果 SHALL NOT 包含 `parallel_groups` 字段（或为空数组）

#### Scenario: 查询含 SLA 的工单上下文
- **WHEN** Agent 调用 `decision.ticket_context` 且工单关联了 SLA 模板
- **THEN** 返回结果 SHALL 包含 `sla_status` 字段，其中 `response_remaining_seconds` 和 `resolution_remaining_seconds` 为距当前时间的剩余秒数

#### Scenario: 查询含活动历史的工单
- **WHEN** Agent 调用 `decision.ticket_context` 且工单有 3 个已完成活动
- **THEN** 返回结果的 `activity_history` SHALL 包含 3 条记录，按完成时间升序排列
