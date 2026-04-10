## 1. 数据模型与基础设施

- [x] 1.1 扩展 RefreshToken 模型：在 `internal/model/refresh_token.go` 添加 IPAddress、UserAgent、LastSeenAt、AccessTokenJTI 字段
- [x] 1.2 实现内存黑名单：创建 `internal/pkg/token/blacklist.go`，包含 Add、IsBlocked、Cleanup、Count 方法（sync.RWMutex + map）
- [x] 1.3 在 IOC 容器中注册 TokenBlacklist 单例（cmd/server/main.go）

## 2. 中间件改造

- [x] 2.1 改造 JWTAuth 中间件：注入 TokenBlacklist 依赖，解析 JWT 后检查 jti 是否在黑名单中，context 中增加 tokenJTI
- [x] 2.2 更新 handler.go 中 JWTAuth 的初始化调用，传入 blacklist 实例

## 3. Auth 服务改造

- [x] 3.1 修改 Login 方法：接收 IP 和 UserAgent 参数
- [x] 3.2 修改 Login 方法：添加并发会话限制逻辑
- [x] 3.3 修改 RefreshTokens 方法：新 RefreshToken 记录写入 IP、UserAgent、LastSeenAt、AccessTokenJTI
- [x] 3.4 修改 ChangePassword 方法：revoke 所有 token 时同时 blacklist 所有 AccessTokenJTI
- [x] 3.5 修改 auth handler 的 Login 路由：从 gin.Context 提取 ClientIP() 和 User-Agent header

## 4. Repository 层扩展

- [x] 4.1 在 RefreshTokenRepo 添加 GetActiveByUserID(userID) 方法：返回该用户所有未撤销、未过期的 refresh token，按 LastSeenAt 升序
- [x] 4.2 在 RefreshTokenRepo 添加 GetActiveSessions() 方法：join user 表，返回所有活跃会话（含 username），支持分页
- [x] 4.3 在 RefreshTokenRepo 添加 GetActiveTokenJTIsByUserID(userID) 方法：返回该用户所有活跃 token 的 AccessTokenJTI 列表

## 5. Session 会话管理服务

- [x] 5.1 创建 `internal/service/session.go`
- [x] 5.2 在 IOC 容器注册 SessionService
- [x] 5.3 创建 `internal/handler/session.go`
- [x] 5.4 在 handler.go 中注册 session 路由（Casbin 保护）

## 6. 类型化设置 API

- [x] 6.1 创建 `internal/service/settings.go`
- [x] 6.2 在 IOC 容器注册 SettingsService
- [x] 6.3 创建 `internal/handler/settings.go`
- [x] 6.4 在 handler.go 中注册 settings 路由（Casbin 保护），删除 /api/v1/config 路由
- [x] 6.5 删除 `internal/handler/system_config.go`

## 7. 种子数据与权限

- [x] 7.1 更新 `internal/seed/menus.go`
- [x] 7.2 更新 `internal/seed/policies.go`
- [x] 7.3 更新 `internal/seed/seed.go`

## 8. 定时任务注册

- [x] 8.1 在 `internal/scheduler/builtin.go` 添加 BlacklistCleanupTask
- [x] 8.2 在 `internal/scheduler/builtin.go` 添加 ExpiredTokenCleanupTask
- [x] 8.3 在 main.go 中注册两个新任务到 scheduler engine

## 9. 前端 — 会话管理页面

- [x] 9.1 创建 `web/src/pages/sessions/index.tsx`：会话列表页面，使用 useListPage hook 调用 GET /api/v1/sessions，表格列：用户名、IP 地址、设备（UA 解析）、登录时间、最后活跃、操作
- [x] 9.2 创建 `web/src/pages/sessions/kick-dialog.tsx`：AlertDialog 踢出确认组件，调用 DELETE /api/v1/sessions/:id
- [x] 9.3 添加 UA 解析工具函数：简单正则提取浏览器名和操作系统
- [x] 9.4 在路由配置中添加 /sessions 路由（lazy-loaded，PermissionGuard 保护）

## 10. 前端 — 设置页面改造

- [x] 10.1 改造 `web/src/pages/settings/index.tsx`：从平铺卡片改为 Tabs 布局（站点信息 / 安全设置 / 任务设置）
- [x] 10.2 创建 `web/src/pages/settings/security-card.tsx`：安全设置 Tab 内容，包含并发会话数输入（number input，min=0），调用 GET/PUT /api/v1/settings/security
- [x] 10.3 创建 `web/src/pages/settings/scheduler-card.tsx`：任务设置 Tab 内容，包含历史保留天数输入（number input，min=0），调用 GET/PUT /api/v1/settings/scheduler

## 11. 前端 — 清理

- [x] 11.1 删除 `web/src/pages/config/` 目录（index.tsx, config-sheet.tsx, delete-dialog.tsx）
- [x] 11.2 从路由配置中移除 /config 路由
- [x] 11.3 更新菜单 store 或导航组件中与 /config 相关的引用（如果有硬编码的话）
