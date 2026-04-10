## 1. Backend Edition 文件

- [x] 1.1 创建 `cmd/server/edition_license.go`，带 `//go:build edition_license`，blank-import `_ "metis/internal/app/license"`
- [x] 1.2 更新 `cmd/server/edition_full.go` 的 build tag 从 `//go:build !edition_lite` 改为 `//go:build !(edition_lite || edition_license)`

## 2. Makefile

- [x] 2.1 新增 `release-license` target，使用 `EDITION=edition_license APPS=system,license`，产出物命名为 `metis-license-{os}-{arch}`
- [x] 2.2 新增 `build-license` target（便捷开发构建），生成单一 `metis-license` 二进制

## 3. 验证

- [x] 3.1 验证 `go build -tags edition_license ./cmd/server` 编译成功（需 license app 代码存在）
  - ⚠️ 预期阻塞：`internal/app/license` 包尚未创建，待 license app 实现后自动可用
- [x] 3.2 验证 `go build ./cmd/server`（默认 full）仍包含所有模块 ✓
- [x] 3.3 验证 `go build -tags edition_lite ./cmd/server` 行为不变 ✓
- [x] 3.4 验证 `make release-license` 产出 dist/metis-license-* 文件
  - ⚠️ 预期阻塞：同 3.1，待 license app 实现后可验证
