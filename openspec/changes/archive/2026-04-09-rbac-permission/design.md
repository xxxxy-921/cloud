## Context

当前 Metis 使用硬编码的 admin/user 双角色模型，权限校验通过 `RequireRole("admin")` 中间件实现，前端菜单在 `lib/nav/` 静态定义。随着功能增长，需要灵活的权限体系支持：动态角色、菜单权限、API 权限、按钮级权限。

现有架构的关键约束：
- Go 后端使用 Gin + GORM + samber/do IOC
- 前端 React 19 + Zustand + React Router 7
- SQLite 默认、PostgreSQL 可选
- 单二进制部署，嵌入前端静态资源
- User 模型当前有 `Role string` 字段（"admin"/"user"）

## Goals / Non-Goals

**Goals:**
- 引入 Casbin 作为统一权限引擎，管理 API 权限和菜单权限
- 支持动态角色 CRUD，角色与权限（Casbin policy）绑定
- 菜单从硬编码迁移为数据库驱动的树形结构，支持 directory/menu/button 三种类型
- 前端动态加载用户菜单树，按钮级权限控制
- 幂等 seed 机制初始化内置角色、菜单、Casbin 策略
- 平滑迁移：User.Role string → User.RoleID FK，保持 API 兼容性过渡

**Non-Goals:**
- 多角色绑定（一个用户只关联一个角色）
- 数据级权限（如"只能看自己部门的数据"）
- ABAC 属性级权限控制
- 组织架构/部门管理
- 权限审计日志

## Decisions

### D1: 使用 Casbin 作为权限引擎

**选择**: casbin/v2 + gorm-adapter/v3

**理由**: Casbin 是 Go 生态成熟的权限库，支持 RBAC/ABAC 多种模型，gorm-adapter 可复用现有 GORM 连接，策略存储在 `casbin_rule` 表中，无需额外基础设施。

**替代方案**:
- 自研权限表：灵活但工作量大，需要自行实现匹配逻辑
- OPA/Rego：过于重量级，适合微服务场景

**Model 定义** (RBAC with resource):
```
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
```

- `sub`: 角色编码（如 "admin"、"editor"）
- `obj`: 资源标识（API 路径如 "/api/v1/users" 或菜单权限如 "system:user:list"）
- `act`: 操作（HTTP method 如 "GET"、"POST" 或 "read"）

### D2: 角色模型设计

**选择**: 独立 Role 表 + User.RoleID 外键（单角色）

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint (BaseModel) | 主键 |
| Name | string | 角色显示名（如"管理员"） |
| Code | string (unique) | 角色编码（如"admin"），用于 Casbin subject |
| Description | string | 角色描述 |
| Sort | int | 排序权重 |
| IsSystem | bool | 系统内置角色不可删除 |

**User 变更**: 移除 `Role string`，新增 `RoleID uint` + `Role Role`（Belongs To 关联）

**理由**: 单角色简化了权限判断和 JWT claims，满足当前需求。多角色可后续通过 UserRole 中间表扩展，Casbin 的 `g` 定义天然支持。

### D3: 菜单模型设计（树形结构）

**选择**: 单表自引用 + ParentID 外键

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint (BaseModel) | 主键 |
| ParentID | *uint | 父菜单 ID，NULL 为顶级 |
| Name | string | 菜单名称 |
| Type | string | "directory" / "menu" / "button" |
| Path | string | 路由路径（menu 类型使用） |
| Icon | string | 图标名称 |
| Permission | string | 权限标识（如 "system:user:list"），用于 Casbin obj |
| Sort | int | 同级排序 |
| IsHidden | bool | 隐藏菜单（不在侧边栏显示但可访问） |
| Children | []Menu | GORM Preload 加载子节点 |

**三种类型的职责**:
- `directory`: 分组容器（如"系统管理"），无路由
- `menu`: 页面菜单（如"用户管理"），有路由路径
- `button`: 操作按钮（如"新增用户"），有 permission 标识，用于按钮级权限

### D4: JWT Claims 扩展

**选择**: JWT 中只携带 roleCode，权限列表通过登录/刷新时单独返回

**理由**: 权限列表可能很长，放入 JWT 会显著增大 token 体积。roleCode 用于后端 Casbin 校验（作为 subject），前端权限列表存储在 Zustand store 中。

**Auth API 响应变更**:
- `GET /api/v1/auth/me` 返回增加 `role: {id, name, code}` 对象和 `permissions: string[]`（用户拥有的菜单 permission 列表）
- Login 响应增加 `permissions: string[]`

### D5: CasbinAuth 中间件替代 RequireRole

**选择**: 单一 `CasbinAuth()` 中间件，从 JWT claims 获取 roleCode，以 (roleCode, requestPath, requestMethod) 调用 Casbin enforcer

**路由注册变更**:
```go
// Before
adminGroup := api.Group("/users").Use(middleware.RequireRole("admin"))

// After
api.Use(middleware.CasbinAuth(enforcer))  // 统一中间件
```

**白名单**: 公共路由（login、refresh、site-info GET）和已认证但不需要细粒度权限的路由（/auth/me、/auth/password、/auth/logout）跳过 Casbin 校验。白名单在中间件中硬编码定义。

### D6: Seed Init 机制

**选择**: CLI 子命令 `metis seed`，幂等执行

**执行逻辑**:
1. 初始化数据库连接
2. 检查并创建内置角色（admin、user），已存在则跳过
3. 检查并创建菜单树（系统管理、首页等），按 permission 字段做幂等判断
4. 加载 Casbin 策略（admin 角色的全部权限、user 角色的基本权限），使用 AddPolicies 批量写入
5. 输出执行结果摘要

**Seed 数据组织**: `internal/seed/` 包下分文件 roles.go、menus.go、policies.go

### D7: 前端动态菜单方案

**选择**: 新增 `menuStore` (Zustand)，登录后调用 `GET /api/v1/menus/user-tree` 获取当前用户菜单树

**流程**:
1. 登录成功 → 并行获取 /auth/me 和 /menus/user-tree
2. menuStore 存储完整菜单树 + 扁平化的 permission 列表
3. Sidebar 从 menuStore 读取菜单树渲染
4. React Router 路由仍然静态定义（避免动态路由的复杂性），菜单只控制可见性
5. AdminGuard 改为 PermissionGuard，基于 permission 标识判断

**按钮级权限**: `usePermission(code: string)` hook 检查 menuStore 中的 permission 列表

## Risks / Trade-offs

**[R1] User.Role → User.RoleID 迁移** → 需要数据迁移脚本。Mitigation: seed 命令中包含迁移逻辑，根据旧 Role 字符串匹配 Role 表记录，自动填充 RoleID。

**[R2] Casbin 策略与菜单权限耦合** → 菜单变更需要同步 Casbin 策略。Mitigation: 菜单管理 API 在增删菜单时自动同步策略（仅限有 permission 标识的菜单）。

**[R3] JWT 中不含完整权限列表** → 前端刷新页面需重新获取权限。Mitigation: authStore.init() 已在 app 启动时执行，可顺便加载菜单和权限。

**[R4] 前端路由仍然静态** → 用户可通过直接输入 URL 访问无权限页面。Mitigation: PermissionGuard 组件在渲染前检查权限，无权限显示 403；后端 CasbinAuth 中间件是最终防线。

**[R5] Casbin 内存策略加载** → 策略量增大可能影响启动速度。Mitigation: 当前系统规模小，策略数量有限（<1000条），内存模式完全足够。
