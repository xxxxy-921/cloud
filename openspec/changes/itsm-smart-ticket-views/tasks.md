## 1. 后端：菜单与 Seed 调整

- [x] 1.1 扁平化 ITSM 菜单 seed — 修改 `internal/app/itsm/seed.go`，去掉"工单管理"中间 directory，5 个工单菜单直接挂在 ITSM 顶级目录下，调整 Sort 值（服务目录 0、服务定义 1、全部工单 2、我的工单 3、我的待办 4、历史工单 5、我的审批 6、优先级管理 7、SLA 管理 8、引擎配置 9）

## 2. 后端：查询逻辑修正

- [x] 2.1 待办查询多维参与者解析 — 修改 `repository_ticket.go ListTodo()`，从按 `assignee_id` 单维过滤改为 JOIN TicketAssignment 的 `user_id / position_id / department_id` 多维解析，对齐 `ListApprovals` 的逻辑。handler 层需从 Org App 获取用户的 positionIDs 和 deptIDs（可选依赖，Org App 不存在时退回 assignee_id 查询）
- [x] 2.2 历史工单用户范围限定 — 修改 `repository_ticket.go ListHistory()`，增加 `(requester_id = :userID OR assignee_id = :userID)` 条件。修改 `handler_ticket.go History()` 从 JWT context 获取 userID 传入
- [x] 2.3 审批查询扩展 — 修改 `repository_ticket.go ListApprovals()`，新增 OR 分支查询 `activity.status = 'pending_approval'` 的活动（不依赖 Assignment）。返回结果增加 `ApprovalKind` 字段（`"workflow"` 或 `"ai_confirm"`）。同步修改 `CountApprovals()` 包含 pending_approval 计数
- [x] 2.4 待办查询增加关键词和状态过滤 — 修改 `handler_ticket.go Todo()` 和 `repository_ticket.go ListTodo()`，支持 keyword（code/title）和 status 查询参数

## 3. 前端：API 类型扩展

- [x] 3.1 扩展 ApprovalItem 类型 — 在 `web/src/apps/itsm/api.ts` 中为 ApprovalItem 增加 `approvalKind: "workflow" | "ai_confirm"` 字段，以及 `aiConfidence`、`aiReasoning`、`activityStatus` 字段
- [x] 3.2 新增 confirmActivity / rejectActivity API 调用 — 确认 `api.ts` 中已有这些函数（已存在则跳过），确保审批列表页可以调用

## 4. 前端：Smart 工单详情页重构

- [x] 4.1 创建 SmartCurrentActivityCard 组件 — 新建 `web/src/apps/itsm/components/smart-current-activity-card.tsx`，实现 6 种 UI 状态判定和分支渲染的容器组件
- [x] 4.2 实现 AI 推理中状态 — 在 SmartCurrentActivityCard 中，当检测到无 pending/pending_approval 活动且工单非终态时，展示 loading 卡片，使用 React Query `refetchInterval: 3000` 自动轮询，60s 后停止
- [x] 4.3 实现 AI 决策待确认状态 — 复用/重构已有的 `ai-decision-panel.tsx`，展示推理过程、置信度、决策详情，以及 [确认] [拒绝] 按钮
- [x] 4.4 实现人工活动等待处理状态 — 展示活动信息（类型、处理人、执行方式）、表单数据（只读 key-value）、操作按钮（根据用户身份和 activityType 分支：approve → 通过/驳回，form/process → 提交）
- [x] 4.5 实现 AI 停用/决策被拒暂停状态 — 展示警告卡片 + 失败原因/被拒信息 + 3 个覆盖操作按钮（重试 AI / 手动跳转 / 重新分配），复用 `override-actions.tsx` 组件逻辑
- [x] 4.6 实现终态只读状态 — 展示最终结果回顾卡片（AI 推理、处理时长）
- [x] 4.7 集成到详情页 — 在 `pages/tickets/[id]/index.tsx` 中，当 `engineType === "smart"` 时渲染 SmartCurrentActivityCard 替代当前的操作按钮区域
- [x] 4.8 表单数据只读渲染组件 — 新建 `web/src/apps/itsm/components/form-data-display.tsx`，解析 formData JSON 为 key-value 列表展示

## 5. 前端：时间线增强

- [x] 5.1 时间线语义图标和色彩 — 修改详情页时间线渲染，为不同 eventType 使用对应的 Lucide 图标和色彩（见 spec: ticket_created→蓝, ai_decision_*→黄/绿/红, activity_completed→绿, override_*→橙, cancelled→灰）
- [x] 5.2 AI 推理展示 — 时间线中 AI 相关事件展示 reasoning 字段，使用可折叠的文本区域

## 6. 前端：列表页增强

- [x] 6.1 我的审批页面支持 AI 确认 — 修改 `pages/tickets/approvals/index.tsx`，根据 `approvalKind` 渲染两种行样式：workflow 行保持现有 approve/deny 按钮，ai_confirm 行展示置信度 + confirm/reject 按钮，AI 行加 🤖 视觉标识
- [x] 6.2 我的待办列表增强 — 修改 `pages/tickets/todo/index.tsx`，增加"当前活动"列、关键词搜索输入框、状态过滤 Tab
- [x] 6.3 我的工单引擎标识和搜索 — 修改 `pages/tickets/mine/index.tsx`，增加引擎类型列（Smart 显示 🤖 标签）、关键词搜索输入框
- [x] 6.4 历史工单无需前端改动 — 后端已限定用户范围，前端查询逻辑不变（确认 endpoint 不需要传 userID，后端从 JWT 获取）

## 7. 国际化

- [x] 7.1 补充中文翻译 — 在 `web/src/apps/itsm/locales/zh-CN.json` 中增加新增 UI 元素的翻译 key（AI 推理中、AI 决策待确认、暂停态提示、覆盖操作文案等）
- [x] 7.2 补充英文翻译 — 在 `web/src/apps/itsm/locales/en.json` 中增加对应英文翻译

## 8. 验证

- [x] 8.1 编译验证 — 运行 `go build -tags dev ./cmd/server/` 确认后端编译通过
- [x] 8.2 前端构建验证 — 运行 `cd web && bun run build` 确认前端构建通过
- [x] 8.3 前端 lint 验证 — 运行 `cd web && bun run lint` 确认无 lint 错误
