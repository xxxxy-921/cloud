## Why

这是 ITSM 双引擎项目的 Phase 2（共 4 期）。Phase 1 建立了 App 骨架、全部数据模型、服务目录管理和基础手动工单能力。本期实现经典工作流引擎——BPMN 式的确定性状态机 + ReactFlow 可视化编辑器。完成后：管理员在 ReactFlow 画布上设计流程 → 用户提工单 → 系统按流程自动流转（审批/处理/动作节点），每一步产生 Activity 和 Timeline 记录。

总体设计见 `openspec/changes/itsm-dual-engine/`。

## What Changes

- WorkflowEngine 接口定义 + ClassicEngine 实现（`internal/app/itsm/engine/`）
- Workflow JSON Schema 校验（一个 start、至少一个 end、边合法性、无孤立节点、网关条件完整性等）
- ClassicEngine 图遍历引擎（Start/Progress/Cancel）——找到当前节点、匹配 outcome 到出边、创建下一个 Activity
- 9 种节点类型执行逻辑：start / form / approve / process / action / gateway / notify / wait / end
- 审批参与人解析（user / position / department / requester_manager，集成 Org App）
- 审批模式支持（单人审批 / 并行会签 / 串行依次审批）
- 动作节点执行 ServiceAction（HTTP webhook + 重试 + TicketActionExecution 记录）
- 网关条件评估（equals / not_equals / contains_any / gt / lt / gte / lte）
- 等待节点（signal 外部信号 + timer 定时器）
- 通知节点（集成 Kernel Channel，非阻塞）
- ReactFlow 可视化编辑器（@xyflow/react，拖拽节点面板、画布连线、节点属性面板、边条件配置、保存/加载）
- 流程实例实时可视化（工单详情页高亮当前步骤、标记已完成节点、已走过的边）
- Scheduler 异步任务：itsm-action-execute（HTTP 动作执行）、itsm-wait-timer（等待节点定时唤醒）
- 工单创建流程改造：`engine_type="classic"` 时自动调用 ClassicEngine.Start() 创建首个 Activity

## Capabilities

### New Capabilities

- `itsm-classic-engine`: 经典工作流引擎——WorkflowEngine 接口 + ClassicEngine 图遍历 + 9 种节点执行 + 条件路由 + 参与人解析 + ReactFlow 编辑器 + 实时可视化

### Modified Capabilities

（无）

## Impact

- **新增后端**：`internal/app/itsm/engine/` 目录 — `engine.go`（WorkflowEngine 接口）、`classic.go`（ClassicEngine 实现）、`validator.go`（Workflow JSON 校验）、`resolver.go`（参与人解析）、`executor_action.go`（动作节点 HTTP 执行）
- **修改后端**：`TicketService.Create()` 在 `engine_type="classic"` 时调用 ClassicEngine.Start()；`TicketService` 新增 Progress/Cancel 方法委托给引擎
- **新增前端**：ReactFlow 工作流编辑器组件（节点面板、画布、属性面板、边配置）、工单详情流程图组件（只读模式、状态高亮）
- **前端依赖**：新增 `@xyflow/react`
- **Scheduler**：注册 `itsm-action-execute` 和 `itsm-wait-timer` 异步任务
- **Handler**：新增工单流转 API（`POST /api/v1/itsm/tickets/:id/progress`）、等待信号 API（`POST /api/v1/itsm/tickets/:id/signal`）
