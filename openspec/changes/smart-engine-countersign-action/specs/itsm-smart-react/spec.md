## MODIFIED Requirements

### Requirement: 初始 seed 构建
ReAct 循环的初始消息 SHALL 包含精简的工单 seed 信息和策略约束，而非全量上下文。

#### Scenario: system message 构建
- **WHEN** 构建初始 messages
- **THEN** system message SHALL 依次包含：(1) 协作规范（如有）(2) Agent system_prompt (3) 可用工具说明和使用指引（含 `decision.execute_action` 工具说明）(4) 最终输出格式要求（DecisionPlan JSON，含 `execution_mode` 字段说明）

#### Scenario: user message 构建
- **WHEN** 构建初始 messages
- **THEN** user message SHALL 包含：(1) 工单基本信息（code, title, status, priority, service_name）(2) allowed_step_types 列表 (3) 当前状态和允许的状态转换 (4) 明确提示 Agent 可通过工具获取更多信息

#### Scenario: user message 不包含全量用户列表
- **WHEN** 构建初始 user message
- **THEN** message 中 SHALL NOT 包含 participant_candidates 全量用户列表

## ADDED Requirements

### Requirement: agenticToolGuidance 包含 execute_action 说明
`agenticToolGuidance` 常量 SHALL 在工具列表中包含 `decision.execute_action` 工具的说明和使用时机。

#### Scenario: 工具使用指引包含 execute_action
- **WHEN** Agent 读取工具使用指引
- **THEN** 指引 SHALL 包含 `decision.execute_action` 的描述："同步执行服务配置的自动化动作并获取结果"
- **AND** 推荐推理步骤 SHALL 引导 Agent 使用 `decision.execute_action` 工具而非输出 `type: "action"` 的活动

#### Scenario: 推荐步骤引导 action 元调用
- **WHEN** 协作规范要求执行触发器动作
- **THEN** 指引 SHALL 明确说明 Agent 应使用 `decision.execute_action` 同步执行，在 ReAct 循环内看到结果后继续推理

### Requirement: agenticOutputFormat 包含 execution_mode 字段
`agenticOutputFormat` 常量 SHALL 在 DecisionPlan JSON 格式说明中包含 `execution_mode` 字段。

#### Scenario: 输出格式包含 execution_mode
- **WHEN** Agent 读取输出格式要求
- **THEN** JSON 模板 SHALL 包含 `"execution_mode": "single|parallel"` 字段
- **AND** 字段说明 SHALL 注明：`"single"` 为默认串行模式，`"parallel"` 为并签模式（activities 中的多个活动并行等待处理）

#### Scenario: 并签模式引导
- **WHEN** 协作规范要求多角色并行审批
- **THEN** 输出格式说明 SHALL 引导 Agent 设置 `execution_mode` 为 `"parallel"` 并在 activities 中列出所有并行角色
