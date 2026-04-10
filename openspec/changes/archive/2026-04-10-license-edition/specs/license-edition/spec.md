## ADDED Requirements

### Requirement: edition_license Go 构建文件
系统 SHALL 提供 `cmd/server/edition_license.go` 文件，带 `//go:build edition_license` 约束，仅 blank-import `metis/internal/app/license` 包。

#### Scenario: 使用 edition_license tag 构建
- **WHEN** 执行 `go build -tags edition_license ./cmd/server`
- **THEN** 编译产物 SHALL 包含内核 + license app，不包含 identity 等其他 App

#### Scenario: edition_license 文件不影响默认构建
- **WHEN** 执行 `go build ./cmd/server` 不带任何 tag
- **THEN** `edition_license.go` SHALL 不被编译，默认构建行为不变

### Requirement: Makefile release-license target
Makefile SHALL 提供 `release-license` target，一键交叉编译许可管理版二进制。产出物命名为 `metis-license-{os}-{arch}`，存放在 `dist/` 目录。

#### Scenario: 执行 release-license
- **WHEN** 执行 `make release-license`
- **THEN** SHALL 生成以下文件：`dist/metis-license-linux-amd64`、`dist/metis-license-linux-arm64`、`dist/metis-license-darwin-amd64`、`dist/metis-license-darwin-arm64`、`dist/metis-license-windows-amd64.exe`

#### Scenario: release-license 使用正确的 edition 和 APPS
- **WHEN** `release-license` target 执行构建
- **THEN** 后端 SHALL 使用 `-tags edition_license`，前端 SHALL 使用 `APPS=system,license` 生成 registry
