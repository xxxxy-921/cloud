## MODIFIED Requirements

### Requirement: 决策循环核心流程
SmartEngine 的每一步决策 SHALL 遵循以下循环：初始 seed 构建 -> ReAct 工具调用循环 -> DecisionPlan 输出 -> Validate -> 信心评估。决策循环中 Agent 通过决策域工具按需查询上下文，而非依赖预构建的全量快照。决策循环通过 Scheduler 异步任务 `itsm-smart-progress` 执行。

#### Scenario: 完整决策循环
- **WHEN** 一个 Activity 完成需要决定下一步
- **THEN** 引擎 SHALL 依次执行：(1) 构建精简初始 seed（工单基本信息 + 策略约束）(2) 启动 ReAct 循环（Agent 通过工具按需获取上下文）(3) Agent 停止工具调用后输出 DecisionPlan (4) 校验 DecisionPlan 合法性 (5) 根据信心分数决定自动执行或等待人工确认

#### Scenario: Agent 决定流程结束
- **WHEN** Agent 的 DecisionPlan 中 `next_step_type` 为 `"complete"`
- **THEN** 引擎 SHALL 将工单状态更新为 `completed`，记录 `finished_at`，在 Timeline 添加完结记录

#### Scenario: 决策循环异步执行
- **WHEN** Activity 完成触发 Progress
- **THEN** 系统 SHALL 通过 Scheduler 的 `itsm-smart-progress` 异步任务执行决策循环，避免阻塞 HTTP 请求

### Requirement: TicketCase 快照构建
系统 SHALL 为 ReAct 循环构建精简的初始 seed，仅包含 Agent 启动推理所需的基本信息。详细上下文（表单数据、SLA、活动历史、知识内容）由 Agent 通过决策域工具按需查询。

初始 seed 字段：
- `ticket`: 工单基本信息（code、title、status、priority_name、source）
- `service`: 服务名称和引擎类型
- `collaboration_spec`: 注入 system prompt（不放在 user message 中）

#### Scenario: 初始 seed 不包含全量表单数据
- **WHEN** 构建初始 seed
- **THEN** seed 中 SHALL NOT 包含完整的 form_data JSON，Agent 需通过 `decision.ticket_context` 工具获取

#### Scenario: 初始 seed 不包含活动历史
- **WHEN** 构建初始 seed 且工单已有多个已完成 Activity
- **THEN** seed 中 SHALL NOT 包含 activity_history，Agent 需通过 `decision.ticket_context` 工具获取

#### Scenario: 初始 seed 不包含 SLA 详情
- **WHEN** 构建初始 seed
- **THEN** seed 中 SHALL NOT 包含 SLA 剩余时间详情，Agent 需通过 `decision.sla_status` 工具获取

### Requirement: TicketPolicySnapshot 编译
系统 SHALL 为 ReAct 循环编译精简的 TicketPolicySnapshot，仅定义 Agent 的行为边界约束。参与人候选列表不再全量灌入。

Policy 字段：
- `allowed_step_types`: 允许的 activity_type 列表
- `allowed_status_transitions`: 当前工单状态允许的状态转换列表
- `current_status`: 当前工单状态

#### Scenario: Policy 不包含全量用户列表
- **WHEN** 编译 Policy
- **THEN** Policy 中 SHALL NOT 包含 `participant_candidates` 字段，Agent 需通过 `decision.resolve_participant` 工具按需查询参与人

#### Scenario: Policy 不包含动作列表
- **WHEN** 编译 Policy
- **THEN** Policy 中 SHALL NOT 包含 `available_actions` 字段，Agent 需通过 `decision.list_actions` 工具按需查询可用动作

#### Scenario: 已完结工单不可操作
- **WHEN** 编译 Policy 且工单状态为 `completed` 或 `cancelled`
- **THEN** Policy SHALL 返回空的 `allowed_step_types` 列表

### Requirement: Agent 调用机制
SmartEngine SHALL 通过 ReAct 循环调用 Agent，替代原有的单次 `llm.Client.Chat()` 调用。Agent 在循环中可使用决策域工具按需获取信息，最终输出 DecisionPlan JSON。

#### Scenario: 构建 Agent 调用上下文
- **WHEN** 引擎准备启动 ReAct 循环
- **THEN** 系统 SHALL 构建消息序列：
  - system message: `[Collaboration Spec]\n\n---\n\n[Agent system_prompt]\n\n---\n\n[工具使用指引]\n\n---\n\n[最终输出格式要求]`
  - user message: `[精简初始 seed JSON]\n\n[策略约束 JSON]\n\n请通过工具获取所需信息，然后输出决策。`
- **AND** ChatRequest SHALL 携带 `Tools` 字段包含所有决策域工具定义

#### Scenario: Agent 多轮工具调用后输出决策
- **WHEN** Agent 在 ReAct 循环中调用了 3 个工具后停止工具调用
- **THEN** 系统 SHALL 将 Agent 的最终文本输出解析为 DecisionPlan JSON

#### Scenario: Agent 首轮直接输出决策（简单场景）
- **WHEN** Agent 在 ReAct 循环第 1 轮即不调用任何工具直接输出 DecisionPlan
- **THEN** 系统 SHALL 正常解析 DecisionPlan，不强制要求 Agent 必须使用工具

#### Scenario: 格式纠正在循环内自然处理
- **WHEN** Agent 输出的内容无法解析为 DecisionPlan JSON
- **THEN** 引擎 SHALL 视为决策失败调用 `handleDecisionFailure()`，不再有独立的 `callAgentWithCorrection()` 重试机制

### Requirement: TicketDecisionPlan 校验调整
DecisionPlan 校验逻辑 SHALL 适配工具按需查询模式，不再依赖全量候选人列表进行校验。

#### Scenario: 校验参与人存在性
- **WHEN** Agent 指定的 `participant_id` 需要校验
- **THEN** 系统 SHALL 直接查询数据库确认该用户存在且 `is_active=true`，而非检查是否在候选列表中

#### Scenario: 校验动作存在性
- **WHEN** Agent 指定的 `action_id` 需要校验
- **THEN** 系统 SHALL 直接查询 `itsm_service_actions` 表确认该动作存在、属于当前服务且 `is_active=true`

### Requirement: 默认超时调整
SmartEngine 的决策超时默认值 SHALL 从 30 秒调整为 60 秒，以适应多轮 ReAct 循环的延迟增加。

#### Scenario: 默认超时
- **WHEN** 服务定义的 `agent_config` 未设置 `decision_timeout_seconds`
- **THEN** 系统 SHALL 使用默认值 60 秒

#### Scenario: 自定义超时范围
- **WHEN** 管理员设置 `decision_timeout_seconds`
- **THEN** 系统 SHALL 接受 10-180 秒范围内的值
