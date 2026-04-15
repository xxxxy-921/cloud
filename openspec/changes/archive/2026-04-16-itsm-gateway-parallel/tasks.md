## 1. NodeData 扩展 + 常量更新

- [x] 1.1 在 `engine/workflow.go` 的 `NodeData` 结构体中新增 `GatewayDirection string` 字段（json tag: `gateway_direction,omitempty`）
- [x] 1.2 在 `engine/engine.go` 的 `UnimplementedNodeTypes` 中移除 `NodeParallel` 和 `NodeInclusive`（它们现在有执行逻辑了）
- [x] 1.3 在 `engine/engine.go` 中新增 gateway direction 常量 `GatewayFork = "fork"` 和 `GatewayJoin = "join"`
- [x] 1.4 在 `engine/engine.go` 中新增错误变量 `ErrGatewayNoOutEdge`、`ErrGatewayJoinIncomplete`

## 2. Parallel Gateway Handlers

- [x] 2.1 在 `engine/classic.go` 中新增 `handleParallelFork` 方法：将当前 token 设为 waiting，为每条出边创建子 token（token_type=parallel, parent=当前token），依次 processNode 推进子 token，记录 Timeline
- [x] 2.2 在 `engine/classic.go` 中新增 `handleParallelJoin` 方法：将当前子 token 标记为 completed，查询兄弟 token 剩余 active/waiting 数量，如果 remaining==0 则唤醒父 token 沿 join 出边继续
- [x] 2.3 在 `engine/classic.go` 中新增 `tryCompleteJoin` 辅助方法：封装 join 合并逻辑——查询兄弟 token 状态、唤醒父 token、调用 processNode

## 3. Inclusive Gateway Handlers

- [x] 3.1 在 `engine/classic.go` 中新增 `handleInclusiveFork` 方法：评估每条出边条件（复用 buildEvalContext + evaluateCondition），为满足条件的出边创建子 token，无条件满足则走 default 边，无 default 则报错
- [x] 3.2 在 `engine/classic.go` 中新增 `handleInclusiveJoin` 方法：逻辑与 handleParallelJoin 相同（可直接调用 tryCompleteJoin）

## 4. processNode 路由更新

- [x] 4.1 修改 `processNode` 中的 `case NodeParallel, NodeInclusive` 分支：移除 ErrNodeNotImplemented，改为根据 `nodeData.GatewayDirection` 分派到 fork/join handler
- [x] 4.2 对 `NodeParallel`：direction="fork" 调用 handleParallelFork，direction="join" 调用 handleParallelJoin，其他值返回错误
- [x] 4.3 对 `NodeInclusive`：direction="fork" 调用 handleInclusiveFork，direction="join" 调用 handleInclusiveJoin，其他值返回错误

## 5. handleEnd 分支处理

- [x] 5.1 修改 `handleEnd`：检查 `token.ParentTokenID`，如果不为 nil（child token），仅完成 token + 创建 end activity，然后调用 `tryCompleteJoin` 检查合并；如果为 nil（root token），保持现有逻辑完成工单

## 6. Validator 增强

- [x] 6.1 在 `validator.go` 中新增 parallel/inclusive 节点校验：fork 至少两条出边、join 至少两条入边且恰好一条出边
- [x] 6.2 新增 gateway_direction 校验：parallel/inclusive 节点必须配置 gateway_direction（fork 或 join）
- [x] 6.3 新增 inclusive fork 条件校验：非默认出边须配置条件（复用 exclusive 的校验逻辑）
- [x] 6.4 确认 parallel/inclusive 从 UnimplementedNodeTypes 移除后不再产生"未实现" warning
