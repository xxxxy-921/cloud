## Why

当前 Metis 的权限控制基于硬编码角色（admin/user），前端菜单写死在 `nav.ts` 中，无法灵活分配权限。随着系统功能增长，需要引入完整的 RBAC 权限体系：支持角色管理、菜单权限、功能（API）权限、按钮级权限，以及动态菜单。同时需要幂等的 seed init 机制来管理内置角色、菜单和权限策略的初始化。

## What Changes

- 引入 Casbin (casbin/v2 + gorm-adapter/v3) 作为权限引擎，替代现有的 `RequireRole` 中间件
- 新增 Role 模型（存储角色元数据：名称、编码、描述、排序、是否系统内置）
- 新增 Menu 模型（树形结构，支持 directory/menu/button 三种类型，含 permission 字段用于 Casbin 权限匹配）
- 将 User.Role 从字符串字段迁移为 User.RoleID 外键关联 Role 表
- **BREAKING**: 移除 `middleware.RequireRole()`，替换为 `middleware.CasbinAuth()` 统一权限中间件
- **BREAKING**: 前端 `nav.ts` 硬编码菜单替换为后端动态菜单 API (`GET /api/v1/menus/user-tree`)
- 新增 seed init CLI 子命令 (`metis seed`)，幂等初始化内置角色、菜单、Casbin 策略
- 新增角色管理页面（CRUD + 权限分配 UI）
- 前端支持按钮级权限控制（根据用户菜单权限隐藏/显示操作按钮）
- Casbin 策略同时管理 API 权限（path + method）和菜单权限（menu:xxx + read）

## Capabilities

### New Capabilities
- `casbin-engine`: Casbin 权限引擎集成，包括 model 定义、gorm-adapter 配置、enforcer IOC 注册、CasbinAuth 中间件
- `role-management`: 角色模型与 CRUD 服务，包括 Role 表、角色 API、角色管理前端页面
- `menu-system`: 菜单模型与动态菜单，包括 Menu 树形表、用户菜单树 API、前端动态菜单加载与渲染
- `permission-assignment`: 权限分配系统，包括角色-权限（Casbin policy）绑定、角色权限分配 API 与前端 UI
- `seed-init`: 幂等 seed 初始化，包括 CLI 子命令、内置角色/菜单/策略的 seed 数据定义与执行
- `button-permission`: 前端按钮级权限控制，根据用户菜单树中 button 类型条目控制操作按钮显隐

### Modified Capabilities
- `server-bootstrap`: 新增 Casbin enforcer 和 Role/Menu 相关 provider 到 IOC 容器，路由注册改为统一 CasbinAuth 中间件
- `database`: AutoMigrate 新增 Role、Menu、casbin_rule 表
- `frontend-routing`: 路由从静态定义改为动态菜单驱动，Sidebar 和路由注册基于后端返回的菜单树

## Impact

**后端**:
- `internal/model/`: 新增 role.go、menu.go；修改 user.go（Role string → RoleID FK）
- `internal/repository/`: 新增 role.go、menu.go
- `internal/service/`: 新增 role.go、menu.go、casbin.go
- `internal/handler/`: 新增 role.go、menu.go；修改 handler.go 路由注册
- `internal/middleware/`: 新增 casbin.go；删除 role.go
- `internal/seed/`: 新增 seed 包（menus.go、policies.go、roles.go）
- `cmd/server/main.go`: 新增 seed 子命令、Casbin 相关 IOC 注册
- `go.mod`: 新增 casbin/v2、gorm-adapter/v3 依赖

**前端**:
- `web/src/lib/nav.ts`: 移除硬编码菜单，改为从 store 读取
- `web/src/stores/menu.ts`: 新增菜单 store
- `web/src/stores/auth.ts`: 用户信息包含 role 对象和权限列表
- `web/src/components/layout/sidebar.tsx`: 基于动态菜单渲染
- `web/src/App.tsx`: 路由定义基于动态菜单
- `web/src/pages/roles/`: 新增角色管理页面（列表 + 权限分配）
- `web/src/hooks/use-permission.ts`: 新增按钮级权限 hook

**数据库**: 新增 Role、Menu 表，Casbin 自动创建 casbin_rule 表，User 表结构变更（Role→RoleID）

**Breaking Changes**: User API 响应格式变化（role 从字符串变为对象），前端菜单加载流程变化
