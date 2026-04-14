## Context

ITSM 双引擎项目 Phase 1。总体架构设计见 `openspec/changes/itsm-dual-engine/design.md`，本文档仅补充 Phase 1 的实现细节。

Phase 1 范围：App 骨架 + 全部模型 + 服务目录 CRUD + 基础工单（手动流转）+ 前端模块。不含工作流引擎（Phase 2/3）、不含 Agent 集成（Phase 3）、不含 SLA 检查任务和报表（Phase 4）。

## Goals / Non-Goals

**Goals:**

- ITSM App 骨架跑通（注册、AutoMigrate、Seed、路由挂载）
- 全部数据模型一次性建好（后续 Phase 只加逻辑不加表）
- 服务目录管理可用（分类树 + 服务定义 + 动作 + 优先级 + SLA）
- 基础工单跑通（提单 → 列表 → 详情 → 手动改状态 → 完结）
- 前端 ITSM 模块可访问

**Non-Goals:**

- 工作流引擎（经典/智能）— Phase 2/3
- ReactFlow 编辑器 — Phase 2
- Agent 集成、Tool 注册 — Phase 3
- SLA 检查定时任务、升级执行 — Phase 4
- 报表和仪表盘 — Phase 4
- 故障关联、复盘 — Phase 4

## Decisions

### D1: 全部模型在 Phase 1 一次性创建

**选择**: 即使 Phase 1 不使用工作流引擎，也创建 TicketActivity、TicketAssignment 等模型。engine_type、workflow_json、collaboration_spec、agent_id 等字段全部预留。

**理由**: 避免后续 Phase 加字段导致 migration 冲突。GORM AutoMigrate 是幂等的，预留字段不影响 Phase 1 功能。

### D2: Phase 1 的工单流转采用"手动模式"

**选择**: 工单创建后状态为 `pending`。管理员可手动更改状态（pending → in_progress → completed/cancelled）。不创建 Activity 链，也不走引擎。

**理由**: Phase 1 的目标是验证数据层和 UI 框架。手动流转足以验证工单 CRUD 和前端交互。Phase 2 接入经典引擎后，工单创建会自动触发 Engine.Start()，Activity 链由引擎创建。

### D3: 服务定义的 engine_type 字段在 Phase 1 可选择但不生效

**选择**: UI 上可以选择"经典"或"智能"，保存到 DB。但实际提单和流转不区分引擎——所有工单都走手动模式。

**理由**: 让管理员提前配置好服务类型，Phase 2/3 接入引擎后无缝切换。

### D4: 前端参考现有 App 模式（Org App 为模板）

**选择**: 服务目录页参考 Org Department 的树形交互。服务定义页参考 License Product 的列表+Sheet 编辑模式。工单列表参考 Task Management 的分页表格模式。

**理由**: 保持 Metis 前端体验一致性。

### D5: Seed 数据包含完整的菜单结构

**选择**: Phase 1 的 Seed 一次性注册所有 ITSM 菜单（包括后续 Phase 的页面入口），后续 Phase 的菜单前端路由在到达时才实际可用。

**理由**: 避免后续 Phase 需要修改 Seed 逻辑。菜单项指向未实现的前端路由时，React Router 的 lazy loading 会显示 404，不影响已实现的功能。

## Risks / Trade-offs

- **[权衡] 模型预留字段导致初期表结构看起来"多余"** → 接受：Phase 1 的 API 不暴露未使用的字段，前端不显示未实现的功能
- **[风险] 手动流转模式和引擎模式的切换** → 缓解：Phase 2 引入引擎后，工单创建逻辑从"直接创建"变为"创建 + Engine.Start()"，改动集中在 TicketService.Create() 一处
