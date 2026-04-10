## Why

Metis 作为管理后台缺少统一的任务调度与监控能力。定时任务（日志清理、数据统计等）和异步任务（导出、批量操作等）没有集中管理入口，管理员无法查看任务状态、执行历史，也无法暂停/恢复/手动触发任务。需要一个内建的任务中心，提供类似 Django Celery / BullMQ Dashboard 的管理体验，但无需外部依赖。

## What Changes

- 新增 `internal/scheduler/` 包：内建任务调度引擎，包含任务注册、Cron 调度、异步队列、执行器
- 新增 `Store` 可插拔存储接口，本次实现基于 GORM 的 SQLite/PostgreSQL 驱动，为未来 Redis 等驱动预留
- 新增 `TaskState`、`TaskExecution` 两个数据库模型
- 新增 `/api/v1/tasks/*` API 端点：任务列表、详情、执行历史、暂停/恢复/手动触发、队列统计
- 新增前端「任务中心」页面（`/tasks`）：统计卡片 + 任务列表 + 任务详情/执行历史
- Seed 新增：任务中心菜单项（系统管理下）、Casbin 权限策略
- SystemConfig 新增 `scheduler.history_retention_days` 配置项，引擎内建历史清理定时任务
- 引入 `robfig/cron/v3` 依赖

## Capabilities

### New Capabilities
- `task-scheduler-engine`: 调度引擎核心 — 任务注册、Cron 调度、异步队列、执行器、Store 接口
- `task-management-api`: 任务管理 REST API — CRUD、暂停/恢复/触发、统计、执行历史
- `task-management-ui`: 任务中心前端页面 — 列表、详情、操作按钮、统计卡片

### Modified Capabilities
- `system-config`: 新增 `scheduler.history_retention_days` 配置键
- `server-bootstrap`: IOC 容器注册 scheduler.Engine，启动/关闭生命周期管理
- `nav-modules`: 系统管理目录下新增「任务中心」菜单项及按钮权限

## Impact

- **后端新增包**: `internal/scheduler/`（约 6 个文件）、`internal/handler/task.go`、`internal/model/task.go`
- **数据库**: 新增 `task_states`、`task_executions` 两张表（AutoMigrate）
- **Seed**: `menus.go` 追加菜单、`policies.go` 追加 Casbin 策略
- **前端**: 新增 `web/src/pages/tasks/` 目录（列表页 + 详情页）
- **依赖**: 新增 `github.com/robfig/cron/v3`
- **IOC**: `cmd/server/main.go` 新增 Engine provider 注册 + Start/Shutdown 调用
