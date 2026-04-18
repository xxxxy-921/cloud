## Context

ITSM BDD 测试套件已覆盖：VPN（单级审批 + 分支路由）、DB Backup（action 节点 + 并行隔离）、Server Access（多分支决策）。Boss 串签审批是最后一个核心业务模式——两级串签（首级指定用户 → 二级部门岗位），也是第一个涉及混合参与者类型和复杂表单（含结构化明细表格）的 BDD。

已有基础设施：bddContext + SQLite in-memory DB、testOrgService（org resolver）、testAgentProvider、syncActionSubmitter、LocalActionReceiver、LLM 工作流生成、`智能引擎执行决策循环直到工单完成` 重试步骤。

参考实现：`bklite-cloud/server/tests/bdd/itsm/features/complex_form_two_level_serial_approval.feature`。

## Goals / Non-Goals

**Goals:**
- 验证智能引擎能连续 3 步正确决策：首级审批(user) → 二级审批(position_department) → complete
- 验证混合参与者类型（user 直接指派 + position_department 岗位解析）都能正确创建 assignment
- 验证审批隔离：二级审批人不能操作首级审批，反之亦然
- 验证复杂表单数据（含 resource_items 结构化明细表格）在工单上完整保留
- 验证整个串签审批链上用户解析和认领/审批权限控制

**Non-Goals:**
- 驳回回流（seed 协作规范明确不生成驳回分支）
- 服务台对话交互（draft_prepare/draft_confirm 等前端交互链路）
- Action 节点（Boss 无 action，已在 DB Backup 中覆盖）
- SLA 和升级规则验证

## Decisions

**D1: 协作规范直接复用 seed.go 中的 Boss 规范**
- 规范已包含完整的字段要求和串签审批链描述
- 轻微调整措辞使 AI 更可靠：明确"首级审批通过后才能安排二级审批"和"二级审批通过后必须立即结束"

**D2: 4 个 Scenario 覆盖 4 个维度**

| Scenario | 维度 | 难度 |
|----------|------|------|
| 完整串签流程 | 端到端 happy path | 基础 |
| 审批隔离与权限边界 | 首级/二级审批人互不可见 | 中等 |
| 复杂表单明细保留 | resource_items 数组跨工单保留 | 中等 |
| 并行串签工单隔离 | 两张工单的审批指派独立 | 高 |

**D3: 复用 db-backup 的完成重试机制**
- 最终 `complete` 步骤使用 `智能引擎执行决策循环直到工单完成`（已在 steps_db_backup_test.go 注册）
- 需要将此步骤提升到 steps_common_test.go 作为共享步骤

**D4: 不需要 LocalActionReceiver / syncActionSubmitter**
- Boss 没有 action 节点，不需要 webhook 接收和同步执行
- 服务发布只需 ServiceCatalog + Priority + Agent + ServiceDefinition（无 ServiceAction）

## Risks / Trade-offs

- **[AI 连续多步决策]** 需要 AI 做 3 次正确决策（首审 → 二审 → complete），gpt-4o 不确定性可能导致偶发失败 → 最后一步使用重试机制，中间步骤通过强化协作规范提升可靠性
- **[participant_type: user 需要解析]** AI 需要调用 `decision.resolve_participant` 获取 serial-reviewer 的 user_id → 协作规范中明确指定用户名
- **[步骤注册共享]** `智能引擎执行决策循环直到工单完成` 目前只在 steps_db_backup_test.go 中注册，需提升到 common → 小改动，不影响现有测试
