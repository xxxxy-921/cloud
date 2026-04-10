## 1. 后端 App 注册表

- [x] 1.1 创建 `internal/app/app.go`：定义 App 接口（Name/Models/Seed/Providers/Routes/Tasks）和全局注册表（Register/All 函数）
- [x] 1.2 创建 `internal/app/scheduler.go`：定义 `TaskDefinition` 类型别名或重导出，避免 App 接口直接依赖 scheduler 包（如需要）

## 2. 后端 App 引导循环

- [x] 2.1 修改 `cmd/server/main.go`：在现有内核初始化后，增加 for 循环遍历 `app.All()`，依次调用 Models（AutoMigrate）→ Providers → Seed → Routes → Tasks
- [x] 2.2 验证无注册 App 时（空注册表），引导循环无副作用，现有功能不受影响

## 3. Go Edition 导入文件

- [x] 3.1 创建 `cmd/server/edition_full.go`：带 `//go:build !edition_lite` 约束，作为将来 blank-import 可选 App 的位置（初始为空注释占位）
- [x] 3.2 创建 `cmd/server/edition_lite.go`：带 `//go:build edition_lite` 约束，空文件

## 4. 前端模块注册机制

- [x] 4.1 创建 `web/src/apps/registry.ts`：定义 AppModule 类型（name + routes），实现 registerApp() 和 getAppRoutes() 函数
- [x] 4.2 修改 `web/src/App.tsx`：在 DashboardLayout 的 children 路由中合并 `getAppRoutes()` 返回的路由

## 5. 前端 Registry 生成脚本

- [x] 5.1 创建 `scripts/gen-registry.sh`：根据 APPS 环境变量生成 `web/src/apps/registry.ts`，默认（无 APPS）生成全量版
- [x] 5.2 脚本增加构建后恢复逻辑：使用 `cp` 备份 + 还原，或 `git checkout` 恢复

## 6. Makefile 集成

- [x] 6.1 修改 Makefile `build` target：支持 `EDITION` 参数传递 go build tag，支持 `APPS` 参数调用 gen-registry.sh
- [x] 6.2 验证 `make dev` 和 `make web-dev` 不受影响，无需传额外参数

## 7. 验证

- [x] 7.1 验证 `make build` 全功能构建正常（无参数，等同当前行为）
- [x] 7.2 验证 `make build EDITION=edition_lite APPS=system` 构建的二进制不含可选模块代码
- [x] 7.3 验证开发流程不变：`make dev` + `make web-dev` 全功能可用
