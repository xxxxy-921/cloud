## Why

当前 ClassicEngine 使用 `ticket.current_activity_id` 单指针追踪流程执行位置。这是一个单 token 模型——同一时刻只能有一个活跃活动。这使得并行网关（多条分支同时执行）、子流程（独立执行上下文）、边界事件（附着在任务上的并行监听）等核心 BPMN 能力完全不可能实现。

Execution Token（执行令牌）是 BPMN 引擎的核心执行模型。每个 token 代表一条执行路径，token 可以 fork（并行分支）、join（汇合）、nest（子流程嵌套），形成树形结构。这是从"基础工单流转"升级到"BPMN 流程引擎"的关键架构跃迁。

## What Changes

- **新增 ExecutionToken 模型**：`itsm_execution_tokens` 表，字段包括 ticket_id, parent_token_id(自引用), node_id(当前节点), status(active/waiting/completed/cancelled/suspended), token_type(main/parallel/subprocess/multi_instance/boundary), scope_id(变量作用域)
- **Activity 新增 token_id FK**：活动不再直接属于 ticket，而是属于 token。通过 token → ticket 关联。
- **ClassicEngine 重构为 token-based 执行**：
  - `Start`: 创建 root token → 推进到首个节点
  - `processNode`: 基于 token 推进，每次创建 activity 时绑定 token_id
  - `Progress`: 完成当前 token 的活动 → 推进 token 到下一节点
  - `Cancel`: 递归取消 token 树下所有活跃 token 及其活动
- **gateway 重命名为 exclusive**：清除旧的 `gateway` 类型，ValidNodeTypes 注册 `exclusive`、`parallel`、`inclusive`（后两者的执行逻辑在 ④ 中实现，本 change 只做类型注册和校验）
- **Ticket 模型调整**：`current_activity_id` 保留但语义变为"最近活跃 token 的当前活动"（便于列表页快速展示），实际执行追踪通过 token 表
- **ValidateWorkflow 增强**：识别新节点类型（exclusive/parallel/inclusive/script/subprocess/timer/signal/b_timer/b_error），对未实现的类型给出友好提示

## Capabilities

### New Capabilities
- `itsm-execution-token`: ExecutionToken 模型 + 状态机 + token 树操作（fork/join/nest/cancel）
- `itsm-token-activity-binding`: Activity 通过 token_id 绑定到执行路径

### Modified Capabilities
- `itsm-classic-engine`: 从单指针模型重构为 token-based 执行
- `itsm-workflow-validator`: 新增节点类型识别 + exclusive/parallel/inclusive 校验规则
- `itsm-node-types`: gateway → exclusive 重命名；注册所有新节点类型常量

## Impact

- **后端**：`internal/app/itsm/` 新增 ExecutionToken 模型 (~50 行)；`engine/classic.go` 重构 (~200 行改动，核心执行循环)；`engine/engine.go` 新增节点类型常量 + token 状态常量 (~30 行)；`engine/validator.go` 增强 (~40 行)；Activity 模型新增 token_id 字段
- **前端**：本 change 无前端改动（token 对终端用户不可见）
- **数据**：需删库重建（gateway → exclusive 不做兼容）
- **依赖**：② itsm-process-variables（token scope_id 与变量作用域联动）
