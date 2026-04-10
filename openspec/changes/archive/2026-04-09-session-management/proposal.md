## Why

当前 Metis 的认证体系只有 JWT + Refresh Token 轮换，没有会话可见性——管理员无法知道谁在线、从哪里登录、用什么设备，也无法强制踢出用户。同时，`/config` 页面暴露了原始的 KV 编辑器，过于底层且危险，系统配置应该通过类型化的设置界面来管理。

## What Changes

- **新增活跃会话管理页面** (`/sessions`)：列出所有在线会话（用户、IP、设备、最后活跃时间），支持管理员强制踢出
- **新增内存 Token 黑名单**：踢出后立即拦截被踢会话的 access token，无需等 30 分钟过期
- **新增并发会话数限制**：登录时检查活跃会话数，超限自动踢出最不活跃的会话，限制值可在系统设置中配置
- **扩展 refresh_tokens 表**：增加 IP、UserAgent、LastSeenAt、AccessTokenJTI 字段用于会话追踪
- **新增定时清理任务**：黑名单清理（每5分钟）+ 过期 token 清理（每天凌晨），注册到现有 scheduler 引擎
- **改造系统设置页面** (`/settings`)：从单纯的站点品牌页扩展为 Tab 式设置中心（站点信息 / 安全设置 / 任务设置）
- **BREAKING**: **删除 `/config` 页面和 `/api/v1/config` 路由**，用类型化的 `/api/v1/settings/*` 专用 API 替代
- **新增类型化设置 API**：`GET/PUT /api/v1/settings/security`、`GET/PUT /api/v1/settings/scheduler`

## Capabilities

### New Capabilities
- `session-management`: 活跃会话列表、强制踢出、内存黑名单、并发限制、定时清理
- `typed-settings-api`: 类型化系统设置 API（替代通用 KV 接口），安全设置和任务设置的读写端点

### Modified Capabilities
- `user-auth`: RefreshToken 模型扩展（IPAddress, UserAgent, LastSeenAt, AccessTokenJTI），JWT 中间件增加黑名单检查，登录流程增加并发会话限制
- `settings-page`: 从两张卡片改为 Tab 式布局（站点信息 / 安全设置 / 任务设置）
- `config-page`: **删除** — 整个页面和相关路由、菜单、权限移除
- `system-config`: 移除公开 API 路由，模型和表保留作为内部存储，仅通过 typed settings service 访问

## Impact

- **后端**：middleware/jwt.go、service/auth.go、model/refresh_token.go、handler/handler.go 有改动；新增 blacklist、session service、settings handler
- **前端**：删除 pages/config/，新增 pages/sessions/，改造 pages/settings/，路由和菜单配置变更
- **数据库**：refresh_tokens 表 AutoMigrate 自动加列；system_configs 表结构不变
- **种子数据**：菜单树变更（删系统配置、加会话管理）、Casbin 策略变更、新增默认配置项
- **Scheduler**：新注册两个定时任务
