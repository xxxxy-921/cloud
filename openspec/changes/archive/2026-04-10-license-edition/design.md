## Context

Metis 使用 Go build tags + 前端 registry 生成脚本实现模块裁剪。现有两个 edition：`edition_full`（全量，默认）和 `edition_lite`（仅内核）。现在需要新增 `edition_license`，包含内核 + license app。

当前 `edition_full.go` 使用 `//go:build !edition_lite` 作为守卫条件——即"非 lite 就是 full"。随着 edition 增多，这个模式需要演进。

## Goals / Non-Goals

**Goals:**
- 新增 `edition_license` build tag，编译出仅含内核 + license app 的二进制
- 保持 edition 体系的可扩展性，未来新增 edition 时改动最小
- 提供 `make release-license` 一键构建便捷命令

**Non-Goals:**
- 不改变现有 `edition_lite` 的行为
- 不修改 `gen-registry.sh` 脚本（它已经支持 APPS 参数）
- 不引入 CI/CD 自动化（后续按需添加）

## Decisions

### D1: edition_full.go 的 build tag 策略

**决策**：采用显式排除模式——`//go:build !edition_lite && !edition_license`。每新增一个 edition，需在 full 的 build tag 中加一个 `!edition_xxx`。

**备选方案**：
- A) 给 full 也加正向 tag `edition_full`：需要改 Makefile 默认行为，`go build` 不带 tag 时什么都编译不到
- B) 使用 `//go:build !(edition_lite || edition_license)` 语法：Go 1.17+ 支持，等价但更清晰

**理由**：方案 B 语法更清晰，且 Go 1.25 完全支持。随着 edition 增多可以用括号分组。选用 B。

### D2: 产出物命名

**决策**：许可管理版产出命名为 `metis-license-{os}-{arch}`，与主产品 `metis-{os}-{arch}` 区分。

**理由**：避免 dist/ 目录下文件互相覆盖，方便 CI 并行构建多个 edition。

### D3: Makefile target 设计

**决策**：新增 `release-license` target，内部调用 `make release EDITION=edition_license APPS=system,license`，并额外 rename 产出物加上 `-license` 后缀。

**备选**：使用 `BINARY_PREFIX` 参数——过度抽象，当前只有两个产品。

## Risks / Trade-offs

- **[edition_full build tag 维护负担]** → 每新增 edition 需更新 full 的排除列表。可接受，edition 数量有限（预期 < 5）。
- **[license app 未实现时 edition_license 编译失败]** → 此 change 仅在 license app 代码存在后才能实际构建。tasks 中标注依赖关系。
