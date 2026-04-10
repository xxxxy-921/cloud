## 1. 后端基础模型与数据库

- [x] 1.1 新增 `internal/model/role.go`：Role 模型（Name, Code, Description, Sort, IsSystem），嵌入 BaseModel
- [x] 1.2 新增 `internal/model/menu.go`：Menu 模型（ParentID, Name, Type, Path, Icon, Permission, Sort, IsHidden, Children），嵌入 BaseModel
- [x] 1.3 修改 `internal/model/user.go`：移除 `Role string` 字段，新增 `RoleID uint` + `Role Role`（BelongsTo 关联）
- [x] 1.4 修改 `internal/database/database.go`：AutoMigrate 新增 Role 和 Menu 模型

## 2. Casbin 引擎集成

- [x] 2.1 `go get` 添加 casbin/v2 和 gorm-adapter/v3 依赖
- [x] 2.2 新增 `internal/casbin/casbin.go`：定义 RBAC model 字符串、初始化 enforcer + gorm-adapter、返回 enforcer 实例
- [x] 2.3 修改 `cmd/server/main.go`：在 IOC 容器中注册 Casbin enforcer provider

## 3. 后端 Repository 层

- [x] 3.1 新增 `internal/repository/role.go`：RoleRepo（Create, FindByID, FindByCode, List, Update, Delete, ExistsByCode, CountUsersByRoleID）
- [x] 3.2 新增 `internal/repository/menu.go`：MenuRepo（Create, FindByID, FindAll, FindByParentID, FindByPermission, Update, Delete, GetTree, HasChildren）

## 4. 后端 Service 层

- [x] 4.1 新增 `internal/service/casbin.go`：CasbinService（GetPoliciesForRole, SetPoliciesForRole, CheckPermission），注入 enforcer
- [x] 4.2 新增 `internal/service/role.go`：RoleService（List, GetByID, Create, Update, Delete），删除角色时联动清理 Casbin 策略
- [x] 4.3 新增 `internal/service/menu.go`：MenuService（GetTree, GetUserTree, Create, Update, Delete），GetUserTree 根据 Casbin 策略过滤用户可见菜单
- [x] 4.4 修改 `internal/service/auth.go`：Login/GetCurrentUser 响应增加 role 对象和 permissions 列表
- [x] 4.5 修改 `internal/service/user.go`：Create/Update 使用 RoleID，响应返回 role 对象

## 5. 后端 Middleware 与 Handler

- [x] 5.1 新增 `internal/middleware/casbin.go`：CasbinAuth 中间件，从 JWT context 取 roleCode，调用 enforcer.Enforce(roleCode, path, method)，含白名单路由跳过
- [x] 5.2 新增 `internal/handler/role.go`：RoleHandler（List, Create, GetByID, Update, Delete, GetPermissions, SetPermissions），注册路由
- [x] 5.3 新增 `internal/handler/menu.go`：MenuHandler（GetTree, GetUserTree, Create, Update, Delete），注册路由
- [x] 5.4 修改 `internal/handler/handler.go`：路由注册移除 RequireRole 中间件，改用 CasbinAuth；新增 /roles 和 /menus 路由组
- [x] 5.5 修改 `internal/handler/user.go` 和 `internal/handler/auth.go`：适配新的 User 响应格式（role 对象而非字符串）
- [x] 5.6 修改 `internal/pkg/token/jwt.go`：JWT claims 中 role 改为从 Role.Code 获取

## 6. Seed Init 机制

- [x] 6.1 新增 `internal/seed/roles.go`：内置角色 seed 数据（admin, user）
- [x] 6.2 新增 `internal/seed/menus.go`：内置菜单树 seed 数据（首页、系统管理及子菜单、各菜单的按钮权限）
- [x] 6.3 新增 `internal/seed/policies.go`：内置 Casbin 策略 seed 数据（admin 全部 API+菜单权限，user 基础权限）
- [x] 6.4 新增 `internal/seed/seed.go`：Seed 执行器，幂等检查（按 code/permission 匹配），输出摘要
- [x] 6.5 新增 `internal/seed/migrate.go`：旧 User.Role 字符串到 RoleID 的数据迁移逻辑
- [x] 6.6 修改 `cmd/server/main.go`：新增 `seed` CLI 子命令，调用 seed 执行器

## 7. IOC 容器注册汇总

- [x] 7.1 修改 `cmd/server/main.go`：注册所有新 provider（Casbin enforcer, RoleRepo, MenuRepo, RoleService, MenuService, CasbinService, RoleHandler, MenuHandler）

## 8. 前端状态与 API

- [x] 8.1 新增 `web/src/stores/menu.ts`：menuStore（menuTree, permissions[], init, clear），从 /api/v1/menus/user-tree 加载
- [x] 8.2 修改 `web/src/stores/auth.ts`：登录成功后并行调用 menuStore.init()，用户类型中 role 改为对象，增加 permissions
- [x] 8.3 修改 `web/src/lib/api.ts`：确保新 API 端点（/roles, /menus）的类型定义

## 9. 前端权限控制组件

- [x] 9.1 新增 `web/src/hooks/use-permission.ts`：usePermission(code) hook，检查 menuStore.permissions
- [x] 9.2 新增 `web/src/components/permission-guard.tsx`：PermissionGuard 组件，替代 AdminGuard
- [x] 9.3 修改 `web/src/App.tsx`：路由守卫从 AdminGuard 改为 PermissionGuard，新增 /roles 和 /menus 路由

## 10. 前端页面 - 角色管理

- [x] 10.1 新增 `web/src/pages/roles/index.tsx`：角色列表页（表格 + 搜索 + 分页）
- [x] 10.2 角色列表页：新增/编辑角色对话框（name, code, description, sort 表单）
- [x] 10.3 角色列表页：删除角色确认对话框，系统角色禁止删除
- [x] 10.4 角色列表页：权限分配对话框（菜单树勾选 + 保存）

## 11. 前端页面 - 菜单管理

- [x] 11.1 新增 `web/src/pages/menus/index.tsx`：菜单管理页（树形表格展示）
- [x] 11.2 菜单管理页：新增/编辑菜单对话框（parentId, name, type, path, icon, permission, sort, isHidden 表单）
- [x] 11.3 菜单管理页：删除菜单确认对话框

## 12. 前端动态菜单渲染

- [x] 12.1 修改 `web/src/components/layout/sidebar.tsx`：从 menuStore 读取菜单树渲染，替代 lib/nav 静态配置
- [x] 12.2 修改 `web/src/components/layout/header.tsx`：面包屑基于动态菜单路径生成
- [x] 12.3 修改 `web/src/pages/users/index.tsx`：操作按钮使用 usePermission hook 控制显隐
- [x] 12.4 修改 `web/src/pages/config/index.tsx`：操作按钮使用 usePermission hook 控制显隐

## 13. 清理与兼容

- [x] 13.1 移除 `internal/middleware/role.go`（RequireRole 中间件）
- [x] 13.2 更新 `web/src/lib/nav/` 为兼容层或完全移除（如 sidebar 已完全切换到动态菜单）
- [x] 13.3 更新用户管理页面：创建/编辑用户时角色选择从下拉字符串改为角色列表选择（RoleID）
