## Why

首页是一个纯占位页面（"欢迎使用 Metis，选择左侧菜单开始操作"），没有实际功能价值，登录后用户总是需要再点一次菜单才能开始工作。同时系统管理菜单排在许可管理前面，但作为基础设施模块它应该排在最后，让业务模块优先展示。

## What Changes

- **移除首页**：删除首页菜单 seed 数据、前端首页组件、对应路由定义
- **调整登录后落地页**：登录成功后自动导航到第一个可用模块的第一个菜单，而非 `/`
- **根路径重定向**：访问 `/` 时自动 redirect 到第一个可用菜单路径
- **调整菜单排序**：系统管理 sort 值改为 9999，确保它始终排在所有 App 模块之后

## Capabilities

### New Capabilities

（无新增能力）

### Modified Capabilities

- `seed-init`: 移除首页菜单 seed 数据；系统管理菜单 sort 从 100 改为 9999
- `frontend-routing`: 移除首页 index route；根路径 `/` 改为 redirect 到第一个可用菜单
- `nav-modules`: 移除首页相关的导航项；登录后默认选中第一个模块的第一个菜单

## Impact

- **后端 seed**：`internal/seed/menus.go` 删除 home 菜单项，修改系统管理 sort 值
- **后端 Casbin**：`internal/seed/casbin.go` 删除 home 相关策略（如有）
- **前端页面**：删除 `web/src/pages/home/` 目录
- **前端路由**：`web/src/App.tsx` 移除 index route，添加 redirect 逻辑
- **前端登录**：`web/src/pages/login/` 中 `navigate("/")` 改为导航到第一个可用菜单
- **无 API 变更**，无数据库 schema 变更
- **需要删除 DB 重新 seed**（用户确认不需要迁移）
