## 1. TDD 报告基础设施

- [x] 1.1 `.gitignore` 添加测试产物条目（coverage.out, coverage.html, test-report.xml, test-llm-report.xml）
- [x] 1.2 Makefile 添加 `test-pretty` target（gotestsum testdox，带 fallback）
- [x] 1.3 Makefile 添加 `test-cover` target（coverprofile + HTML 报告 + 终端覆盖率摘要）
- [x] 1.4 Makefile 添加 `test-report` target（gotestsum JUnit XML + 覆盖率 HTML）
- [x] 1.5 Makefile 添加 `test-llm-report` target（LLM 测试 + 报告输出）
- [x] 1.6 更新 .PHONY 声明
- [x] 1.7 验证：运行 `make test-cover` 确认 coverage.html 生成

## 2. BDD 基础设施 — godog 引入

- [x] 2.1 `go get -t github.com/cucumber/godog@latest` 引入测试依赖
- [x] 2.2 创建 `internal/app/itsm/features/` 目录 + `.gitkeep`
- [x] 2.3 创建 `internal/app/itsm/features/example.feature`（@wip 标记，格式说明）
- [x] 2.4 创建 `internal/app/itsm/bdd_test.go`（godog TestSuite 入口）
- [x] 2.5 创建 `internal/app/itsm/steps_common_test.go`（bddContext 结构体 + reset）
- [x] 2.6 删除旧 `internal/app/itsm/workflow_generate_bdd_test.go` 占位文件
- [x] 2.7 Makefile 添加 `test-bdd` target
- [x] 2.8 验证：`go test ./internal/app/itsm/ -run TestBDD -v` 正常运行（0 scenarios）
- [x] 2.9 验证：`go test ./internal/app/itsm/` 全量测试不受影响
