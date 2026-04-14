## ADDED Requirements

### Requirement: 工单数据模型

系统 SHALL 使用统一的工单数据模型，支持经典和智能两种引擎（本期不实现引擎，仅手动流转）。

Ticket 模型：code（唯一编号如 TICK-000001）、title、description、service_id（FK→ServiceDefinition）、engine_type（继承自服务）、status（枚举）、priority_id（FK→Priority）、requester_id（FK→User）、assignee_id（FK→User，可选）、current_activity_id（FK→TicketActivity，可选）、source（"catalog"|"agent"）、agent_session_id（uint，可选）、form_data（JSON）、sla_response_deadline（时间，可选）、sla_resolution_deadline（时间，可选）、sla_status（"on_track"|"breached_response"|"breached_resolution"）、finished_at（时间，可选），嵌入 BaseModel。

TicketActivity 模型：ticket_id、name、activity_type（"form"|"approve"|"process"|"action"|"end"）、status（"pending"|"active"|"completed"|"rejected"|"failed"|"blocked"）、node_id（字符串，经典模式引用 workflow_json 节点 ID）、execution_mode（"single"|"parallel"|"serial"）、form_schema（JSON）、form_data（JSON）、transition_outcome（"submit"|"approve"|"reject"|"success"|"failure"）、ai_decision（JSON，智能模式）、ai_reasoning（文本）、ai_confidence（float）、overridden_by（uint，被人工覆盖时记录操作人 ID）、decision_reasoning（文本）、started_at、finished_at，嵌入 BaseModel。

TicketAssignment 模型：ticket_id、activity_id、participant_type（"user"|"requester_manager"|"position"|"department"）、user_id（指定人时）、position_id（指定岗位时）、department_id（指定部门时）、assignee_id（实际认领人）、status（"pending"|"active"|"claimed"|"completed"|"rejected"|"failed"）、sequence（并行/串行的顺序）、is_current、claimed_at、finished_at，嵌入 BaseModel。

TicketTimeline 模型：ticket_id、activity_id（可选）、operator_id（FK→User）、event_type（枚举）、message、details（JSON）、reasoning（文本），嵌入 BaseModel。

TicketActionExecution 模型：ticket_id、activity_id、service_action_id、status（"pending"|"success"|"failed"）、request_payload（JSON）、response_payload（JSON）、failure_reason、retry_count，嵌入 BaseModel。

TicketLink 模型：parent_ticket_id、child_ticket_id、link_type（"related"|"caused_by"|"blocked_by"），嵌入 BaseModel。

PostMortem 模型：ticket_id（唯一）、root_cause、impact_summary、action_items（JSON 数组）、lessons_learned、created_by，嵌入 BaseModel。

#### Scenario: 模型自动迁移
- **WHEN** ITSM App 的 Models() 被调用
- **THEN** 返回上述所有模型，main.go 自动 AutoMigrate

### Requirement: 工单状态枚举

工单状态 SHALL 包含以下值：pending（待处理）、in_progress（处理中）、waiting_approval（待审批）、waiting_action（待动作执行）、completed（已完成）、failed（失败）、cancelled（已取消）。

#### Scenario: 初始状态
- **WHEN** 工单创建
- **THEN** 状态为 pending

#### Scenario: 终态不可变
- **WHEN** 工单状态为 completed、failed 或 cancelled
- **THEN** 不允许再变更状态（除 cancelled 可由管理员撤销取消）

### Requirement: 工单编号自动生成

系统 SHALL 为每个工单自动生成唯一编号，格式为 `TICK-XXXXXX`（6 位零填充自增数字）。

#### Scenario: 自动生成编号
- **WHEN** 创建新工单
- **THEN** 系统自动生成下一个序号，如 TICK-000001、TICK-000002

#### Scenario: 编号唯一性
- **WHEN** 并发创建多个工单
- **THEN** 每个工单编号 MUST 唯一

### Requirement: 经典入口提单

系统 SHALL 支持用户从服务目录选择经典服务后填写表单提交工单。

#### Scenario: 提交工单
- **WHEN** 用户选择一个 is_active 的服务，填写 form_data 并提交
- **THEN** 系统创建 Ticket（source="catalog"、engine_type 继承自服务、status=pending）、记录 Timeline 事件（ticket_created）、根据服务的 SLA 计算 deadline

#### Scenario: 表单校验
- **WHEN** form_data 不符合服务定义的 form_schema
- **THEN** 系统返回 400 并提示具体校验错误

#### Scenario: 自动计算 SLA deadline
- **WHEN** 服务绑定了 SLA 模板
- **THEN** 工单创建时根据 SLA 的 response_minutes 和 resolution_minutes 计算 sla_response_deadline 和 sla_resolution_deadline

### Requirement: 工单手动状态流转（Phase 1）

在引擎未接入前（Phase 1），系统 SHALL 支持管理员手动变更工单状态。

#### Scenario: 手动指派处理人
- **WHEN** 管理员为 pending 工单指定 assignee_id
- **THEN** 工单 assignee_id 更新，状态变为 in_progress，记录 Timeline

#### Scenario: 手动完结
- **WHEN** 处理人标记工单为已完成
- **THEN** 工单状态变为 completed，finished_at 设为当前时间，记录 Timeline

#### Scenario: 取消工单
- **WHEN** 用户或管理员取消工单
- **THEN** 工单状态变为 cancelled，记录 Timeline（包含取消原因）

### Requirement: 工单时间线

系统 SHALL 为每个工单维护完整的时间线事件记录。

#### Scenario: 自动记录事件
- **WHEN** 工单创建、状态变更、指派变更、评论添加
- **THEN** 系统自动创建 TicketTimeline 记录，包含 event_type、operator_id、message

#### Scenario: 查询时间线
- **WHEN** 请求 GET /api/v1/itsm/tickets/:id/timeline
- **THEN** 系统返回该工单的时间线事件列表，按创建时间升序

### Requirement: 工单列表与筛选

系统 SHALL 提供工单列表页面，支持多维度筛选。

#### Scenario: 分页查询
- **WHEN** 请求 GET /api/v1/itsm/tickets?page=1&pageSize=20
- **THEN** 系统返回分页工单列表（ListResult 格式）

#### Scenario: 多维度筛选
- **WHEN** 请求携带 status、priority_id、service_id、assignee_id、requester_id 筛选参数
- **THEN** 系统按条件过滤返回

#### Scenario: 关键词搜索
- **WHEN** 请求携带 keyword 参数
- **THEN** 系统在 code、title、description 中模糊匹配

#### Scenario: DataScope 数据权限
- **WHEN** 启用了 Org App 的 DataScope
- **THEN** 工单列表按用户部门范围过滤

### Requirement: 我的工单视图

系统 SHALL 提供"我的工单"视图，展示当前用户作为申请人提交的工单。

#### Scenario: 查询我的工单
- **WHEN** 请求 GET /api/v1/itsm/tickets/mine?status=pending
- **THEN** 系统返回 requester_id 为当前用户的工单列表，支持按 status 筛选

#### Scenario: 我的工单包含所有状态
- **WHEN** 请求我的工单不传 status
- **THEN** 返回该用户所有工单（含进行中和历史已完结的）

### Requirement: 我的待办视图

系统 SHALL 提供"我的待办"视图，展示当前用户需要处理的工单。

#### Scenario: 查询我的待办
- **WHEN** 请求 GET /api/v1/itsm/tickets/todo
- **THEN** 系统返回 assignee_id 为当前用户且状态为 pending/in_progress/waiting_approval 的工单列表

#### Scenario: 待办排序
- **WHEN** 返回待办列表
- **THEN** 按优先级 value 升序（P0 最先）、创建时间升序排列

### Requirement: 历史工单视图

系统 SHALL 提供"历史工单"视图，展示已完结的工单。

#### Scenario: 查询历史工单
- **WHEN** 请求 GET /api/v1/itsm/tickets/history
- **THEN** 系统返回状态为 completed、failed、cancelled 的工单列表

#### Scenario: 按处理人查看历史
- **WHEN** 请求携带 assignee_id 参数
- **THEN** 返回该处理人处理过的历史工单

#### Scenario: 按时间范围筛选
- **WHEN** 请求携带 start_date 和 end_date 参数
- **THEN** 返回 finished_at 在该范围内的历史工单

### Requirement: 工单详情页

系统 SHALL 提供工单详情页面，展示工单完整信息。

#### Scenario: 详情内容
- **WHEN** 请求 GET /api/v1/itsm/tickets/:id
- **THEN** 返回工单基础信息、服务信息（名称/分类路径）、优先级、SLA 状态、当前处理人、表单数据

#### Scenario: 详情页包含时间线
- **WHEN** 渲染工单详情页
- **THEN** 页面同时展示时间线事件列表

### Requirement: 审批参与人类型支持

TicketAssignment 的 participant_type 字段 SHALL 支持以下类型，为 Phase 2 引擎预留完整的派单能力。

#### Scenario: 指定人
- **WHEN** participant_type 为 "user"，user_id 有值
- **THEN** 该用户即为目标处理人（assignee_id = user_id）

#### Scenario: 指定部门
- **WHEN** participant_type 为 "department"，department_id 有值
- **THEN** 该部门下任一成员可认领此 Assignment

#### Scenario: 指定岗位
- **WHEN** participant_type 为 "position"，position_id 有值
- **THEN** 拥有该岗位的任一成员可认领此 Assignment

#### Scenario: 申请人主管
- **WHEN** participant_type 为 "requester_manager"
- **THEN** 系统通过 Org App 查询申请人的直属主管作为目标处理人

### Requirement: Seed 数据

ITSM App 的 Seed() MUST 创建初始菜单、Casbin 策略、默认优先级和 SLA 模板。

#### Scenario: 首次安装
- **WHEN** 系统 Install 模式运行
- **THEN** 创建 ITSM 相关菜单（服务目录、服务定义、工单管理、优先级、SLA 等）、admin 角色的 Casbin 策略、P0~P4 默认优先级、默认 SLA 模板

#### Scenario: 增量同步
- **WHEN** 系统 Sync 模式重启
- **THEN** 幂等检查，仅创建缺失的菜单/策略/优先级/SLA（不覆盖已修改的记录）
