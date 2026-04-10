## Why

Metis 当前所有 API 完全开放，没有任何认证和用户管理能力。需要引入用户体系，实现登录认证、用户管理和基础角色控制，为后续权限管理（Casbin RBAC）、SSO、审计日志等能力打基础。

## What Changes

- 新增 User 模型，支持用户名/密码登录，bcrypt 哈希存储密码
- 新增 RefreshToken 模型，支持 JWT access token + opaque refresh token 双 token 认证
- 新增认证 API：登录、登出、token 刷新、查看/修改个人信息、修改密码
- 新增用户管理 API (admin)：用户 CRUD、重置密码、激活/停用
- 新增 JWT 认证中间件，保护需认证的路由
- 新增角色检查中间件（admin/user 两级，硬编码枚举，预留 Casbin 扩展口）
- 新增 CLI 子命令 `metis create-admin` 用于创建初始管理员账号
- 前端新增登录页、用户管理页、token 存储与自动刷新机制
- 前端 API 层增加 Bearer token 自动附加和 401 自动刷新逻辑

## Capabilities

### New Capabilities
- `user-auth`: 用户认证体系——User/RefreshToken 模型、JWT 双 token 认证流程、登录登出、密码管理、认证中间件
- `user-management`: 用户管理——admin 用户 CRUD、重置密码、激活停用、CLI 创建初始管理员
- `user-auth-frontend`: 前端认证——登录页、token 存储/刷新、受保护路由、用户管理页面

### Modified Capabilities
- `server-bootstrap`: 新增 User/RefreshToken 相关的 IOC provider 注册、认证中间件挂载、路由分组（公开/受保护/admin）
- `frontend-routing`: 新增登录页路由、用户管理路由、未认证重定向逻辑
- `database`: AutoMigrate 注册 User 和 RefreshToken 模型

## Impact

- **后端新增文件**: model/user.go, model/refresh_token.go, repository/user.go, repository/refresh_token.go, service/auth.go, service/user.go, handler/auth.go, handler/user.go, middleware/jwt.go, middleware/role.go, pkg/token/jwt.go, cmd/server/admin.go (CLI 子命令)
- **后端修改**: cmd/server/main.go (IOC + 路由), internal/database/database.go (AutoMigrate)
- **前端新增**: pages/login/, pages/users/, stores/auth.ts
- **前端修改**: lib/api.ts (token 拦截器), App.tsx (路由守卫), layout 组件 (用户菜单)
- **新依赖**: golang-jwt/jwt/v5, golang.org/x/crypto (bcrypt)
- **预留扩展**: Casbin RBAC (casbin_rule 表)、UserIdentity (SSO)、AuditLog（审计）暂不实现
