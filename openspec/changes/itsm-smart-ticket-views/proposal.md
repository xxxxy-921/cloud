## Why

ITSM 模块的 Smart 引擎后端已基本完备（AI 决策循环、活动创建、覆盖操作），但前端用户侧交互存在系统性缺失：工单详情页只有管理员视角（指派/完结/取消），缺少参与者视角（确认 AI 决策/审批/提交）；列表页查询逻辑不完整（待办仅按 assignee_id 单维查询、历史工单无用户范围限定）；菜单因三级嵌套渲染限制无法显示。这些问题导致 Smart 工单在实际使用中无法形成完整的交互闭环。

## What Changes

- **扁平化 ITSM 菜单**：去掉"工单管理"中间 directory，5 个工单菜单直接挂在 ITSM 顶级目录下，绕过 sidebar 二级渲染限制
- **Smart 工单详情页重构**：以"当前活动卡片"为核心交互区，按 6 种 UI 状态分支渲染（AI 推理中、AI 决策待确认、人工活动等待处理、AI 停用/决策被拒、已完成、已取消）
- **表单数据只读渲染**：解析 activity.formData 为 key-value 展示（不做 formSchema 动态编辑）
- **审批查询扩展**：ListApprovals 加入 `status=pending_approval` 分支，使 AI 低置信度决策出现在"我的审批"中，返回 `approvalKind` 字段区分工作流审批和 AI 确认
- **待办查询升级**：ListTodo 对齐 ListApprovals 的多维参与者解析（user_id OR position_id OR department_id）
- **历史工单用户范围限定**：ListHistory 加 `(requester_id=me OR assignee_id=me)` 过滤
- **列表页增强**：我的待办增加当前活动列/搜索/过滤；我的工单增加引擎类型标识和搜索；审批列表支持 AI 确认卡片
- **时间线增强**：语义图标/色彩、AI 推理展示、决策事件高亮
- **AI 推理中自动轮询**：检测到 Smart 工单处于推理中间态时，3s 轮询自动刷新，60s 超时提示

## Capabilities

### New Capabilities
- `itsm-smart-ticket-detail`: Smart 引擎工单详情页交互，包含 6 种 UI 状态的分支渲染、当前活动卡片、表单数据展示、AI 决策确认/拒绝、暂停态引导
- `itsm-ticket-list-views`: 工单列表视图增强（我的工单、我的待办、历史工单、我的审批），包含查询逻辑修正和 UI 改进

### Modified Capabilities
- `itsm-approval-api`: 审批查询扩展，加入 pending_approval 状态支持和 approvalKind 区分字段
- `itsm-approval-ui`: 审批列表页支持 AI 决策确认卡片的渲染和操作

## Impact

- **后端**：`internal/app/itsm/seed.go`（菜单扁平化）、`repository_ticket.go`（ListTodo/ListHistory 查询修改）、`handler_ticket.go` + `service_ticket.go`（审批查询扩展）
- **前端**：`web/src/apps/itsm/pages/tickets/[id]/`（详情页重构）、`web/src/apps/itsm/pages/tickets/mine|todo|history|approvals/`（列表页增强）、`web/src/apps/itsm/components/`（新增当前活动卡片组件）、`web/src/apps/itsm/api.ts`（类型扩展）、`web/src/apps/itsm/locales/`（翻译补充）
- **API**：无新增端点，仅现有端点的查询逻辑和返回结构调整
- **数据库**：无 schema 变更，仅 seed 数据调整
