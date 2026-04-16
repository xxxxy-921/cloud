## Context

项目使用纯 Go stdlib testing，18 个测试文件覆盖 license/itsm/ai/user 模块。现有 Makefile 有 `test`、`test-license`、`test-fuzz`、`test-llm` 四个 target，均为裸 `go test` 调用，无报告输出。ITSM 引擎（ClassicEngine 1400+ 行，token 推进、网关、子流程、边界事件）急需 BDD 级别的行为测试，但目前只有一个空的 `workflow_generate_bdd_test.go` 占位文件。

## Goals / Non-Goals

**Goals:**
- gotestsum CLI 工具集成，提供 testdox 格式彩色终端输出和 JUnit XML
- `go test -coverprofile` 集成，生成 HTML 覆盖率报告
- Makefile targets 统一测试报告入口
- godog 引入 go.mod，BDD suite 骨架搭建完成
- 共享 test context 结构体（db + engine + 当前实体），为后续 step definitions 打基础
- 一个空的 `.feature` 示例文件说明格式约定

**Non-Goals:**
- 不编写实际 BDD feature 场景或 step definitions（后续按需）
- 不引入 CI/CD pipeline（那是独立变更）
- 不修改现有测试代码
- 不引入 testify 或其他断言库（保持 stdlib 风格）

## Decisions

### D1: gotestsum 作为 CLI 工具安装，不进 go.mod

**选择**: `go install gotest.tools/gotestsum@latest` 或 `brew install gotestsum`。Makefile targets 检测是否安装，未安装则 fallback 到 `go test`。

**理由**: gotestsum 是测试运行器不是被测代码的依赖。不污染 go.mod，保持依赖干净。

**替代方案**: 把 gotestsum 作为 `tools.go` 管理 → 过重，项目无此模式。

### D2: godog 作为 go.mod 测试依赖

**选择**: `go get -t github.com/cucumber/godog@latest`，仅在 `_test.go` 文件中 import。

**理由**: godog 的 step definitions 需要在测试代码中 import `godog` 包，必须进 go.mod。但 `-t` 标记确保它只是测试依赖，不影响生产二进制。

### D3: BDD 文件组织 — features/ 在 itsm 包内

**选择**:
```
internal/app/itsm/
├── features/                    ← .feature 文件
│   └── .gitkeep
├── bdd_test.go                  ← godog suite 入口 (TestMain 或 TestFeatures)
└── steps_common_test.go         ← 共享 context + 通用 steps
```

**理由**: godog 的 `go test` 集成要求 `_test.go` 在被测包中。`features/` 子目录放 Gherkin 文件，godog 的 `WithPaths()` 指向它。

**替代方案**: 独立 `test/bdd/` 目录 → 需要 `_test` 包，无法访问内部类型，增加复杂度。

### D4: 共享 test context 结构体

**选择**:
```go
type bddContext struct {
    t       *testing.T
    db      *gorm.DB
    engine  *engine.ClassicEngine
    catalog *ServiceCatalog
    service *ServiceDefinition
    ticket  any          // 未来 Ticket 类型
    lastErr error
}
```

**理由**: 每个 Scenario 一个 context 实例，Given/When/Then steps 都操作同一个 context。这是 godog 的标准模式（ScenarioInitializer 注入）。

### D5: 覆盖率报告用内置 go tool cover

**选择**: `go test -coverprofile=coverage.out` + `go tool cover -html=coverage.out -o coverage.html`，不引入第三方覆盖率工具。

**理由**: Go 内置覆盖率工具已足够。HTML 报告清晰标注行级覆盖。JUnit XML 由 gotestsum 生成，两者互补。

### D6: 替换现有 BDD 占位文件

**选择**: 删除 `workflow_generate_bdd_test.go`（15 行纯注释），内容并入 `bdd_test.go` 的注释中。

**理由**: 新的 `bdd_test.go` 是真正的 godog suite 入口，占位文件不再需要。

## Risks / Trade-offs

- **godog 版本兼容**: godog v0.14+ 要求 Go 1.21+ → 项目用 Go 1.26，无风险
- **gotestsum 未安装时**: Makefile fallback 到 `go test`，功能不受影响但无美化输出
- **go.sum 膨胀**: godog 会拉入一些间接依赖（cucumber 消息协议等） → 可接受，只影响测试
- **BDD 骨架无实际测试**: 搭好基础设施但没有 feature → 不影响 `go test ./...`，空 features/ 目录 godog 会 skip
