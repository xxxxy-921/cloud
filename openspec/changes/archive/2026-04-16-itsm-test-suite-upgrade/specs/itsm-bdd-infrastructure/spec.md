## ADDED Requirements

### Requirement: godog 测试依赖引入
项目 SHALL 在 go.mod 中引入 `github.com/cucumber/godog` 作为测试依赖。

#### Scenario: godog 可在测试代码中导入
- **WHEN** 测试文件 import `github.com/cucumber/godog`
- **THEN** `go test` 编译成功

### Requirement: BDD suite 入口文件
系统 SHALL 提供 `bdd_test.go` 作为 godog 测试 suite 的入口，使用 `godog.TestSuite` 配置 features 路径和 scenario initializer。

#### Scenario: godog suite 可运行
- **WHEN** 执行 `go test ./internal/app/itsm/ -run TestBDD -v`
- **THEN** godog suite 启动，扫描 `features/` 目录
- **AND** 无 feature 文件时不报错（0 scenarios, 0 steps）

### Requirement: features 目录结构
系统 SHALL 在 `internal/app/itsm/features/` 下创建目录结构，包含 `.gitkeep` 和一个示例 `.feature` 文件说明格式约定。

#### Scenario: features 目录存在且包含格式说明
- **WHEN** 查看 `internal/app/itsm/features/` 目录
- **THEN** 目录存在，包含 `example.feature`（注释说明格式约定，标记为 @wip 不执行）

### Requirement: 共享 BDD test context
系统 SHALL 提供 `steps_common_test.go`，定义 `bddContext` 结构体作为所有 step definitions 的共享状态容器。

#### Scenario: bddContext 包含核心字段
- **WHEN** 查看 `bddContext` 结构体
- **THEN** 包含以下字段：db (*gorm.DB)、lastErr (error)
- **AND** 提供 `reset()` 方法在每个 Scenario 前重置状态

### Requirement: BDD 测试可通过 Makefile 运行
系统 SHALL 提供 `make test-bdd` target 运行 BDD 测试。

#### Scenario: make test-bdd 运行 godog suite
- **WHEN** 执行 `make test-bdd`
- **THEN** 运行 `go test ./internal/app/itsm/ -run TestBDD -v`

### Requirement: 删除旧 BDD 占位文件
旧的 `workflow_generate_bdd_test.go` 占位文件 SHALL 被删除，其内容并入新的 `bdd_test.go` 注释中。

#### Scenario: 旧占位文件不存在
- **WHEN** 查看 `internal/app/itsm/` 目录
- **THEN** `workflow_generate_bdd_test.go` 不存在
