## Context

Metis 是一个 Go 单二进制 Web 应用（Gin + GORM + samber/do IOC），前端嵌入编译。当前无任何后台任务调度能力。参考 NekoAdmin 的 BullMQ 调度系统设计，但需适配 Metis 单进程、无 Redis 的架构约束。

现有技术栈约束：
- 数据库：SQLite（默认）/ PostgreSQL（可选），通过 GORM 抽象
- 依赖注入：samber/do v2，所有组件通过 IOC 容器管理
- 响应格式：统一 `R{code, message, data}`
- 权限：Casbin RBAC + 菜单权限（seed 注册）
- 前端：React 19 + Zustand + TanStack Query + shadcn/ui

## Goals / Non-Goals

**Goals:**
- 提供内建的任务调度引擎（定时任务 + 异步任务），随应用启停
- 提供可插拔的 Store 接口，本次实现 GORM 驱动（SQLite + PG）
- 管理员可通过 UI 统一查看任务状态、执行历史，暂停/恢复/手动触发
- 代码注册任务（Registry 模式），无需 UI 动态创建
- 引擎自带历史清理内建任务

**Non-Goals:**
- 不支持任务间依赖编排（DAG）
- 不实现 Redis Store（仅预留接口）
- 不实现失败通知机制
- 不支持通过 UI 动态创建/删除任务定义
- 不支持分布式多实例调度（单进程足够）

## Decisions

### D1: 调度库选择 — robfig/cron/v3

**选择**: `github.com/robfig/cron/v3`
**理由**: Go 生态最成熟的 cron 调度库，无外部依赖，API 简洁，支持秒级精度和标准 cron 表达式。
**替代方案**:
- `go-co-op/gocron`: 更高层封装但灵活性不足
- 自建 ticker: 不值得重造轮子

### D2: 异步队列方案 — DB-backed + Channel 通知

**选择**: GORM 轮询 + channel 唤醒混合模式
**理由**: 无需引入 Redis，契合单二进制哲学。Enqueue 时通过 channel 通知 poller 立即唤醒，默认 3s 轮询作为兜底。SQLite 的写入性能对管理后台任务量绰绰有余。
**替代方案**:
- `hibiken/asynq` (Redis): 增加部署复杂度
- `riverqueue/river` (PG only): 不支持 SQLite

### D3: 存储抽象 — Store 接口

**选择**: 定义 `Store` 接口，本次用 `GormStore` 实现
**理由**: 为未来 Redis 等驱动预留扩展点，不增加当前实现复杂度。GormStore 自动兼容 SQLite 和 PostgreSQL。

### D4: 并发控制 — 固定 goroutine pool

**选择**: 最大 5 个 worker goroutine
**理由**: 管理后台场景任务量不大，固定上限避免资源失控。通过 buffered channel 实现简单的 semaphore 模式。

### D5: 任务定义方式 — 代码注册 + DB 状态同步

**选择**: 任务在代码中通过 `TaskDef` 结构体定义并调用 `engine.Register()` 注册。引擎启动时将代码定义同步到 DB `task_states` 表（记录运行时状态如 active/paused）。
**理由**: 任务 handler 是 Go 函数，必须在代码中定义。DB 只存储可变的运行时状态（暂停/恢复），不存储任务定义本身。

### D6: Model 位置 — internal/model/task.go

**选择**: TaskState 和 TaskExecution 结构体放在 `internal/model/` 目录
**理由**: 保持与现有 User、Role、Menu 等 model 一致的组织方式，AutoMigrate 统一在 database.go 注册。

### D7: 生命周期管理 — do.Shutdowner

**选择**: Engine 实现 `do.Shutdowner` 接口
**理由**: 与现有 database.DB 的关闭方式一致，由 IOC 容器统一管理优雅关闭。Start() 在 main.go 中显式调用。

## Risks / Trade-offs

- **[SQLite 并发写] → 缓解**: 单 writer goroutine + WAL 模式，管理后台任务量下不构成瓶颈
- **[进程重启丢失运行中任务] → 缓解**: 启动时扫描 status=running 的记录，标记为 stale/failed；cron 任务由 robfig/cron 自动恢复
- **[轮询 vs 事件驱动] → 可接受**: 3s 轮询对管理后台场景延迟可接受，channel 唤醒覆盖大部分实时需求
- **[单进程限制] → 可接受**: Metis 定位为单实例管理后台，不需要分布式调度
