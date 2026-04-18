## Why

Smart Engine 当前有两个关键能力缺失导致无法覆盖真实 ITSM 场景：

1. **并签（Countersign）**——多角色并行审批后汇聚的能力不存在。Smart Engine 的 `executeDecisionPlan` 对多个 activities 是 last-one-wins 覆盖 `current_activity_id`，没有 parallel group 概念，`Progress` 也不具备汇聚检查。
2. **Action 元调用**——当前 action 执行是"声明式"的（AI 输出 `type: "action"`，scheduler 异步执行 webhook，完成后无人自动触发下一轮决策）。在 Agentic 模式下，action 应该是 AI 决策推理过程中的 tool call：Agent 同步调用、看到结果、继续推理。

这两个缺失意味着带并签审批的工单和带自动化触发器的工单在智能引擎下完全无法运转。需要引擎改造 + LLM 驱动的 BDD 严格验证。

## What Changes

- **Smart Engine 并签**：`DecisionPlan` 新增 `execution_mode: "parallel"` 字段；`executeDecisionPlan` 识别并签模式并创建共享 `activity_group_id` 的并行活动组；`Progress` 增加汇聚检查——同组未全部完成时不触发下一轮决策，全部完成后汇聚推进。
- **Action 元调用**：新增 `decision.execute_action` 决策工具，AI Agent 在 ReAct loop 中同步执行 HTTP webhook 并获取结果继续推理；移除 Smart Engine 对 `type: "action"` 活动 + scheduler 异步执行的路径依赖。
- **AI 输出格式扩展**：`agenticOutputFormat` 增加 `execution_mode` 字段说明和并签引导；`agenticToolGuidance` 增加 `decision.execute_action` 工具说明和使用时机。
- **ticket_context 增强**：`decision.ticket_context` 工具返回值增加 `parallel_groups` 状态，使 AI 在后续决策循环中能感知并签进度。
- **BDD 验证**：新增 LLM 驱动的并签 BDD 场景（2 scenarios）；修改现有 db_backup BDD 场景适配 action 元调用模式（步骤简化，验证点不变）。

## Capabilities

### New Capabilities
- `itsm-smart-countersign`: Smart Engine 多角色并签——`DecisionPlan` parallel mode、activity group、Progress 汇聚检查、BDD 验证
- `itsm-smart-action-tool`: Smart Engine Action 元调用——`decision.execute_action` 决策工具、Agent 同步执行 webhook、BDD 验证

### Modified Capabilities
- `itsm-smart-engine`: `DecisionPlan` 结构扩展 + `executeDecisionPlan` 并签分支 + `Progress` 汇聚逻辑
- `itsm-decision-tools`: 新增 `decision.execute_action` tool + `ticket_context` 增强返回 parallel_groups
- `itsm-smart-react`: `agenticOutputFormat` 和 `agenticToolGuidance` 更新

## Impact

- **引擎核心**：`engine/smart.go`（DecisionPlan、executeDecisionPlan、Progress）、`engine/smart_react.go`（prompt）、`engine/smart_tools.go`（新 tool + ticket_context 增强）、`engine/tasks.go`（HandleActionExecute smart 分支简化）
- **数据模型**：`model_ticket.go` TicketActivity 新增 `ActivityGroupID` 字段；`engine/classic.go` activityModel 同步
- **BDD 测试**：新增 3 个测试文件（并签 feature + steps + support）；修改 2 个文件（db_backup feature + steps 适配）；`bdd_test.go` 注册新步骤
- **不影响**：Classic Engine（并行网关逻辑不变）、前端（无 UI 变更）、API（无接口变更）
