## MODIFIED Requirements

### Requirement: 节点类型 — gateway 网关节点
gateway 节点 SHALL 重命名为 `exclusive` 节点。`exclusive` 节点 SHALL 根据条件自动选择**唯一一条**出边（排他网关语义）。exclusive 是自动节点，不创建需要人工干预的 Activity。节点 `data.conditions` 定义条件列表，每个条件关联一条出边。条件评估基于工单字段值（Ticket 字段、流程变量 `var.*`、表单数据 `form.*`）。

系统 SHALL 同时注册 `parallel` 和 `inclusive` 节点类型常量。**parallel 和 inclusive 节点的执行逻辑已实现**：processNode 根据 `NodeData.gateway_direction`（"fork"/"join"）分派到对应的 handler。ValidateWorkflow 对这两个类型 SHALL 执行 fork/join 配对校验。

NodeData SHALL 新增 `GatewayDirection string`（json: `gateway_direction`）字段，值为 `"fork"` 或 `"join"`，仅对 parallel/inclusive 节点有意义。

#### Scenario: exclusive 条件匹配到对应出边
- **WHEN** 流程到达 exclusive 节点，第一个条件评估为 true
- **THEN** 系统沿该条件对应的出边继续，跳过后续条件评估

#### Scenario: exclusive 无条件匹配时走默认边
- **WHEN** exclusive 节点的所有条件均评估为 false，但存在 default=true 的出边
- **THEN** 系统沿默认出边继续

#### Scenario: exclusive 无条件匹配且无默认边
- **WHEN** exclusive 节点所有条件均为 false，且没有默认出边
- **THEN** 系统记录错误到 Timeline，将工单标记为异常状态

#### Scenario: parallel 节点 fork 分派
- **WHEN** processNode 遇到 type="parallel" 且 gateway_direction="fork" 的节点
- **THEN** 系统调用 handleParallelFork 执行并行分裂

#### Scenario: parallel 节点 join 分派
- **WHEN** processNode 遇到 type="parallel" 且 gateway_direction="join" 的节点
- **THEN** 系统调用 handleParallelJoin 执行合并检查

#### Scenario: inclusive 节点 fork 分派
- **WHEN** processNode 遇到 type="inclusive" 且 gateway_direction="fork" 的节点
- **THEN** 系统调用 handleInclusiveFork 执行条件化分裂

#### Scenario: inclusive 节点 join 分派
- **WHEN** processNode 遇到 type="inclusive" 且 gateway_direction="join" 的节点
- **THEN** 系统调用 handleInclusiveJoin 执行动态合并

---

### Requirement: 节点类型 — end 结束节点
end 节点 SHALL 标记流程完成。系统 SHALL 根据 token 的层级区分行为：
- **root token**（parent_token_id 为 nil）：到达 end 节点时，将工单状态设为 `completed`，记录流程完结 Timeline 事件
- **child token**（parent_token_id 不为 nil）：到达 end 节点时，仅将当前 token 标记为 `completed`，然后触发 join 合并检查（tryCompleteJoin）

一个合法的 workflow_json 中 SHALL 至少有一个 end 节点。end 节点 SHALL 无出边。

#### Scenario: root token 正常到达 end 节点
- **WHEN** root token（main 类型）流转到达 end 节点
- **THEN** 工单状态设为 completed，记录 Timeline "流程完结"

#### Scenario: child token 到达 end 节点
- **WHEN** 并行分支的子 token 到达 end 节点（而不是 join 节点）
- **THEN** 子 token 标记为 completed，触发 tryCompleteJoin 检查父 token 的所有子 token 是否全部完成

#### Scenario: 多个 end 节点（不同分支）
- **WHEN** workflow_json 包含多个 end 节点（如正常完结和异常完结分支各一个）
- **THEN** 到达任一 end 节点均完结工单（仅 root token 时），end 节点的 `data.label` 记录到 Timeline 区分完结类型

---

### Requirement: Workflow JSON Schema 校验
系统 SHALL 在保存 workflow_json 时进行完整性校验。校验规则包括：有且仅有一个 start 节点；至少一个 end 节点；start 节点有且仅有一条出边；end 节点无出边；所有边的 source 和 target 引用存在的节点 ID；无孤立节点（每个非 start 节点至少有一条入边）；**exclusive** 节点的每条非默认出边 SHALL 配置条件；**exclusive** 节点至少有两条出边；节点类型 SHALL 是合法的已注册类型之一。

**Parallel/Inclusive 校验新增规则：**
- parallel/inclusive 节点 SHALL 移出 UnimplementedNodeTypes（不再输出"未实现"warning）
- parallel 和 inclusive 的 fork 节点 SHALL 至少有两条出边
- parallel 和 inclusive 的 join 节点 SHALL 至少有两条入边且恰好一条出边
- inclusive fork 节点的每条非默认出边 SHALL 配置条件（复用 exclusive 的校验逻辑）
- parallel/inclusive 节点 SHALL 有 gateway_direction 属性（"fork" 或 "join"），缺失时报错

对 `script`、`subprocess`、`timer`、`signal`、`b_timer`、`b_error` 等已注册但未实现执行逻辑的节点类型，ValidateWorkflow SHALL 通过校验但返回 **warning** 级别的 ValidationError。

#### Scenario: 校验通过
- **WHEN** 管理员保存 workflow_json，内容包含 1 个 start、1 个 end、合法的边关系
- **THEN** 校验通过，workflow_json 保存成功

#### Scenario: exclusive 出边缺少条件
- **WHEN** exclusive 节点的某条非默认出边没有配置 condition
- **THEN** 校验失败，返回错误"排他网关节点 {node_id} 的出边 {edge_id} 缺少条件配置"

#### Scenario: exclusive 出边不足
- **WHEN** exclusive 节点只有一条出边
- **THEN** 校验失败，返回错误"排他网关节点 {node_id} 至少需要两条出边"

#### Scenario: parallel fork 出边不足
- **WHEN** parallel fork 节点只有一条出边
- **THEN** 校验失败，返回错误"并行网关 fork 节点 {node_id} 至少需要两条出边"

#### Scenario: parallel join 入边不足
- **WHEN** parallel join 节点只有一条入边
- **THEN** 校验失败，返回错误"并行网关 join 节点 {node_id} 至少需要两条入边"

#### Scenario: parallel join 出边数量不为一
- **WHEN** parallel join 节点有 0 条或 2+ 条出边
- **THEN** 校验失败，返回错误"并行网关 join 节点 {node_id} 必须有且仅有一条出边"

#### Scenario: inclusive fork 出边缺少条件
- **WHEN** inclusive fork 节点的某条非默认出边没有配置 condition
- **THEN** 校验失败，返回错误"包含网关 fork 节点 {node_id} 的出边 {edge_id} 缺少条件配置"

#### Scenario: parallel/inclusive 缺少 gateway_direction
- **WHEN** parallel 或 inclusive 节点没有配置 gateway_direction 属性
- **THEN** 校验失败，返回错误"节点 {node_id} 类型 {type} 必须配置 gateway_direction（fork 或 join）"

#### Scenario: 非法节点类型
- **WHEN** 节点的 type 不在已注册的合法类型中
- **THEN** 校验失败，返回错误"节点 {node_id} 的类型 {type} 不合法"

#### Scenario: 未实现节点类型的 warning
- **WHEN** workflow_json 中包含 type="script" 的节点
- **THEN** 校验返回 warning 级别信息"节点 {node_id} 类型 script 已注册但执行逻辑尚未实现，当前版本不支持运行"
