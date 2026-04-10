## Why

系统缺乏对用户登录行为和管理操作的追踪记录能力。当出现安全事件或需要排查问题时，无法回答"谁在什么时候从哪里做了什么"。审计日志是企业级应用的基础安全能力，需要尽早建立。

## What Changes

- 新增统一的审计日志表（单表设计），通过 `category` 字段区分日志类型（auth / operation / application）
- 新增审计日志只读查询 API：`GET /api/v1/audit-logs?category=...`
- 新增审计日志前端页面，使用 Tab 区分「登录活动」和「操作记录」
- 新增审计中间件，自动捕获 operation 类操作（Handler 声明式 + 中间件收集）
- 在 auth handler 中接入登录/登出/锁定等事件的审计记录
- 新增审计服务层 `AuditService.Log()` 方法，供 service/scheduler 层直接调用（application 类事件的接入点）
- 新增按 category 分开的日志保留策略配置（存储在 system_config）
- 新增定时清理任务，按保留天数自动清理过期审计日志
- 在菜单种子中添加「审计日志」菜单项及 Casbin 权限策略

## Capabilities

### New Capabilities
- `audit-log`: 统一审计日志能力，包含数据模型、捕获机制（auth 钩子 / operation 中间件 / 手动接入）、查询 API、前端页面（Tab 分类）、日志保留策略与定时清理

### Modified Capabilities
- `user-auth`: 登录成功/失败/登出流程中接入审计日志记录
- `settings-page`: 安全设置中新增日志保留天数配置项
- `typed-settings-api`: 新增 audit_log_retention_* 配置键

## Impact

- **后端新增文件**: model/audit_log.go, repository/audit_log.go, service/audit_log.go, handler/audit_log.go, middleware/audit.go
- **后端修改文件**: cmd/server/main.go (IOC 注册), database/database.go (AutoMigrate), handler/auth.go (接入审计), seed/menus.go + policies.go (菜单与权限), scheduler/builtin.go (清理任务), seed/seed.go (默认配置)
- **前端新增**: pages/audit-logs/ (列表页 + 两个 Tab 组件)
- **前端修改**: App.tsx (路由注册)
- **API 新增**: `GET /api/v1/audit-logs`
- **数据库**: 新增 `audit_logs` 表
