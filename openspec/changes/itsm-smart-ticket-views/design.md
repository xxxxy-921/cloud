## Context

ITSM 模块采用双引擎架构（Classic + Smart），后端 Smart 引擎已完整实现 AI 决策循环、活动创建、覆盖操作等能力。但前端停留在管理员视角（指派/完结/取消），缺少参与者视角的交互闭环。列表页查询逻辑存在不一致（待办仅按 assignee_id 单维查询、历史无用户范围限定）。菜单因 sidebar 仅支持二级渲染而无法显示三级嵌套的工单子菜单。

本次改动聚焦 Smart 引擎工单，Classic 引擎改动留给后续 change。

**现有关键代码**：
- Smart 引擎: `internal/app/itsm/engine/smart.go` — `runDecisionCycle()` 根据置信度分流：高置信度自动执行 `executeDecisionPlan()`，低置信度等待确认 `pendApprovalDecisionPlan()`
- 详情页: `web/src/apps/itsm/pages/tickets/[id]/index.tsx` — 已有 Classic/Smart 分支渲染、AI 决策面板、覆盖操作组件
- 审批查询: `repository_ticket.go ListApprovals()` — 已实现 user/position/department 多维参与者解析

## Goals / Non-Goals

**Goals:**
- Smart 工单从提单到结单的完整前端交互闭环
- 详情页以"当前活动卡片"为核心，按 6 种 UI 状态分支渲染
- 列表页查询逻辑正确性（待办多维解析、历史用户范围、审批含 AI 确认）
- 菜单可见（扁平化绕过二级限制）

**Non-Goals:**
- Classic 引擎交互改造（后续单独 change）
- Claim 领取机制（Smart 引擎由 AI 直接指定用户，不需要）
- formSchema 动态表单编辑（只做 formData 只读展示）
- 评论/补充说明数据模型（暂不新增字段）
- 动作执行记录展示 + 重试 UI
- sidebar 三级菜单渲染支持
- SSE 实时推送（用 3s 轮询替代）

## Decisions

### D1: 详情页按 6 种 UI 状态分支渲染

**选择**：在详情页顶部新增 `SmartCurrentActivityCard` 组件，根据工单和活动状态判定当前处于哪种 UI 状态，渲染对应的卡片内容和操作按钮。

**状态判定逻辑**（按优先级）：
1. `ticket.status ∈ {completed, cancelled}` → 终态（只读）
2. `ticket.aiFailureCount >= 3` → AI 停用（覆盖操作）
3. 当前活动 `status = pending_approval` → AI 决策待确认（确认/拒绝）
4. 当前活动 `status ∈ {pending, in_progress}` 且 `activityType ∈ {approve, form, process}` → 人工活动等待处理（审批/提交按钮，需判断当前用户是否为处理人）
5. 无 pending/pending_approval 活动 且 ticket 非终态 → AI 推理中（loading + 轮询）

**替代方案考虑**：将状态判定放后端返回 → 增加 API 改动，前端已有足够信息判断，不必要。

### D2: AI 推理中使用 React Query refetchInterval 轮询

**选择**：当检测到 UI 状态为"AI 推理中"时，启用 `refetchInterval: 3000` 自动轮询 ticket 数据，60s 后停止并提示。

**理由**：React Query 天然支持条件轮询，改动量最小。SSE 需要新端点和连接管理，收益不大（AI 决策通常 5-15s 完成）。

### D3: 审批查询扩展策略

**选择**：在 `ListApprovals` repository 方法中新增一个 OR 分支，查询 `activity.status = 'pending_approval'`（不限 activity_type，不限 assignment）。返回结果增加 `approvalKind` 字段（`"workflow"` 或 `"ai_confirm"`），前端据此渲染不同卡片样式。

**理由**：`pending_approval` 活动没有 Assignment（设计如此 — AI 确认是管理行为），所以不能用 Assignment JOIN。采用两段查询 UNION 或在单查询中用 OR 条件。

**替代方案考虑**：为 pending_approval 创建 Assignment → 改变 Smart 引擎的语义设计，牵连大，不值得。

### D4: 待办查询对齐审批的多维解析

**选择**：修改 `ListTodo` repository，从当前仅按 `assignee_id` 过滤，改为与 `ListApprovals` 相同的 JOIN Assignment 逻辑（`user_id = me OR position_id IN myPositions OR department_id IN myDepts`）。

**理由**：当前 Smart 引擎的 `executeDecisionPlan()` 创建 assignment 时只用 `ParticipantType: "user"` 且直接设了 AssigneeID，所以实际效果不变。但这为未来 AI 支持岗位/部门指派做好准备，且保持代码一致性。

### D5: 历史工单加用户范围过滤

**选择**：`ListHistory` 增加 `WHERE (requester_id = :userID OR assignee_id = :userID)` 条件。handler 从 JWT context 获取 userID 传入。

**理由**：普通用户不应看到所有人的历史工单。管理员如需全局视图可使用"全部工单"页面。

### D6: 扁平化菜单而非修改 sidebar

**选择**：修改 `seed.go`，去掉"工单管理"中间 directory，5 个工单菜单（全部工单、我的工单、我的待办、历史工单、我的审批）直接挂在 ITSM 顶级目录下。

**理由**：改 sidebar 组件影响所有 App，风险和工作量大。扁平化是最小改动，后续做三级菜单支持时可以随时改回。

### D7: 拒绝 AI 决策后显示内联暂停卡片

**选择**：拒绝后页面自动 refetch，进入一个"暂停态"卡片，直接内联展示 3 个覆盖操作按钮（重试 AI / 手动跳转 / 重新分配），复用已有的 `override-actions.tsx` 组件逻辑。

**替代方案考虑**：Toast + 跳转 → 操作断裂，体验差。

## Risks / Trade-offs

- **[轮询开销]** → 3s 轮询仅在"AI 推理中"状态激活，60s 超时自动停止。单用户影响极小。
- **[审批查询变复杂]** → UNION/OR 查询可能影响性能 → pending_approval 活动数量极少（每工单最多 1 个），不构成性能问题。
- **[seed 数据迁移]** → 已有数据库的菜单层级不会自动调整 → 文档中说明需要手动调整或重建数据库。Sync 逻辑只添加不修改，无法自动扁平化。
- **[formData 只做只读]** → 限制了 process 类型活动中处理人填写新数据的能力 → 明确为 Non-Goal，后续 change 处理。
