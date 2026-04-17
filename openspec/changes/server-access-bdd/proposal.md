## Why

VPN 开通申请的 BDD 已经验证了双引擎（经典+智能）在 2-way 路由下的正确性。现在需要为**生产服务器临时访问申请**建立 BDD 测试，验证智能引擎在 3-way 路由场景下的分支决策能力，包括模糊语义判定和审批责任边界。该场景参考 bklite-cloud 的 `server_access_branch_decision` BDD 设计。

## What Changes

- 新增 `server_access_branch_decision.feature`：5 个场景覆盖运维/网络/安全 3 条分支 + 边界语义 + 责任边界
- 新增 `server_access_support_test.go`：协作规范、4 组 case payload、LLM workflow 生成、smart service 发布
- 新增 `steps_server_access_test.go`：server access 专属 step definitions（约 6-7 个新 step）
- 修改 `bdd_test.go`：注册 `registerServerAccessSteps`

## Capabilities

### New Capabilities
- `itsm-bdd-server-access`: 生产服务器临时访问申请的 BDD 测试套件，覆盖智能引擎 3-way 分支决策、模糊语义判定、审批责任边界验证

### Modified Capabilities
- `itsm-bdd-infrastructure`: 新增通用断言 step（岗位分配验证、可见性验证、越权认领/审批失败），可被后续 BDD 场景复用

## Impact

- **测试文件**: `internal/app/itsm/features/`、`internal/app/itsm/` 下新增 3 个文件，修改 1 个
- **依赖**: 复用现有 smart engine 基础设施（`bddContext`、`SmartEngine`、`testAgentProvider` 等）
- **运行条件**: 需要 `LLM_TEST_*` 环境变量（与现有 BDD 一致）
- **无 breaking change**: 纯新增测试，不影响生产代码
