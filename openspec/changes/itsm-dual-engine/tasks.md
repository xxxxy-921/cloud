## 1. App 骨架与基础模型

- [ ] 1.1 创建 `internal/app/itsm/app.go` — 实现 App 接口（Name/Models/Seed/Providers/Routes/Tasks），实现 LocaleProvider 接口
- [ ] 1.2 创建 ITSM 基础模型 — ServiceCatalog（树形分类）、ServiceDefinition（含 engine_type/workflow_json/collaboration_spec/agent_id）、ServiceAction（HTTP webhook）
- [ ] 1.3 创建工单核心模型 — Ticket、TicketActivity、TicketAssignment、TicketTimeline、TicketActionExecution
- [ ] 1.4 创建 SLA 模型 — SLATemplate、EscalationRule
- [ ] 1.5 创建 Priority 模型 — 优先级定义（P0~P4）
- [ ] 1.6 创建故障扩展模型 — TicketLink（工单关联）、PostMortem（故障复盘）
- [ ] 1.7 Edition 注册 — `edition_full.go` 加 `import _ "metis/internal/app/itsm"`
- [ ] 1.8 Seed 数据 — 默认 P0~P4 优先级、默认 SLA 模板、ITSM 菜单项、Casbin 策略

## 2. 服务目录管理（后端）

- [ ] 2.1 ServiceCatalog Repository + Service — 树形 CRUD、获取树结构、面包屑路径
- [ ] 2.2 ServiceCatalog Handler — RESTful API（/api/v1/itsm/catalogs/*）
- [ ] 2.3 ServiceDefinition Repository + Service — CRUD、按分类筛选、启用/禁用
- [ ] 2.4 ServiceDefinition Handler — RESTful API（/api/v1/itsm/services/*），含 engine_type 校验
- [ ] 2.5 ServiceAction Repository + Service + Handler — CRUD、绑定到服务（/api/v1/itsm/actions/*）
- [ ] 2.6 Priority Repository + Service + Handler — CRUD（/api/v1/itsm/priorities/*）
- [ ] 2.7 SLATemplate Repository + Service + Handler — CRUD（/api/v1/itsm/sla/*）
- [ ] 2.8 EscalationRule Repository + Service + Handler — CRUD，绑定到 SLA

## 3. 经典工作流引擎（后端）

- [ ] 3.1 定义 WorkflowEngine 接口 — Start/Progress/Cancel
- [ ] 3.2 实现 Workflow JSON Schema 校验 — 节点/边验证（一个 start、至少一个 end、无孤立节点、边合法性）
- [ ] 3.3 实现 ClassicEngine.Start — 找到 start 节点 → 创建首个 Activity → 创建 Assignment
- [ ] 3.4 实现 ClassicEngine.Progress — 图遍历（找出边 → 按 outcome 过滤 → 创建下一个 Activity）
- [ ] 3.5 实现网关节点条件评估 — 基于 form_data 的规则表达式（equals/contains_any/gt/lt）
- [ ] 3.6 实现审批节点 — 单人/并行/串行模式，参与人解析（user/position/department/requester_manager）
- [ ] 3.7 实现动作节点 — 触发 ServiceAction（HTTP 请求），记录 TicketActionExecution
- [ ] 3.8 实现通知节点 — 通过 Kernel Channel 发送通知
- [ ] 3.9 实现等待节点 — 等待外部信号 API + 定时触发
- [ ] 3.10 实现 ClassicEngine.Cancel — 取消工单 + 终止所有进行中的 Activity

## 4. 智能工作流引擎（后端）

- [ ] 4.1 实现 SmartEngine 结构 — 通过 IOC 注入 AI App 的 AgentService 和 LLM Client（AI App 不存在时禁用智能引擎）
- [ ] 4.2 实现 TicketCase 快照构建 — 组装工单状态、表单数据、服务定义、SLA、Collaboration Spec、历史活动
- [ ] 4.3 实现 TicketPolicySnapshot 编译 — 从 ServiceDefinition 提取允许的活动类型、参与人类型、可用动作
- [ ] 4.4 实现 Agent 决策调用 — 构建 prompt（TicketCase + Policy）→ 调用 LLM → 解析 TicketDecisionPlan（结构化输出）
- [ ] 4.5 实现决策校验 — 验证 Agent 输出是否符合 Policy 约束
- [ ] 4.6 实现 SmartEngine.Progress — 决策循环（Snapshot → Policy → Decide → Validate → Create Activity）
- [ ] 4.7 实现信心机制 — confidence 阈值比较，低于阈值创建待确认状态的 Activity
- [ ] 4.8 实现人工覆盖 — 强制跳转/改派/驳回 API
- [ ] 4.9 实现决策超时 — 30s 超时后转人工队列
- [ ] 4.10 实现 Fallback — AI 不可用时创建人工决策队列
- [ ] 4.11 实现 SmartEngine.Start — 首次决策（分类 + 优先级确认 + 首步派单）
- [ ] 4.12 实现 SmartEngine.Cancel — 取消工单 + 终止决策

## 5. 工单生命周期（后端）

- [ ] 5.1 Ticket Repository + Service — 创建/查询/更新/取消，按 engine_type 分派引擎
- [ ] 5.2 TicketActivity Repository + Service — Activity CRUD、状态流转
- [ ] 5.3 TicketAssignment Repository + Service — 认领/完成/拒绝
- [ ] 5.4 TicketTimeline Repository + Service — 事件记录、查询
- [ ] 5.5 Ticket Handler — 工单 CRUD API（/api/v1/itsm/tickets/*），含创建/流转/取消/认领/处理
- [ ] 5.6 工单编号生成 — 自增编号（如 TICK-000001）
- [ ] 5.7 我的工单 / 我的待办 API — 按 requester_id / assignee_id 筛选

## 6. SLA 引擎与故障管理（后端）

- [ ] 6.1 SLA 计算 — 工单创建时根据 SLA 模板计算 response_deadline 和 resolution_deadline
- [ ] 6.2 SLA 检查任务 — Scheduler 定时任务（每分钟），扫描未完结工单检查超时
- [ ] 6.3 升级执行 — 超时后按 EscalationRule 执行通知/改派/提升优先级
- [ ] 6.4 SLA 状态追踪 — on_track / breached_response / breached_resolution
- [ ] 6.5 故障关联 — TicketLink 模型（parent/child），关联/取消关联 API
- [ ] 6.6 故障复盘 — PostMortem 模型（根因分析/改进措施），CRUD API
- [ ] 6.7 P0/P1 自动通知 — 高优先级工单创建时通过 Channel 通知升级链

## 7. Agent 工具注册（后端）

- [ ] 7.1 实现 ITSM ToolProvider — 定义 6 个 Builtin Tool 的 inputSchema 和 handler
- [ ] 7.2 itsm.search_services — 搜索可用智能服务，返回名称/描述/表单要求
- [ ] 7.3 itsm.create_ticket — 创建工单（设置 source=agent, agent_session_id），触发 SmartEngine.Start
- [ ] 7.4 itsm.query_ticket — 查询工单状态/当前步骤/处理人/SLA 剩余
- [ ] 7.5 itsm.list_my_tickets — 查询调用者的工单列表
- [ ] 7.6 itsm.cancel_ticket / itsm.add_comment — 取消工单和添加评论
- [ ] 7.7 Tool 注册集成 — Providers() 中通过 IOC 获取 AI App ToolRegistry，AI App 不存在时跳过

## 8. Scheduler 任务注册

- [ ] 8.1 itsm-sla-check（scheduled, 每分钟）— SLA 超时检查 + 升级触发
- [ ] 8.2 itsm-escalation-check（scheduled, 每分钟）— 故障升级链检查
- [ ] 8.3 itsm-ticket-timeout（scheduled, 每5分钟）— 长期无响应工单自动关闭/提醒
- [ ] 8.4 itsm-action-execute（async）— 异步执行 ServiceAction（HTTP webhook）
- [ ] 8.5 itsm-smart-progress（async）— 异步执行智能引擎决策循环

## 9. 前端：App 骨架与服务目录

- [ ] 9.1 创建 `web/src/apps/itsm/module.ts` — registerApp + registerTranslations
- [ ] 9.2 创建 i18n 文件 — `locales/zh-CN.json` + `locales/en.json`
- [ ] 9.3 ITSM API 客户端 — `web/src/apps/itsm/api.ts`
- [ ] 9.4 服务目录管理页 — 树形 CRUD（类似 Org Department 的树形交互）
- [ ] 9.5 服务定义列表页 — 分页表格、搜索、启用/禁用开关
- [ ] 9.6 服务定义编辑（Sheet 抽屉）— 基础信息 + engine_type 选择
- [ ] 9.7 经典服务配置 — 表单 Schema 编辑器 + workflow_json 关联
- [ ] 9.8 智能服务配置 — Collaboration Spec 编辑器 + Agent 选择器 + 知识库绑定 + 信心阈值
- [ ] 9.9 动作管理 — ServiceAction CRUD（HTTP 配置表单）
- [ ] 9.10 SLA 模板管理页 + 升级规则配置
- [ ] 9.11 优先级管理页

## 10. 前端：ReactFlow 工作流编辑器

- [ ] 10.1 引入 @xyflow/react 依赖
- [ ] 10.2 工作流画布组件 — 拖拽节点、连线、缩放、自动布局
- [ ] 10.3 节点类型组件 — start/form/approve/process/action/gateway/notify/wait/end 各自的渲染样式
- [ ] 10.4 属性面板 — 选中节点后编辑：类型、名称、参与人、表单Schema、执行模式
- [ ] 10.5 边属性 — transition_outcome、transition_kind、网关条件编辑器
- [ ] 10.6 节点工具栏 — 从侧栏拖入节点类型
- [ ] 10.7 工作流校验 — 前端实时校验（一个 start、至少一个 end、边完整性）
- [ ] 10.8 保存/加载 — 将 ReactFlow 状态序列化为 workflow_json 并保存到 ServiceDefinition

## 11. 前端：工单功能

- [ ] 11.1 经典提单入口 — 服务目录浏览（卡片列表/树形导航）→ 选服务 → 动态表单 → 提交
- [ ] 11.2 工单列表页 — 分页表格、筛选（状态/优先级/服务/处理人）、搜索、来源标记（📋/⚡）
- [ ] 11.3 工单详情页 — 时间线 + 当前活动操作面板 + 表单 + 状态流转
- [ ] 11.4 经典工单流程图 — 在工单详情中渲染 workflow_json，高亮当前步骤
- [ ] 11.5 智能工单动态流程图 — 基于已走过的 Activity 链动态渲染路径
- [ ] 11.6 审批操作 — 通过/驳回按钮 + 审批意见
- [ ] 11.7 处理操作 — 填写处理结果表单 + 完成/转派
- [ ] 11.8 人工覆盖操作 — 智能工单的覆盖面板（接受AI建议 / 改派 / 强制跳转）
- [ ] 11.9 我的工单页面 — 按 requester 筛选
- [ ] 11.10 我的待办页面 — 按 assignee 筛选，支持认领
- [ ] 11.11 AI Copilot 入口 — 智能工单详情中的"AI 协助"按钮，打开 Agent Session Chat

## 12. 前端：报表与仪表盘

- [ ] 12.1 ITSM Dashboard 首页 — 我的待办数/今日新建/SLA达成率/优先级分布
- [ ] 12.2 工单吞吐量报表 — 折线图（日/周/月）
- [ ] 12.3 SLA 达成率报表 — 按服务/时间统计
- [ ] 12.4 平均解决时长报表 — 按服务/优先级
- [ ] 12.5 分类统计 — 饼图/柱状图
- [ ] 12.6 处理人工作量 — 待办数/已完结数/平均处理时长
- [ ] 12.7 AI 决策统计 — 自动执行比例/人工覆盖比例/平均信心度（智能服务专属）

## 13. 故障管理前端

- [ ] 13.1 优先级管理页面 — P0~P4 编辑（颜色/描述/默认响应解决时间）
- [ ] 13.2 升级链配置页面 — 按优先级配置多级升级
- [ ] 13.3 故障关联操作 — 工单详情中"关联工单"功能
- [ ] 13.4 故障复盘页面 — 创建/编辑复盘记录（根因分析 + 改进措施）

## 14. 集成与收尾

- [ ] 14.1 `_bootstrap.ts` 添加 `import "./itsm/module"`
- [ ] 14.2 i18n 完善 — 确保所有 UI 文本有 zh-CN 和 en 翻译
- [ ] 14.3 Seed 数据验证 — 确保幂等（Install + Sync 模式）
- [ ] 14.4 权限验证 — Casbin 策略覆盖所有 ITSM API
- [ ] 14.5 数据权限 — 工单列表根据 DataScope 过滤（Org App 集成）
- [ ] 14.6 审计日志 — 关键操作（创建/流转/取消/覆盖）记录审计
- [ ] 14.7 端到端测试 — 经典服务全流程（提单→审批→处理→完结）
- [ ] 14.8 端到端测试 — 智能服务全流程（Agent 提单→AI 决策→处理→完结）
