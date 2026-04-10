## MODIFIED Requirements

### Requirement: Go edition 导入文件
系统 SHALL 在 `cmd/server/` 下提供 edition 文件来控制可选 App 的编译。默认的 `edition_full.go` 的 build tag SHALL 从 `//go:build !edition_lite` 更新为 `//go:build !(edition_lite || edition_license)`，排除所有自定义 edition。`edition_lite.go` 不变。

#### Scenario: 默认构建包含全部模块
- **WHEN** 执行 `go build ./cmd/server` 不带任何 build tag
- **THEN** `edition_full.go` SHALL 生效，所有可选 App SHALL 被编译进二进制

#### Scenario: lite 版本排除可选模块
- **WHEN** 执行 `go build -tags edition_lite ./cmd/server`
- **THEN** `edition_lite.go` SHALL 生效，可选 App 的代码 SHALL 不被编译

#### Scenario: license 版本不触发 full
- **WHEN** 执行 `go build -tags edition_license ./cmd/server`
- **THEN** `edition_full.go` SHALL 不生效（被 build tag 排除），仅 `edition_license.go` 的 import 生效

#### Scenario: 新增 edition 变体
- **WHEN** 需要新的版本组合
- **THEN** 开发者 SHALL 创建新的 `edition_<name>.go` 文件，并在 `edition_full.go` 的 build tag 中添加对应的排除条件
