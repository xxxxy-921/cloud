## ADDED Requirements

### Requirement: Parallel Gateway Fork 执行
当 processNode 遇到 `type="parallel"` 且 `gateway_direction="fork"` 的节点时，系统 SHALL 执行并行分裂：
1. 将当前 token 的 status 设为 `waiting`
2. 为该节点的每条出边创建一个子 token（token_type="parallel", parent_token_id=当前token.ID, status="active", scope_id 继承父 token）
3. 依次对每个子 token 调用 processNode 推进到出边目标节点
4. 记录 Timeline 事件"并行网关分裂：N 条分支"

#### Scenario: 并行分裂两条分支
- **WHEN** 流程到达 parallel fork 节点，该节点有 2 条出边分别指向节点 A 和节点 B
- **THEN** 当前 token 变为 waiting，创建 2 个子 token（均为 active），分别推进到节点 A 和节点 B

#### Scenario: 并行分裂三条分支
- **WHEN** parallel fork 节点有 3 条出边
- **THEN** 创建 3 个子 token，按出边顺序依次递归 processNode

#### Scenario: 分支到达人工节点停止
- **WHEN** 并行分裂后某条分支的目标是 form 节点
- **THEN** 该分支的子 token 在 form 节点创建 pending Activity 后停止递归，其他分支继续各自推进

#### Scenario: fork 节点无出边
- **WHEN** parallel fork 节点没有出边
- **THEN** 系统返回错误"并行网关 fork 节点 {node_id} 至少需要两条出边"

---

### Requirement: Parallel Gateway Join 执行
当 processNode 遇到 `type="parallel"` 且 `gateway_direction="join"` 的节点时，系统 SHALL 执行合并检查：
1. 将当前子 token 标记为 `completed`
2. 查询同一 parent_token_id 下所有兄弟 token 中 status 为 `active` 或 `waiting` 的数量
3. 如果 remaining == 0（所有分支已完成），将父 token 的 status 从 `waiting` 改为 `active`，沿 join 节点的唯一出边继续 processNode
4. 如果 remaining > 0，当前分支结束，不继续推进（等待其他分支）

#### Scenario: 最后一条分支到达 join 触发合并
- **WHEN** 两条并行分支中，第二条分支（最后一条）的 token 到达 join 节点
- **AND** 第一条分支的子 token 已经 completed
- **THEN** 当前子 token 标记为 completed，父 token 恢复 active，沿 join 后的出边继续流转

#### Scenario: 非最后一条分支到达 join 等待
- **WHEN** 两条并行分支中，第一条分支的 token 到达 join 节点
- **AND** 第二条分支的子 token 仍为 active
- **THEN** 当前子 token 标记为 completed，不触发合并，流程暂停在此

#### Scenario: join 后合并继续走到 end
- **WHEN** join 合并触发后，join 节点的出边指向 end 节点
- **THEN** 父 token 推进到 end 节点，工单完结

#### Scenario: join 节点无出边
- **WHEN** parallel join 节点没有出边
- **THEN** 系统返回错误"并行网关 join 节点 {node_id} 必须有一条出边"

---

### Requirement: Parallel Gateway 深度限制
并行分支共享全局 depth 计数器。fork 创建的子 token 在调用 processNode 时传入 `depth+1`。如果任一分支超过 MaxAutoDepth，该分支 SHALL 中止并标记异常。

#### Scenario: 分支自动步进超过深度限制
- **WHEN** 并行分支中某条路径连续经过超过 50 个自动节点
- **THEN** 该分支的子 token 标记为 cancelled，工单标记为 failed，记录 Timeline 错误
