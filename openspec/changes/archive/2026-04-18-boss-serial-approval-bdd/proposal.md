## Why

已有的 BDD 测试覆盖了单级审批（VPN）、动作节点（DB Backup）和分支路由（Server Access），但尚未覆盖 **两级串签审批** 这一核心业务模式。高风险变更协同申请（Boss）是 ITSM 种子服务之一，其"指定用户首审 → 部门岗位二审 → 完成"的串签流程引入了混合参与者类型（user + position_department）、复杂表单（含结构化明细表格）和审批隔离等新维度，需要通过 BDD 端到端验证智能引擎的连续多步决策能力。

## What Changes

- 新增 `features/boss_serial_approval.feature` 文件，包含 4 个 Scenario 覆盖串签审批全流程
- 新增 `boss_support_test.go`：协作规范、复杂表单 case payload（含 resource_items 明细表格）、LLM 工作流生成、服务发布
- 新增 `steps_boss_test.go`：Boss 串签专属 step definitions（Given/When/Then）
- 复用已有 `steps_common_test.go` 基础设施（bddContext、participant 表、decision cycle steps）
- 复用 db-backup 中引入的 `智能引擎执行决策循环直到工单完成` 步骤处理 AI 非确定性

## Capabilities

### New Capabilities
- `boss-serial-approval-bdd`: BDD 测试覆盖高风险变更协同申请的两级串签审批流程，验证混合参与者类型、复杂表单保留、审批隔离和连续多步 AI 决策

### Modified Capabilities

## Impact

- `internal/app/itsm/features/` — 新增 feature 文件
- `internal/app/itsm/*_test.go` — 新增 support 和 steps 文件
- `internal/app/itsm/bdd_test.go` — 注册新 step definitions
- 不涉及生产代码变更（所有 participant_type: user 和 position_department 的处理路径已在 db-backup BDD 中修复）
