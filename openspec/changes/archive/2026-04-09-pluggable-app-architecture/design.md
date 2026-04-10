## Context

Metis 是一个 Go + React 单体应用，通过 samber/do IOC 管理依赖，所有模块（用户、角色、菜单、任务、通知、审计等）硬编码在 `main.go` 中注册。当前需要支持按场景分发不同功能组合的版本。

现有结构：
- 后端：`internal/handler/`、`internal/service/`、`internal/repository/` 等包，所有 provider 在 `main.go` 中逐一注册
- 前端：`web/src/pages/` 下所有页面，`App.tsx` 中硬编码路由
- 构建：`Makefile` 执行 `bun run build` + `go build`，产出单二进制

## Goals / Non-Goals

**Goals:**
- 新业务模块（AI、许可等）以 App 形式独立开发，实现统一接口
- 后端通过 Go build tags 在编译时真正排除未选模块的代码
- 前端通过构建脚本生成 registry 文件，Vite tree-shaking 排除未选模块
- 开发时全功能可用，体验与当前一致（`make dev` + `make web-dev`）
- 现有系统管理代码不移动、不重构

**Non-Goals:**
- 不做运行时动态加载插件（Go plugin 机制）
- 不将现有系统管理功能拆分为独立 App（它作为内核始终存在）
- 不做模块间的接口抽象层（直接通过 IOC 引用具体类型）
- 不做前端微前端架构
- 不做模块的独立版本管理

## Decisions

### D1: System 管理作为内核，不作为可插拔 App

**选择**: 用户、角色、菜单、认证、设置、任务、审计等现有功能保持为内核的一部分，代码位置不变。

**备选方案**:
- A) 所有功能都拆成 App（包括 System）— 过度抽象，几乎所有 App 都依赖 User/Role/Auth，拆分后依赖关系复杂化且无实际收益
- B) System 作为特殊的"必选 App" — 增加一层不必要的间接性

**理由**: 系统管理是任何版本都需要的基础设施，不存在"不需要用户管理"的场景。将其作为内核避免了不必要的抽象，同时现有代码零改动。

### D2: Go build tags + edition 导入文件控制后端模块

**选择**: 每个可选 App 包自带 `init()` 注册，在 `cmd/server/` 下用 edition 文件（带 build tag）控制哪些 App 被导入。默认（无 tag）编译全部模块。

**备选方案**:
- A) 每个 App 的 register.go 加 build tag — 默认构建不含可选模块，开发时需手动加 tag，影响开发体验
- B) Go plugin 动态加载 — 平台受限（不支持 Windows），运行时开销，调试困难

**理由**: edition 文件方式让默认 `go build` 就是全功能版，开发零摩擦。裁剪时加一个 tag（如 `-tags edition_lite`）即可。

### D3: 前端构建脚本生成 registry + git 提交全量版

**选择**: `web/src/apps/registry.ts` 在 git 中维护全量版本（导入所有模块）。生产构建前由 `scripts/gen-registry.sh` 根据 `APPS` 环境变量覆盖此文件，生成只导入所需模块的版本。Vite tree-shaking 排除未导入模块。

**备选方案**:
- A) 运行时条件加载 — 代码仍在 bundle 中，无法真正排除
- B) `import.meta.env` 条件导入 — Vite/Rollup 对动态 import 的死代码消除不够可靠
- C) 多 Vite 配置/多入口 — 维护成本高

**理由**: 生成文件方式简单可靠，开发时 git 里的全量文件直接可用，构建时脚本覆盖实现裁剪。一个 ~15 行的 shell 脚本解决问题。

### D4: App 之间通过 IOC 容器直接引用内核服务

**选择**: 可选 App 需要内核能力（如 UserService）时，直接通过 `do.MustInvoke` 从 IOC 容器获取具体类型。

**备选方案**:
- A) 定义接口层隔离 — 当前只有 3-5 个模块，接口抽象是过度工程
- B) 通过事件总线解耦 — 增加复杂度，调试困难

**理由**: IOC 容器已经提供了依赖管理，直接引用具体类型简单高效。如果未来模块数量增长到需要隔离，再引入接口层也不迟。

### D5: 每个 App 的 Seed 独立管理菜单和 Casbin 策略

**选择**: App 接口包含 `Seed()` 方法，每个 App 在其中注册自己的菜单树和 admin 角色的 Casbin 策略。菜单通过 `parentId` 挂在已有的菜单节点下（如系统管理目录下）或创建新的顶级目录。

**理由**: 与现有 seed 机制一致，每个 App 自治管理自己的菜单和权限，启动时按顺序执行。

## Risks / Trade-offs

- **[风险] edition 文件膨胀** — 如果版本组合很多（>5），edition 文件数量增长 → 缓解：用 Makefile 动态生成 edition 文件，或改用 App 级 build tag 组合
- **[风险] registry.ts 被构建覆盖后忘记恢复** → 缓解：构建脚本完成后自动 `git checkout web/src/apps/registry.ts` 恢复，或使用临时文件
- **[权衡] App 直接引用内核类型，模块边界较弱** → 当前模块数少，简单优先；未来可加接口层
- **[权衡] 前端全量 registry 在 git 中需手动维护** → 新增 App 时需在 registry.ts 加一行 import，遗忘则开发时不可见
