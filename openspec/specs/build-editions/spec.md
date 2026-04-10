# Capability: build-editions

## Purpose
Controls compilation of optional App modules through Go build tags and frontend registry generation, enabling different product editions (full, lite, custom).

## Requirements

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

### Requirement: 前端 registry 生成脚本
系统 SHALL 提供 `scripts/gen-registry.sh` 脚本，根据 `APPS` 环境变量生成 `web/src/apps/registry.ts`。脚本 SHALL 始终包含 `import './system/module'`，并根据 APPS 变量中的值添加其他模块的导入。

#### Scenario: 指定模块生成 registry
- **WHEN** 执行 `APPS=system,ai ./scripts/gen-registry.sh`
- **THEN** 生成的 registry.ts SHALL 仅包含 system 和 ai 模块的 import 语句

#### Scenario: 未指定 APPS 变量
- **WHEN** 执行 `./scripts/gen-registry.sh` 不设 APPS 变量
- **THEN** 脚本 SHALL 生成包含所有已知模块的全量 import（等同于 git 中的默认版本）

### Requirement: Git 中的全量 registry 文件
`web/src/apps/registry.ts` SHALL 在 git 中提交一个包含所有模块导入的全量版本。此文件在开发模式下直接使用，仅在生产构建时被脚本覆盖。

#### Scenario: 开发时使用 git 中的全量文件
- **WHEN** 开发者执行 `make web-dev`
- **THEN** Vite dev server SHALL 使用 git 中的全量 registry.ts，所有模块可用

#### Scenario: 构建后恢复 registry
- **WHEN** 生产构建完成后
- **THEN** 构建流程 SHALL 恢复 registry.ts 到 git 中的全量版本（通过 `git checkout` 或使用临时文件）

### Requirement: Makefile 构建参数
Makefile SHALL 支持 `EDITION` 参数控制后端 build tag，`APPS` 参数控制前端模块裁剪。

#### Scenario: 全功能构建
- **WHEN** 执行 `make build`
- **THEN** SHALL 构建包含所有模块的全功能版本（后端无 tag，前端全量 registry）

#### Scenario: 指定版本构建
- **WHEN** 执行 `make build EDITION=edition_lite APPS=system`
- **THEN** 后端 SHALL 以 `-tags edition_lite` 编译，前端 SHALL 仅包含 system 模块

#### Scenario: 开发命令不受影响
- **WHEN** 执行 `make dev` 或 `make web-dev`
- **THEN** SHALL 与当前行为完全一致，不需要传任何额外参数
