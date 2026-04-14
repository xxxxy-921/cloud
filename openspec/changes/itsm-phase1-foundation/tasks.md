## 1. App 骨架与数据模型

- [ ] 1.1 创建 `internal/app/itsm/app.go` — ITSMApp 结构体实现 App 接口（Name/Models/Seed/Providers/Routes/Tasks）+ LocaleProvider 接口
- [ ] 1.2 创建配置模型 — ServiceCatalog（树形分类）、ServiceDefinition（含 engine_type/workflow_json/collaboration_spec/agent_id/agent_config）、ServiceAction（HTTP webhook）
- [ ] 1.3 创建 SLA 模型 — SLATemplate、EscalationRule
- [ ] 1.4 创建 Priority 模型 — 优先级定义
- [ ] 1.5 创建工单模型 — Ticket（含 source/agent_session_id/sla_status/sla_deadline）、TicketActivity（含 ai_decision/ai_confidence/overridden_by）、TicketAssignment（含 participant_type/position_id/department_id）、TicketTimeline、TicketActionExecution
- [ ] 1.6 创建故障扩展模型 — TicketLink（工单关联）、PostMortem（故障复盘）
- [ ] 1.7 Edition 注册 — `cmd/server/edition_full.go` 加 `import _ "metis/internal/app/itsm"`
- [ ] 1.8 验证 — `go build -tags dev ./cmd/server/` 编译通过，AutoMigrate 创建所有表

## 2. Seed 数据

- [ ] 2.1 ITSM 菜单 Seed — 服务目录、服务定义、工单管理（全部工单/我的工单/我的待办/历史工单）、优先级管理、SLA 管理，按 Org App 的 Seed 模式幂等创建
- [ ] 2.2 Casbin 策略 Seed — admin 角色对 `/api/v1/itsm/*` 的完整 CRUD 策略
- [ ] 2.3 默认优先级 Seed — P0(紧急,#FF0000)、P1(高,#FF6600)、P2(中,#FFAA00)、P3(低,#00AA00)、P4(最低,#888888)
- [ ] 2.4 默认 SLA 模板 Seed — "标准"(响应4h/解决24h)、"紧急"(响应30min/解决4h)

## 3. 服务目录后端

- [ ] 3.1 ServiceCatalog Repo — Create/Update/Delete/FindByID/FindTree/FindByParentID
- [ ] 3.2 ServiceCatalog Service — CRUD + 树形查询 + 删除校验（有子分类或关联服务时拒绝）
- [ ] 3.3 ServiceCatalog Handler — POST/PUT/DELETE /api/v1/itsm/catalogs, GET /api/v1/itsm/catalogs/tree
- [ ] 3.4 ServiceDefinition Repo — Create/Update/Delete/FindByID/List(按catalog_id筛选,分页,搜索)
- [ ] 3.5 ServiceDefinition Service — CRUD + code 唯一性校验 + 启用/禁用
- [ ] 3.6 ServiceDefinition Handler — RESTful /api/v1/itsm/services/*
- [ ] 3.7 ServiceAction Repo + Service + Handler — CRUD /api/v1/itsm/services/:id/actions
- [ ] 3.8 Priority Repo + Service + Handler — CRUD /api/v1/itsm/priorities
- [ ] 3.9 SLATemplate Repo + Service + Handler — CRUD /api/v1/itsm/sla
- [ ] 3.10 EscalationRule Repo + Service + Handler — CRUD /api/v1/itsm/sla/:id/escalations

## 4. 工单后端

- [ ] 4.1 Ticket Repo — Create/Update/FindByID/FindByCode/List(多维筛选+分页)
- [ ] 4.2 工单编号生成 — TICK-XXXXXX 格式自增，并发安全
- [ ] 4.3 TicketTimeline Repo + Service — 事件记录/查询
- [ ] 4.4 Ticket Service — 创建工单（表单校验+SLA计算+Timeline记录）、手动指派、手动完结、取消
- [ ] 4.5 Ticket Handler — POST /api/v1/itsm/tickets（创建）、GET /api/v1/itsm/tickets（列表+筛选）、GET /api/v1/itsm/tickets/:id（详情）、PUT /api/v1/itsm/tickets/:id/assign（指派）、PUT /api/v1/itsm/tickets/:id/complete（完结）、PUT /api/v1/itsm/tickets/:id/cancel（取消）
- [ ] 4.6 我的工单 API — GET /api/v1/itsm/tickets/mine（按 requester_id 筛选，支持 status 过滤）
- [ ] 4.7 我的待办 API — GET /api/v1/itsm/tickets/todo（按 assignee_id + 活跃状态筛选，按优先级+时间排序）
- [ ] 4.8 历史工单 API — GET /api/v1/itsm/tickets/history（终态工单，支持 assignee_id/时间范围筛选）
- [ ] 4.9 工单时间线 API — GET /api/v1/itsm/tickets/:id/timeline

## 5. IOC 注册与路由

- [ ] 5.1 Providers() — 注册所有 Repo、Service、Handler 到 IOC 容器
- [ ] 5.2 Routes() — 挂载所有 Handler 路由到 /itsm 子组
- [ ] 5.3 验证 — 启动服务器，确认所有 API 可通过 curl 调用

## 6. 前端：App 骨架

- [ ] 6.1 创建 `web/src/apps/itsm/module.ts` — registerApp + registerTranslations
- [ ] 6.2 创建 i18n — `locales/zh-CN.json` + `locales/en.json`（所有 UI 文本）
- [ ] 6.3 创建 API 客户端 — `web/src/apps/itsm/api.ts`（所有后端 API 封装）
- [ ] 6.4 `_bootstrap.ts` 添加 `import "./itsm/module"`
- [ ] 6.5 验证 — 前端编译通过，ITSM 菜单可见

## 7. 前端：服务目录管理页

- [ ] 7.1 服务目录管理页 — 树形列表（参考 Org Department），支持增删改拖拽排序
- [ ] 7.2 服务定义列表页 — 分页表格、搜索、按分类筛选、启用/禁用开关、engine_type 标记（📋经典/⚡智能）
- [ ] 7.3 服务定义编辑 Sheet — 基础信息（名称/编码/描述/分类/SLA/引擎类型）+ 表单 Schema 编辑器 + engine_type 切换后显示不同配置区（经典:workflow_json 预留区域 / 智能:Spec+Agent 预留区域）
- [ ] 7.4 动作管理 — ServiceAction CRUD（嵌在服务定义详情内的 Tab 或区域）
- [ ] 7.5 优先级管理页 — 简单列表 + Sheet 编辑（名称/编码/颜色/描述/默认时间）
- [ ] 7.6 SLA 模板管理页 — 列表 + Sheet 编辑 + 升级规则配置（嵌套列表）

## 8. 前端：工单功能

- [ ] 8.1 提单入口页（经典） — 服务目录浏览（卡片/分类树导航）→ 选服务 → 动态表单（根据 form_schema 渲染）→ 提交
- [ ] 8.2 全部工单列表页 — 分页表格、多维筛选器（状态/优先级/服务/处理人）、搜索、来源标记
- [ ] 8.3 我的工单页 — requester 视角，含进行中和已完结，按状态 Tab 切换
- [ ] 8.4 我的待办页 — assignee 视角，仅活跃工单，按优先级排序突出紧急工单
- [ ] 8.5 历史工单页 — 已完结工单列表，时间范围筛选
- [ ] 8.6 工单详情页 — 基础信息卡片 + 时间线 + 操作面板（指派/完结/取消按钮）
- [ ] 8.7 详情页审计字段 — 操作按钮触发时通过 c.Set() 记录 audit_action/audit_resource

## 9. 集成验证

- [ ] 9.1 端到端验证 — 创建分类 → 定义经典服务 → 用户提单 → 管理员指派 → 处理人完结 → 历史工单可查
- [ ] 9.2 Seed 幂等验证 — 重启服务器，Sync 不重复创建
- [ ] 9.3 权限验证 — 非 admin 用户无法访问管理 API，普通用户只能看自己的工单
