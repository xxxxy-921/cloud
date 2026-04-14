## Context

ITSM 双引擎项目 Phase 2。总体架构设计见 `openspec/changes/itsm-dual-engine/design.md`，本文档仅补充 Phase 2 经典工作流引擎的实现细节。

Phase 1 已完成：App 骨架 + 全部模型（Ticket/TicketActivity/TicketAssignment/TicketTimeline/TicketActionExecution 等）+ 服务目录 CRUD + 基础工单（手动流转）。Phase 2 在此基础上实现 ClassicEngine——BPMN 式的确定性图遍历引擎 + ReactFlow 可视化编辑器 + 流程实例运行时可视化。

Phase 2 范围：WorkflowEngine 接口 + ClassicEngine 实现 + 9 种节点类型执行 + 参与人解析 + 动作执行 + 网关条件评估 + 等待节点 + 通知节点 + ReactFlow 编辑器 + 流程可视化 + Scheduler 任务 + TicketService 集成。不含 SmartEngine（Phase 3）、Agent 集成（Phase 3）、SLA 检查任务（Phase 4）、报表（Phase 4）。

## Goals / Non-Goals

**Goals:**

- 定义 WorkflowEngine 接口（Start/Progress/Cancel），为 Phase 3 的 SmartEngine 留好扩展点
- ClassicEngine 以 workflow_json 为执行源，直接解析 ReactFlow 格式的节点/边 JSON
- 实现 9 种节点类型的完整执行语义（start/form/approve/process/action/gateway/notify/wait/end）
- 参与人解析通过 Org App 的 IOC 注入实现（优雅降级：Org App 不存在时 position/department 类型报错，user 类型始终可用）
- 动作节点通过 HTTP Webhook 执行 ServiceAction，支持重试
- ReactFlow 编辑器让管理员在浏览器中可视化设计工作流
- 工单详情页实时展示流程图，高亮当前步骤、标记已完成节点

**Non-Goals:**

- SmartEngine / Agent 驱动 — Phase 3
- Agent Tool 注册 — Phase 3
- SLA 检查定时任务、升级执行 — Phase 4
- 报表和仪表盘 — Phase 4
- 故障关联、复盘 — Phase 4
- 子流程嵌套、多实例并行执行 — 不在当前 ITSM 范围

## Decisions

### D1: ClassicEngine 是纯图遍历——找当前节点、匹配出边、创建下一 Activity

**选择**: ClassicEngine 不维护独立的状态机数据结构。每次 `Progress()` 调用时：
1. 从 workflow_json 解析出节点和边
2. 找到当前 Activity 对应的节点
3. 根据 Activity 的 outcome 匹配出边（edge.data.outcome）
4. 创建目标节点的 Activity
5. 根据目标节点类型执行对应逻辑（自动节点立即递归处理，人工节点等待用户操作）

**理由**: 简单且可预测。workflow_json 是唯一的真实来源（single source of truth），不需要额外的编译或缓存步骤。每次执行都从 JSON 解析，保证流程定义变更立即生效。

### D2: Workflow JSON 运行时解析，不编译为状态机

**选择**: 不将 workflow_json 预编译为内部状态机表示。每次 Start/Progress 时直接解析 JSON，构建节点/边的 map。

**替代方案**: 在保存时编译为 Go 结构体并缓存。

**理由**: ITSM 工作流节点数通常在 10-30 之间，JSON 解析开销可忽略。运行时解析意味着管理员修改 workflow_json 后立即对新工单生效，无需"发布"步骤。已创建的工单仍按工单快照中的 workflow_json 执行（Ticket 表保存创建时的 workflow_json 副本）。

### D3: 参与人解析通过 Org App IOC 可选注入

**选择**: `ParticipantResolver` 在构造时通过 IOC 尝试获取 `OrgService`（`do.InvokeAs` 或类似方式）。获取成功则支持 position/department 类型解析；获取失败则这两种类型返回错误。user 类型和 requester_manager 类型始终可用（user 直接返回指定用户 ID，requester_manager 从 Ticket.requester 查询其上级）。

**理由**: Org App 是可选模块，ITSM 不能硬依赖。但参与人解析中"按岗位/部门查找处理人"是核心需求，通过 IOC 可选注入平衡了功能完整性和模块独立性。

### D4: 动作节点执行通过 Scheduler 异步任务

**选择**: 当 ClassicEngine 遍历到 action 节点时，不在当前请求中同步执行 HTTP 调用。而是创建 Activity（status=in_progress），然后提交 `itsm-action-execute` 异步任务。任务执行器发起 HTTP 请求，成功后自动调用 Progress(outcome="success")，失败后按重试策略重试，最终失败记录 outcome="failed"。

**理由**: HTTP 调用可能耗时较长（配置 timeout 默认 30s），同步执行会阻塞用户请求。通过 Scheduler 异步化保证请求快速返回，同时复用已有的重试和持久化机制。

### D5: ReactFlow 编辑器输出标准 nodes/edges JSON，后端仅读取核心字段

**选择**: 前端 ReactFlow 编辑器保存完整的 ReactFlow 状态（包括 position、style 等布局属性）到 `ServiceDefinition.workflow_json`。后端 ClassicEngine 仅读取以下字段：
- 节点：`id`, `type`, `data`（业务配置）
- 边：`id`, `source`, `target`, `data`（outcome、condition 等）

其他 ReactFlow 特有属性（position, style, selected 等）后端忽略。

**理由**: 前端需要布局信息来还原编辑器状态，后端不关心渲染细节。这种分离确保 ReactFlow 版本升级改变布局属性时不影响后端引擎逻辑。

### D6: 等待节点 timer 使用 Scheduler 异步任务 + 延迟执行

**选择**: 等待节点有两种模式：
- `signal` — 等待外部 API 调用 `POST /api/v1/itsm/tickets/:id/signal` 触发继续
- `timer` — 创建 `itsm-wait-timer` 异步任务，payload 中包含 `execute_after` 时间戳，Scheduler 轮询时检查时间到达后执行

**理由**: Scheduler 已支持异步任务轮询（每 3s），在任务 payload 中加入 `execute_after` 字段即可实现简单的延迟执行，无需引入额外的定时器机制。

## Risks / Trade-offs

- **[风险] ReactFlow JSON 格式演进** → 缓解：后端仅依赖 id/type/data/source/target，不依赖布局属性；前端 ReactFlow 版本锁定，升级时做兼容性测试
- **[风险] 复杂工作流中多网关嵌套导致路径爆炸** → 缓解：保存时 Validator 校验图连通性，检测无限循环（遍历深度限制 100），运行时设置单次 Progress 最大自动步进数（防止 gateway→gateway→... 死循环）
- **[风险] HTTP 动作执行超时** → 缓解：每个 ServiceAction 可配置 timeout（默认 30s，最大 300s），重试次数可配置（默认 3 次），重试间隔指数退避
- **[权衡] 工单创建时保存 workflow_json 快照 vs 始终引用最新版本** → 选择快照：工单创建时将当前 workflow_json 复制到 Ticket 记录中，确保正在执行的工单不受流程定义变更影响
- **[权衡] 通知节点阻塞 vs 非阻塞** → 选择非阻塞：通知节点发送后立即 Progress 到下一节点，不等待送达确认。发送失败记录到 Timeline 但不阻断流程
- **[权衡] 参与人解析的 Org App 可选依赖** → 接受：position/department 类型在无 Org App 时不可用，管理员在编辑器中配置时前端可提示"需要安装组织架构模块"
