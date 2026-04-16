# itsm-test-reporting

## Purpose

提供测试结果的多种输出格式和报告生成能力，包括美化终端输出、覆盖率 HTML 报告、JUnit XML 报告等。

## Requirements

### Requirement: 美化测试终端输出
系统 SHALL 提供 `make test-pretty` target，使用 gotestsum 的 testdox 格式输出测试结果，包含彩色 PASS/FAIL 标记和测试耗时。

#### Scenario: gotestsum 已安装时输出 testdox 格式
- **WHEN** 执行 `make test-pretty` 且 gotestsum 已安装
- **THEN** 终端输出为 testdox 格式（每个测试一行，✓/✗ 标记）

#### Scenario: gotestsum 未安装时 fallback
- **WHEN** 执行 `make test-pretty` 且 gotestsum 未安装
- **THEN** 自动 fallback 到 `go test -v ./...`，不报错

### Requirement: 测试覆盖率 HTML 报告
系统 SHALL 提供 `make test-cover` target，生成 HTML 格式的行级覆盖率报告。

#### Scenario: 生成覆盖率报告
- **WHEN** 执行 `make test-cover`
- **THEN** 生成 `coverage.out`（原始数据）和 `coverage.html`（HTML 报告）
- **AND** 终端输出总覆盖率百分比

### Requirement: 综合测试报告
系统 SHALL 提供 `make test-report` target，同时生成 JUnit XML 报告和覆盖率 HTML 报告。

#### Scenario: 生成综合报告
- **WHEN** 执行 `make test-report` 且 gotestsum 已安装
- **THEN** 生成 `test-report.xml`（JUnit XML）、`coverage.out` 和 `coverage.html`

### Requirement: LLM 测试报告升级
现有 `make test-llm` target SHALL 支持可选的报告输出模式。

#### Scenario: test-llm 带报告输出
- **WHEN** 执行 `make test-llm-report` 且 gotestsum 已安装
- **THEN** 生成 `test-llm-report.xml`（JUnit XML）并使用 testdox 格式终端输出

### Requirement: 测试产物被 gitignore
所有测试产物 SHALL 被 `.gitignore` 排除。

#### Scenario: 测试产物不被追踪
- **WHEN** `.gitignore` 包含测试产物条目
- **THEN** `coverage.out`、`coverage.html`、`test-report.xml`、`test-llm-report.xml` 不被 git 追踪
