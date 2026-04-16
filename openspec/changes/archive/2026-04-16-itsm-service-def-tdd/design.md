## Context

`WorkflowGenerateService` 是 ITSM 服务定义的核心 LLM 管道，将自然语言协作规范转换为 BPMN 工作流 JSON。管道包含多个阶段：prompt 构建 → LLM 调用 → JSON 提取 → 结构校验 → 重试循环。目前纯函数（`extractJSON`、`buildUserMessage`）和 LLM 集成均无测试覆盖。

现有测试基础设施（`test_helpers_test.go`）提供了 in-memory SQLite、service factory、HTTP handler 测试工具，可以复用。

## Goals / Non-Goals

**Goals:**
- 为 `extractJSON`、`buildUserMessage`、`buildActionsContext` 提供确定性单元测试
- 用环境变量门控的 LLM 集成测试验证结构化提取能力（协作规范 → 合法工作流 JSON）
- 提供 `make test-llm` 快捷入口 + `.env.test` 凭据隔离
- 引入 BDD 基础设施（空框架 + 标记），为后续 BDD 用例做准备

**Non-Goals:**
- 不编写实际 BDD 测试用例
- 不修改 `WorkflowGenerateService` 生产代码
- 不测试 prompt 质量（那是 BDD 的范畴）
- 不 mock LLM —— 单元测试覆盖纯函数，集成测试直接调真实 LLM

## Decisions

### D1: 所有测试放一个文件 `workflow_generate_test.go`

**选择**: 单文件，通过 `TestExtract*` / `TestBuild*` / `TestLLM*` 命名前缀区分层次。

**理由**: 测试量级不大（~10 个 case），拆多文件增加认知负担。可以通过 `-run` 精确选择。

**替代方案**: 按层拆为 `extract_json_test.go` + `workflow_generate_integration_test.go` —— 被放弃因为当前规模不需要。

### D2: LLM 集成测试通过环境变量门控

**选择**: 检查 `LLM_TEST_BASE_URL` + `LLM_TEST_API_KEY` + `LLM_TEST_MODEL`，缺任一即 `t.Skip`。

**理由**: 比 build tag 更灵活（CI 中设 secret 即可运行），比 `-short` 更语义化。

**替代方案**: `//go:build integration` build tag —— 被放弃因为需要额外 `-tags` 参数，容易忘记。

### D3: LLM 集成测试使用精简 system prompt，不走 Agent/Model/Provider 链路

**选择**: 测试中硬编码精简版 system prompt，直接通过 `llm.NewClient()` 创建客户端。

**理由**: TDD 目标是验证提取管道（LLM → extractJSON → ValidateWorkflow），不是验证 Agent 配置。精简 prompt 减少外部依赖、提高稳定性。

**替代方案**: 从 seed 数据加载 `itsm.generator` Agent 的 system prompt —— 留给 BDD 阶段。

### D4: `.env.test` 存放测试凭据

**选择**: 项目根目录 `.env.test`，格式为 `KEY=VALUE`（无 export 前缀），`.gitignore` 排除。

**理由**: 与项目已有的 `.env` / `.env.local` gitignore 模式一致。`Makefile` 通过 `cat .env.test | xargs` 加载。

### D5: LLM 集成测试的断言策略 — 结构约束而非精确匹配

**选择**: 断言基于结构不变量：
- 输出为合法 JSON
- 恰好 1 个 start 节点，≥1 个 end 节点
- 所有 edge 引用的 source/target 存在于 nodes 中
- 无 error 级别 `ValidationError`（warning 可接受）

**理由**: LLM 输出非确定性，同一 prompt 两次调用产生不同结构。只有结构约束是稳定可断言的。

## Risks / Trade-offs

- **LLM 测试不稳定**: LLM 可能偶尔输出无法解析的格式 → 集成测试设 `t.Parallel()` 不阻塞，失败信息包含完整 LLM 响应便于诊断
- **外部服务依赖**: LLM 端点不可用时集成测试全 skip → 环境变量门控已覆盖，但 CI 中需确保服务可达
- **精简 prompt 与生产 prompt 行为差异**: 精简 prompt 可能比生产 prompt 更容易通过 → 可接受，BDD 阶段会用完整 prompt 补充
