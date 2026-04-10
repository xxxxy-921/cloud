## Why

Metis 的许可管理功能需要作为独立产品发布——「Metis 许可管理版」，仅包含内核（系统管理）+ license app，不含 identity 等其他 App。现有 edition 体系支持 full 和 lite 两种，需要新增 `edition_license` 变体来支撑这个产品形态。提前规划打包，确保 license app 开发完成后可以直接构建分发。

## What Changes

- 新增 `cmd/server/edition_license.go`，带 `//go:build edition_license` tag，仅 import license app
- 调整 `edition_full.go` 的 build tag 条件，排除新增的 edition tag
- `gen-registry.sh` 无需修改，已支持 `APPS=system,license` 参数
- Makefile 新增 `release-license` 便捷 target，封装 `EDITION=edition_license APPS=system,license make release`
- 产出物命名：`dist/metis-license-{os}-{arch}`，区别于主产品

## Capabilities

### New Capabilities
- `license-edition`: 许可管理版 edition 定义——Go build tag 文件、Makefile target、构建产出物命名

### Modified Capabilities
- `build-editions`: 现有 edition_full.go 的 build tag 条件需更新，从 `!edition_lite` 变为排除所有自定义 edition（`!edition_lite && !edition_license`）

## Impact

- **构建系统**：`cmd/server/edition_license.go` 新增文件，`edition_full.go` build tag 调整
- **Makefile**：新增 `release-license` target
- **前端**：无代码改动，构建时通过 `APPS=system,license` 控制模块裁剪
- **产出物**：dist/ 目录下新增 `metis-license-*` 二进制文件
- **依赖**：license app 必须先实现完成（依赖 `license-product-management` change）
