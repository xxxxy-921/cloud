## Why

并行网关（Parallel Gateway）是 BPMN 中最重要的控制流结构之一。ITSM 场景中大量存在并行审批需求：如"IT 主管和财务主管同时审批"、"安全评审与技术评审并行进行"。当前引擎只有排他网关（Exclusive Gateway，原名 gateway），只能走单条路径，无法表达这些并行语义。

包含网关（Inclusive Gateway）是并行网关的条件化版本——只 fork 满足条件的分支，join 时只等待实际激活的分支。适用于"根据工单类型决定需要哪些并行审批"的场景。

两种网关共享 token fork/join 底层机制，应在同一个 change 中实现。

## What Changes

- **Parallel Gateway (fork)**：遇到 `parallel` 类型节点且 `gatewayDirection=fork` 时，为所有出边创建子 token（token_type=parallel），父 token 进入 waiting 状态
- **Parallel Gateway (join)**：遇到 `parallel` 类型节点且 `gatewayDirection=join` 时，完成当前子 token；检查同一父 token 下所有兄弟 token 是否全部 completed；全部到达则合并——创建新活动在 join 后的下一节点上，使用父 token 继续推进
- **Inclusive Gateway (fork)**：为所有条件匹配的出边创建子 token（至少一条 default 边保底），父 token 进入 waiting 状态
- **Inclusive Gateway (join)**：等待所有 active 子 token 到达（忽略未创建的分支），然后合并
- **Fork/Join 配对校验**：ValidateWorkflow 新增规则——parallel 和 inclusive 的 fork 必须有对应的 join（通过出边路径分析），join 节点入边数必须 >= 2
- **Progress 路由增强**：Progress 时根据 activity.token_id 找到对应 token 推进，而非全局单指针

## Capabilities

### New Capabilities
- `itsm-parallel-gateway`: Parallel Gateway fork/join 执行逻辑 + token 分裂/合并
- `itsm-inclusive-gateway`: Inclusive Gateway 条件化 fork + 动态 join

### Modified Capabilities
- `itsm-classic-engine`: processNode 新增 parallel/inclusive case；Progress 基于 token 路由
- `itsm-workflow-validator`: fork/join 配对校验规则

## Impact

- **后端**：`engine/classic.go` 新增 handleParallelFork/handleParallelJoin/handleInclusiveFork/handleInclusiveJoin (~250 行)；validator 增强 (~60 行)
- **前端**：本 change 无前端改动（节点 UI 在 ⑥ itsm-bpmn-designer 中实现）
- **依赖**：③ itsm-execution-tokens（token fork/join 底层）
