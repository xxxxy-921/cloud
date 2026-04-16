## Why

ITSM 服务定义的核心 LLM 管道（`WorkflowGenerateService`）目前零测试覆盖。`extractJSON`、`buildUserMessage` 等纯函数没有单元测试，LLM 结构化提取能力（协作规范 → 合法工作流 JSON）也没有集成测试验证。需要引入 TDD 测试确保基础能力可靠，同时为后续 BDD 场景测试铺好基础设施。

## What Changes

- 新增 `workflow_generate_test.go`，包含两层测试：
  - **Layer 1 单元测试**：`extractJSON`、`buildUserMessage`、`buildActionsContext` 的确定性测试
  - **Layer 2 LLM 集成测试**：通过环境变量门控（`LLM_TEST_BASE_URL` / `LLM_TEST_API_KEY` / `LLM_TEST_MODEL`），直接调用 LLM 验证结构化提取能力
- `.gitignore` 新增 `.env.test` 条目，防止测试凭据泄露
- `Makefile` 新增 `test-llm` target，从 `.env.test` 加载凭据运行 LLM 集成测试
- 引入 BDD 测试基础设施占位（文件 + 空框架），不含实际 BDD 用例

## Capabilities

### New Capabilities
- `itsm-service-def-tdd`: ITSM 服务定义的 TDD 测试体系 — 单元测试 + LLM 集成测试 + BDD 基础设施

### Modified Capabilities

(无)

## Impact

- 新增测试文件：`internal/app/itsm/workflow_generate_test.go`
- 修改：`.gitignore`（加 `.env.test`）、`Makefile`（加 `test-llm` target）
- 新增项目根目录文件：`.env.test`（本地凭据，不提交）
- 不影响任何生产代码
