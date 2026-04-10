## Why

当前系统所有功能模块（用户管理、角色、菜单、任务、通知、审计等）硬编码在一起，无法按需裁剪。需要支持按场景分发不同版本（如"系统管理+AI"、"系统管理+许可"），在开发时保持全功能体验，打包时选择性包含模块。

## What Changes

- 新增 `internal/app/` 包，定义统一的 App 接口和全局注册表
- 将现有系统管理功能保持为内核（不移动代码），新业务模块以 App 形式开发
- 新增 Go build tags + edition 导入文件，控制后端编译时模块裁剪
- 新增前端模块注册机制（`web/src/apps/`），App 页面通过 registry 注册路由
- 新增构建脚本，打包前根据环境变量生成前端 registry 文件，实现 tree-shaking 排除
- 修改 `main.go`，增加 App 引导循环（Models → Providers → Seed → Routes → Tasks）
- 修改 `App.tsx`，合并内核路由与 App 插件路由
- 修改 Makefile，支持 `EDITION` 和 `APPS` 参数控制打包范围

## Capabilities

### New Capabilities
- `app-registry`: 后端 App 接口定义、全局注册表、拓扑排序引导，以及前端模块注册机制
- `build-editions`: Go build tags + edition 文件 + 前端 registry 生成脚本 + Makefile 集成，实现编译时模块裁剪

### Modified Capabilities
- `server-bootstrap`: main.go 增加 App 引导循环，在现有内核初始化后遍历注册的 App 完成 Models/Providers/Seed/Routes/Tasks
- `frontend-routing`: App.tsx 合并 App 插件路由到现有路由树中
- `web-embed`: 构建流程增加 registry 生成步骤

## Impact

- **后端**: `cmd/server/main.go` 新增 ~15 行引导循环；新增 `internal/app/` 包（~50 行接口+注册表）；新增 `cmd/server/edition_*.go` 文件
- **前端**: 新增 `web/src/apps/registry.ts`（~20 行）；`App.tsx` 新增 1 行路由合并
- **构建**: 新增 `scripts/gen-registry.sh`（~15 行）；Makefile 新增 EDITION/APPS 参数
- **现有代码**: 系统管理相关代码不移动、不修改
- **新模块开发**: 后端在 `internal/app/<name>/` 下实现 App 接口；前端在 `web/src/apps/<name>/` 下注册路由
