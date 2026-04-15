## ADDED Requirements

### Requirement: Inclusive Gateway Fork 执行
当 processNode 遇到 `type="inclusive"` 且 `gateway_direction="fork"` 的节点时，系统 SHALL 执行条件化分裂：
1. 评估每条出边的条件（复用 exclusive gateway 的 evaluateCondition + buildEvalContext）
2. 为所有条件满足的出边创建子 token（token_type="parallel", parent_token_id=当前token.ID, status="active"）
3. 如果没有任何条件满足，走 default 出边（仅创建一个子 token）
4. 如果没有条件满足且无 default 边，返回错误
5. 将当前 token 的 status 设为 `waiting`
6. 依次对每个子 token 调用 processNode 推进
7. 记录 Timeline 事件"包含网关分裂：N/M 条分支激活"

#### Scenario: 两条条件满足 + 一条不满足
- **WHEN** inclusive fork 有 3 条出边（条件 A、条件 B、条件 C），其中 A 和 B 满足
- **THEN** 创建 2 个子 token 分别推进到 A 和 B 的目标节点，C 的目标不激活

#### Scenario: 无条件满足走默认边
- **WHEN** inclusive fork 的所有非默认出边条件均不满足，但存在 default 出边
- **THEN** 仅为 default 出边创建 1 个子 token

#### Scenario: 全部条件满足
- **WHEN** inclusive fork 的所有出边条件均满足（含 default）
- **THEN** 为每条出边创建子 token（含 default 边），行为等同 parallel fork

#### Scenario: 无条件满足且无默认边
- **WHEN** inclusive fork 的所有条件均不满足且无 default 出边
- **THEN** 系统记录错误到 Timeline，工单标记为 failed

---

### Requirement: Inclusive Gateway Join 执行
当 processNode 遇到 `type="inclusive"` 且 `gateway_direction="join"` 的节点时，系统 SHALL 执行动态合并：
1. 将当前子 token 标记为 `completed`
2. 查询同一 parent_token_id 下所有兄弟 token 中 status 为 `active` 或 `waiting` 的数量
3. 如果 remaining == 0，唤醒父 token 继续
4. 如果 remaining > 0，等待

合并逻辑与 parallel join 完全相同——因为 inclusive fork 在分裂时只为实际激活的分支创建子 token，join 只需等待已创建的子 token 全部完成即可。

#### Scenario: inclusive 动态合并 — 仅等待已激活分支
- **WHEN** inclusive fork 激活了 2 条（共 3 条）分支
- **AND** 这 2 条分支的子 token 均已到达 join 并 completed
- **THEN** 父 token 恢复 active，沿 join 后的出边继续（不等待未激活的第 3 条分支）

#### Scenario: inclusive 单分支激活 — 退化为串行
- **WHEN** inclusive fork 仅激活 1 条分支（其他条件不满足）
- **AND** 该分支到达 join
- **THEN** 父 token 恢复 active，行为与排他网关类似
