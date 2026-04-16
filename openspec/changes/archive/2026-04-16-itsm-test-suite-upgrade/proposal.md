## Why

当前测试套件（18 个文件，~3100 行）使用纯 stdlib testing，无覆盖率追踪、无美化报告、无 BDD 框架。随着 ITSM 引擎（ClassicEngine、SmartEngine）复杂度上升，需要：1）提升 TDD 测试的可观测性（覆盖率 + 报告），2）引入 BDD 基础设施为引擎行为测试做准备。

## What Changes

- 引入 **gotestsum** 作为 CLI 工具（不进 go.mod），提供彩色终端输出 + JUnit XML 报告
- 新增 Makefile targets：`test-pretty`（美化输出）、`test-cover`（覆盖率 HTML）、`test-report`（综合报告）
- 升级现有 `test-llm` target 支持报告输出
- 引入 **godog** (cucumber/godog) 作为 Go 依赖，搭建 BDD 测试骨架
- 创建 `features/` 目录结构、`bdd_test.go` godog suite 入口、`steps_common_test.go` 共享 context
- `.gitignore` 补充测试产物（coverage.out, coverage.html, test-report.xml）
- 不编写实际 BDD feature 场景（后续按需添加）

## Capabilities

### New Capabilities
- `itsm-test-reporting`: 测试报告与覆盖率基础设施 — gotestsum 集成、Makefile targets、覆盖率 HTML 报告
- `itsm-bdd-infrastructure`: BDD 测试基础设施 — godog 集成、features 目录、suite 入口、共享 step context

### Modified Capabilities

(无)

## Impact

- `go.mod` / `go.sum`：新增 `github.com/cucumber/godog` 测试依赖
- `Makefile`：新增/修改 4 个 test targets
- `.gitignore`：新增测试产物排除
- 新增文件：`internal/app/itsm/features/` 目录、`bdd_test.go`、`steps_common_test.go`
- 不影响任何生产代码，不修改现有测试
