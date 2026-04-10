## 1. 后端 Seed 数据调整

- [x] 1.1 `internal/seed/menus.go` — 删除首页（home）菜单项的 seed 数据
- [x] 1.2 `internal/seed/menus.go` — 将系统管理目录的 Sort 值从 100 改为 9999
- [x] 1.3 `internal/seed/seed.go` — 删除 user 角色的 home 相关 Casbin 策略

## 2. 前端首页移除

- [x] 2.1 删除 `web/src/pages/home/` 目录
- [x] 2.2 `web/src/App.tsx` — 移除首页 index route 的 `<HomePage />` 引用和 import
- [x] 2.3 `web/src/App.tsx` — 添加 `<DefaultRedirect />` 组件作为 index route，从 menu store 读取第一个可用菜单路径并 `<Navigate to={path} replace />`，fallback 到 `/users`

## 3. 前端导航适配

- [x] 3.1 检查 sidebar / nav 模块中是否有首页相关的硬编码导航项，如有则移除
- [x] 3.2 检查面包屑组件中是否有 "首页" 硬编码，如有则移除

## 4. 验证

- [x] 4.1 `go build -tags dev ./cmd/server/` 确认后端编译通过
- [x] 4.2 `cd web && bun run lint` 确认前端 lint 通过（本次改动无新增 lint 错误）
