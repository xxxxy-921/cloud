## ADDED Requirements

### Requirement: Smart 工单详情页 6 种 UI 状态渲染
Smart 引擎工单详情页 SHALL 在基本信息区域下方渲染一个"当前活动卡片"（SmartCurrentActivityCard），根据以下优先级判定 UI 状态并渲染对应内容：

1. **终态**：`ticket.status ∈ {completed, cancelled}` → 只读回顾卡片，展示最终 AI 推理和处理时长
2. **AI 停用**：`ticket.aiFailureCount >= 3` → 暂停卡片，展示失败原因和覆盖操作按钮
3. **AI 决策待确认**：当前活动 `status = pending_approval` → AI 决策面板 + 确认/拒绝按钮
4. **人工活动等待处理**：当前活动 `status ∈ {pending, in_progress}` 且 `activityType ∈ {approve, form, process}` → 表单数据展示 + 操作按钮（按用户身份分支）
5. **AI 推理中**：无 pending/pending_approval 活动且 ticket 非终态 → loading 状态 + 自动轮询

#### Scenario: 工单处于 AI 推理中
- **WHEN** Smart 工单的 ticket.status = in_progress 且无 pending/pending_approval 活动
- **THEN** 详情页展示"AI 正在分析"loading 卡片，启用 3s 轮询自动刷新 ticket 数据

#### Scenario: 轮询超时
- **WHEN** AI 推理中状态持续超过 60s
- **THEN** 停止轮询并展示"AI 推理超时，请手动刷新"提示

#### Scenario: AI 决策待确认
- **WHEN** 当前活动 status = pending_approval 且 aiDecision 非空
- **THEN** 展示 AI 决策面板（推荐的下一步、置信度进度条、推理过程）和 [确认 AI 决策] [拒绝 AI 决策] 按钮

#### Scenario: 人工活动等待处理 - 我是处理人
- **WHEN** 当前活动为人工活动，且 assignment.assigneeId 等于当前用户
- **THEN** 展示表单数据和操作按钮（approve 类型显示 [通过] [驳回]，form/process 类型显示 [提交]）

#### Scenario: 人工活动等待处理 - 我不是处理人
- **WHEN** 当前活动为人工活动，且 assignment.assigneeId 不等于当前用户
- **THEN** 展示表单数据（只读），不显示操作按钮

#### Scenario: AI 停用
- **WHEN** ticket.aiFailureCount >= 3
- **THEN** 展示警告卡片"AI 决策已停用"，显示 [重试 AI] [手动跳转] [重新分配] 按钮

#### Scenario: 终态
- **WHEN** ticket.status = completed 或 cancelled
- **THEN** 展示只读回顾卡片，无操作按钮

### Requirement: AI 决策拒绝后内联暂停引导
当用户拒绝 AI 决策后，详情页 SHALL 自动刷新并渲染暂停态卡片，展示被拒绝的建议信息和 3 个覆盖操作入口（重试 AI / 手动指定下一步 / 指派处理人）。

#### Scenario: 拒绝后展示暂停卡片
- **WHEN** 用户点击 [拒绝 AI 决策] 并成功
- **THEN** 页面自动刷新，展示暂停态卡片，包含被拒绝的建议（活动名 + 置信度）、拒绝人和时间、3 个覆盖操作按钮

### Requirement: 表单数据只读渲染
当活动包含 formData 时，详情页当前活动卡片 SHALL 将 formData（JSON 对象）解析为 key-value 列表展示。每个字段展示 key 标签和 value 内容。

#### Scenario: 展示表单数据
- **WHEN** 当前活动 formData 为 `{"github_account": "zhangsan", "reason": "开发需要"}`
- **THEN** 卡片中展示两行：Github Account = zhangsan, Reason = 开发需要

#### Scenario: formData 为空
- **WHEN** 当前活动 formData 为空或 null
- **THEN** 不渲染表单数据区域

### Requirement: 时间线语义增强
工单详情页时间线 SHALL 为不同事件类型使用语义化的图标和色彩：
- `ticket_created` → 蓝色创建图标
- `ai_decision_executed` / `ai_decision_pending` → 黄色 AI 图标
- `ai_decision_confirmed` → 绿色确认图标
- `ai_decision_rejected` / `ai_decision_failed` → 红色拒绝图标
- `activity_completed` / `workflow_completed` → 绿色完成图标
- `ticket_cancelled` → 灰色取消图标
- `override_jump` / `override_reassign` → 橙色覆盖图标

AI 相关事件 SHALL 展示 reasoning 字段内容（如果存在）。

#### Scenario: AI 决策事件展示推理
- **WHEN** 时间线包含 ai_decision_executed 事件且 reasoning 非空
- **THEN** 事件卡片展示语义图标 + 消息 + 可展开的推理过程文本

#### Scenario: 普通事件
- **WHEN** 时间线包含 ticket_created 事件
- **THEN** 事件卡片展示蓝色创建图标 + 消息 + 时间戳
