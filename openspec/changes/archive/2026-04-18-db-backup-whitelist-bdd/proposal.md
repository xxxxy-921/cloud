## Why

Server access BDD 验证了智能引擎的分支决策能力，但尚未覆盖 **Action 节点**（HTTP webhook 自动触发）。生产数据库备份白名单临时放行是一个典型的"预检动作→人工审批→放行动作"流程，能验证智能引擎对 Action 节点的编排能力、动作执行记录的正确性、以及并行工单间 Action 记录的隔离性。参考 bklite-cloud 的 `db_backup_whitelist_action_flow` BDD 设计。

## What Changes

- 新增 `db_backup_whitelist_action_flow.feature`：3 个场景覆盖完整流程、权限校验、并行工单隔离
- 新增 `db_backup_support_test.go`：协作规范、2 组 case payload、LLM workflow 生成、smart service 发布（含 2 个 ServiceAction：precheck + apply）
- 新增 `steps_db_backup_test.go`：db backup 专属 step definitions（LocalActionReceiver、action 断言等）
- 修改 `bdd_test.go`：注册 `registerDbBackupSteps`
- 扩展 `engine/executor_action.go` 的 `replaceTemplateVars`：支持 `{{ticket.form_data.*}}` 模板变量
- 新增 `syncActionSubmitter` 测试基础设施：同步执行 action 任务并自动 progress

## Capabilities

### New Capabilities
- `itsm-bdd-db-backup-whitelist`: 数据库备份白名单临时放行 BDD 测试套件，覆盖 Action 节点自动触发、预检→审批→放行完整流程、权限校验、并行工单 Action 隔离

### Modified Capabilities
- `itsm-bdd-infrastructure`: 新增 syncActionSubmitter（同步执行 action 任务的 TaskSubmitter 实现）和 LocalActionReceiver（测试用 HTTP 接收器），可被后续含 Action 节点的 BDD 场景复用

## Impact

- **测试文件**: `internal/app/itsm/features/` 新增 1 个 feature 文件，`internal/app/itsm/` 新增 2 个 test 文件，修改 2 个
- **生产代码**: `engine/executor_action.go` 扩展模板变量支持 `{{ticket.form_data.*}}`（向后兼容）
- **依赖**: 复用现有 smart engine + action executor 基础设施
- **运行条件**: 需要 `LLM_TEST_*` 环境变量（与现有 BDD 一致）
- **无 breaking change**: 模板变量扩展向后兼容，其余为纯新增测试
