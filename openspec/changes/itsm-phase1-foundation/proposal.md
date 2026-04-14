## Why

这是 ITSM 双引擎项目的 Phase 1（共 4 期）。本期目标是搭建 ITSM App 骨架、全部数据模型、服务目录管理、以及基础工单能力（手动流转，不含工作流引擎）。完成后可验证：创建服务分类 → 定义服务 → 用户提交工单 → 管理员查看/手动处理工单 → 完结。

总体设计见 `openspec/changes/itsm-dual-engine/`。

## What Changes

- 新增 ITSM App 骨架（`internal/app/itsm/app.go`），实现 App 接口 + LocaleProvider
- 创建全部数据模型（为后续 Phase 预留字段）：ServiceCatalog、ServiceDefinition、ServiceAction、Priority、SLATemplate、EscalationRule、Ticket、TicketActivity、TicketAssignment、TicketTimeline、TicketActionExecution、TicketLink、PostMortem
- 服务目录管理（后端 + 前端）：树形分类 CRUD、服务定义 CRUD（含 engine_type 选择但引擎逻辑不实现）、动作定义 CRUD
- 优先级管理（后端 + 前端）：P0~P4 CRUD
- SLA 模板管理（后端 + 前端）：SLA 模板 + 升级规则 CRUD
- 基础工单能力（后端 + 前端）：经典入口提单（表单提交）、工单列表/详情、手动状态流转、时间线记录、我的工单/待办
- 前端 App 模块注册、i18n、路由
- Seed 数据：菜单、Casbin 策略、默认优先级、默认 SLA 模板
- Edition 注册 + Bootstrap 注册

## Capabilities

### New Capabilities

- `itsm-service-catalog`: 服务目录树形分类、服务定义（含双模式字段但引擎不实现）、动作定义、优先级、SLA 模板
- `itsm-ticket-lifecycle`: 工单基础生命周期——手动创建/手动流转/手动完结，统一数据模型（Activity/Assignment/Timeline）

### Modified Capabilities

（无）

## Impact

- **新增后端**：`internal/app/itsm/` — 全部 model + 服务目录/工单/SLA/优先级的 repo/service/handler
- **新增前端**：`web/src/apps/itsm/` — 模块注册、服务目录页、服务定义编辑页、优先级页、SLA 页、工单列表/详情、提单页、我的工单/待办
- **Edition**：`cmd/server/edition_full.go` 增加 `import _ "metis/internal/app/itsm"`
- **Bootstrap**：`web/src/apps/_bootstrap.ts` 增加 `import "./itsm/module"`
- **Seed**：ITSM 菜单项、P0~P4 默认优先级、默认 SLA 模板、Casbin 策略
