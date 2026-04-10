## 1. 数据模型与数据库

- [x] 1.1 创建 `internal/model/audit_log.go` — AuditLog 结构体（含 Category/Level 枚举、ToResponse 方法）
- [x] 1.2 在 `internal/database/database.go` 的 AutoMigrate 中注册 AuditLog 模型
- [x] 1.3 在 `internal/seed/seed.go` 中添加审计日志保留天数的默认 SystemConfig 种子数据（audit.retention_days_auth=90, audit.retention_days_operation=365, audit.retention_days_application=30）

## 2. 后端核心层（Repository → Service）

- [x] 2.1 创建 `internal/repository/audit_log.go` — Create、List（分页+筛选+category）、DeleteBefore（按 category 和时间清理）
- [x] 2.2 创建 `internal/service/audit_log.go` — Log() 异步写入方法、List() 查询方法、Cleanup() 清理方法

## 3. 审计中间件

- [x] 3.1 创建 `internal/middleware/audit.go` — Gin 中间件，c.Next() 后检查 audit_action context key，仅 2xx 时调用 AuditService.Log()

## 4. Handler 与路由

- [x] 4.1 创建 `internal/handler/audit_log.go` — GET /api/v1/audit-logs 端点，解析 category/keyword/action/resource/date_from/date_to 参数
- [x] 4.2 修改 `internal/handler/handler.go` — 注册审计日志路由，挂载审计中间件到写操作路由组
- [x] 4.3 修改 `internal/handler/auth.go` — 在 Login/Logout 方法中调用 AuditService.Log() 记录 auth 类事件

## 5. 现有 Handler 接入审计

- [x] 5.1 修改用户管理 handler — 在 Create/Update/Delete 等写操作中通过 c.Set("audit_*") 声明审计元数据
- [x] 5.2 修改角色管理 handler — 同上
- [x] 5.3 修改其他写操作 handler（菜单、设置、公告、渠道等）— 同上

## 6. 种子数据与权限

- [x] 6.1 修改 `internal/seed/menus.go` — 添加「审计日志」菜单项（系统管理目录下，图标 ClipboardList，路径 /audit-logs，权限 system:audit-log:list）
- [x] 6.2 修改 `internal/seed/policies.go` — 添加 admin 角色对 /api/v1/audit-logs 的 GET 权限
- [x] 6.3 修改 `internal/middleware/casbin.go` — 确认审计日志路由不在白名单中（需要认证+授权）

## 7. 定时清理任务

- [x] 7.1 修改 `internal/scheduler/builtin.go` — 注册 audit_log_cleanup 定时任务（cron: 0 3 * * *），按 category 读取保留天数并清理过期记录

## 8. Settings 扩展

- [x] 8.1 修改 `internal/service/settings.go` — SecuritySettings 结构体新增 AuditRetentionDaysAuth/AuditRetentionDaysOperation 字段，更新 Get/Update 方法
- [x] 8.2 修改 `internal/handler/settings.go` — 安全设置 API 返回和接受新的保留天数字段
- [x] 8.3 修改 `web/src/pages/settings/security-card.tsx` — 安全设置卡片新增「日志保留策略」区域，两个数字输入框

## 9. IOC 注册

- [x] 9.1 修改 `cmd/server/main.go` — 通过 do.Provide() 注册 AuditLogRepository、AuditLogService，在 Handler.New() 中注入

## 10. 前端页面

- [x] 10.1 创建 `web/src/pages/audit-logs/index.tsx` — 审计日志主页面，Tabs 组件包含两个 Tab
- [x] 10.2 创建 `web/src/pages/audit-logs/auth-tab.tsx` — 登录活动 Tab（表格：时间/用户/事件badge/IP/设备，筛选：用户名搜索+事件类型+日期范围）
- [x] 10.3 创建 `web/src/pages/audit-logs/operation-tab.tsx` — 操作记录 Tab（表格：时间/操作者/操作badge/资源类型/摘要，筛选：摘要搜索+资源类型+日期范围）
- [x] 10.4 修改 `web/src/App.tsx` — 添加 /audit-logs 路由（lazy load + PermissionGuard）
