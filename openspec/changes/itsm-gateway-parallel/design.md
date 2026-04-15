## Context

③ itsm-execution-tokens 引入了 ExecutionToken 模型和 token 驱动的 processNode 流程。当前 `classic.go` 中 `NodeParallel` 和 `NodeInclusive` 返回 `ErrNodeNotImplemented`。Token 树结构（parent_token_id 自引用）已就位，`TokenWaiting` 和 `TokenParallel` 常量已定义。引擎采用同步递归事务模型（单次 `gorm.DB` 事务贯穿整个流程推进）。

WFNode.Data 中尚无 `gatewayDirection` 字段；WFEdge.Data.Condition 结构用于排他网关条件评估，包含网关条件也适用于包含网关的 fork 分支筛选。

## Goals / Non-Goals

**Goals:**
- 实现 Parallel Gateway 的 fork（分裂子 token 并行推进）和 join（等待所有子 token 完成后合并）
- 实现 Inclusive Gateway 的条件化 fork（仅激活满足条件的分支）和动态 join（仅等待已激活分支）
- 增强 ValidateWorkflow：parallel/inclusive 节点 fork 必须有对应 join
- handleEnd 区分 root token（完结工单）和 child token（仅完成分支）
- Cancel 增强：递归取消子 token 树

**Non-Goals:**
- 前端节点 UI（⑥ itsm-bpmn-designer）
- 嵌套并行网关（fork 内再 fork）的支持 — 当前仅支持一层深度，嵌套留作后续优化
- 异步 token 调度器 — 保持同步递归模型

## Decisions

### D1: Fork/Join 分离策略 — 同一类型节点通过 gatewayDirection 区分

**决定**: parallel 和 inclusive 节点复用同一 `type`，通过 `NodeData.GatewayDirection` 字段（`"fork"` / `"join"`）区分角色。

**替代方案**:
- A. 使用不同的 type 常量（如 `parallel_fork` / `parallel_join`）— 需要注册更多节点类型，增加复杂度
- B. 前端/后端约定靠出入边数量推断 — 脆弱且不可读

**理由**: 与 BPMN 2.0 规范一致（gatewayDirection 是标准属性），一个 `type` + 属性比多个 `type` 更简洁。前端只需为 parallel/inclusive 各渲染一种菱形，属性面板选 fork/join。

### D2: 同步递归 fork — 每条分支依次递归 processNode

**决定**: handleParallelFork 在单次事务中为每条出边创建子 token，然后依次调用 `processNode(childToken, targetNode)`。每条分支递归到遇到人工节点（form/approve/process/wait）或 join 节点时自然停止。

**替代方案**: 异步模型——fork 时仅创建子 token，靠后台 worker 分别推进。

**理由**: 引擎当前完全是同步事务模型。引入异步调度器是架构级变更，超出本 change 范围。同步递归对并行网关足够——每条分支最终都会停在人工节点或 join。

### D3: Join 触发机制 — 子 token 到达 join 时 DB 计数判断

**决定**: 当子 token 到达 join 节点时：
1. 将当前子 token 标记为 `completed`
2. 查询同一 `parent_token_id` 下的兄弟 token，统计仍为 `active` 或 `waiting` 的数量
3. 如果 remaining == 0，唤醒父 token（status → active），继续 join 后的下一节点

**替代方案**: fork 时在父 token 上记录"期望子 token 数"，join 时做倒计数。

**理由**: DB 查询是 source of truth，无需维护额外计数器。子 token 数可能因 inclusive gateway 的条件筛选而不确定，计数器方案对 inclusive 不适用。

### D4: Inclusive Gateway fork — 条件评估 + 默认边保底

**决定**: handleInclusiveFork 评估每条出边的条件，为所有满足条件的出边创建子 token。如果没有任何条件满足，走 default 边。至少激活一条分支。

条件评估复用 exclusive gateway 的 `evaluateCondition` + `buildEvalContext` 函数。

### D5: handleEnd 行为分支 — 根据 parent_token_id 区分

**决定**:
- `token.ParentTokenID == nil`（root token）→ 完成工单（现有行为不变）
- `token.ParentTokenID != nil`（child token）→ 仅完成当前 token，触发 join 合并检查（`tryCompleteJoin`）

### D6: 子 token 到达 join 的路由 — processNode 中 join 节点特殊处理

**决定**: 在 processNode 的 switch 中：
- `NodeParallel` + `direction=fork` → `handleParallelFork`
- `NodeParallel` + `direction=join` → `handleParallelJoin`
- 同理 `NodeInclusive` → fork/join

join handler 不递归 processNode，而是完成子 token 后检查合并条件。

### D7: NodeData 扩展

**决定**: NodeData 新增 `GatewayDirection string` 字段（json tag: `gateway_direction`），值为 `"fork"` 或 `"join"`。此字段仅对 parallel/inclusive 类型有意义。

### D8: Cancel 增强 — 现有批量更新已覆盖

**决定**: Cancel() 的现有逻辑 `WHERE ticket_id = ? AND status IN (active, waiting)` 已自动覆盖所有子 token。无需额外递归取消逻辑。

## Risks / Trade-offs

**[嵌套 fork 的深度限制]** → 当前不支持嵌套并行（fork 内再 fork），如果 fork 路径上遇到另一个 fork，需要确保 MaxAutoDepth 和 join 合并逻辑正确处理。Mitigation: 本 change 暂不支持嵌套，validator 可以检测并 warn。

**[Join 竞态：多个分支同时到达 join]** → 因为是同步递归模型（单事务），不存在真正的竞态。所有分支 processNode 是顺序执行的，最后一个到达 join 的分支触发合并。无需加锁。

**[分支中的自动步进链过深]** → fork 出的分支如果走了很多自动节点（exclusive → notify → ...），MaxAutoDepth 以全局 depth 计算，所有分支共享深度计数器。这是正确的——防止 fork 路径上无限递归。

**[Inclusive join 的"所有活跃分支"判断]** → 依赖子 token 状态查询。如果某条分支因异常（如 action 节点 HTTP 失败）而停在中间，该子 token 保持 active，join 永远不会触发。Mitigation: 需要超时机制或手动干预——这是 BPMN 引擎的通用问题，可在后续 change 中处理。
