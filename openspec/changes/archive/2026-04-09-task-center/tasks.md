## 1. 数据模型与存储层

- [x] 1.1 创建 `internal/model/task.go`：定义 TaskState 和 TaskExecution 结构体
- [x] 1.2 在 `internal/database/database.go` 的 AutoMigrate 中注册 TaskState 和 TaskExecution
- [x] 1.3 创建 `internal/scheduler/types.go`：定义 TaskDef、TaskType、ExecStatus、ExecTrigger、QueueStats、Store 接口、ExecutionFilter 等
- [x] 1.4 创建 `internal/scheduler/gorm_store.go`：实现 GormStore（SaveTaskState、GetTaskState、ListTaskStates、Enqueue、Dequeue、UpdateExecution、ListExecutions、GetExecution、Stats、Close）

## 2. 调度引擎核心

- [x] 2.1 创建 `internal/scheduler/executor.go`：goroutine pool 执行器（Submit、Wait、并发限制 max=5、超时控制、重试逻辑）
- [x] 2.2 创建 `internal/scheduler/engine.go`：Engine 结构体（Registry map、cron dispatcher、queue poller、notify channel、Register、Enqueue、Start、Stop、Shutdown）
- [x] 2.3 创建 `internal/scheduler/builtin.go`：注册内建任务 scheduler-history-cleanup（读取 SystemConfig 配置，删除过期执行记录）

## 3. IOC 集成与生命周期

- [x] 3.1 在 `cmd/server/main.go` 中注册 scheduler.Engine provider（依赖 database.DB）
- [x] 3.2 在 main.go 中注册内建任务、调用 engine.Start()，确保在 Gin 监听前启动
- [x] 3.3 确认 engine.Shutdown() 通过 do.Shutdowner 接口在应用关闭时自动调用

## 4. Seed 数据

- [x] 4.1 在 `internal/seed/menus.go` 追加「任务中心」菜单项及按钮权限（system:task:list、system:task:pause、system:task:resume、system:task:trigger）
- [x] 4.2 在 `internal/seed/policies.go` 追加 Admin 角色的 7 条任务 API Casbin 策略
- [x] 4.3 在 seed 中添加 `scheduler.history_retention_days` 默认配置（值=30）

## 5. API Handler

- [x] 5.1 创建 `internal/handler/task.go`：实现 ListTasks、GetTask、ListExecutions、GetStats、PauseTask、ResumeTask、TriggerTask handler
- [x] 5.2 在 `internal/handler/handler.go` 中注册 `/api/v1/tasks` 路由组（GET /tasks、GET /tasks/stats、GET /tasks/:name、GET /tasks/:name/executions、POST /tasks/:name/pause、POST /tasks/:name/resume、POST /tasks/:name/trigger）

## 6. 前端 — 任务列表页

- [x] 6.1 创建 `web/src/pages/tasks/index.tsx`：统计卡片组件（总任务数、运行中、今日完成、今日失败）
- [x] 6.2 实现任务列表表格（Tab 切换定时/异步、名称、描述、Cron、状态 badge、上次执行、操作按钮）
- [x] 6.3 实现操作按钮：暂停/恢复 toggle、手动触发（带确认弹窗），用 PermissionGuard 控制显隐

## 7. 前端 — 任务详情页

- [x] 7.1 创建 `web/src/pages/tasks/detail.tsx`：任务配置卡片（名称、类型、描述、Cron、超时、重试、状态）
- [x] 7.2 实现执行历史表格（ID、触发方式 badge、状态 badge、耗时、错误信息、时间）+ 分页

## 8. 前端 — 路由与集成

- [x] 8.1 在 `web/src/App.tsx` 中添加 `/tasks` 和 `/tasks/:name` 路由（lazy-loaded、PermissionGuard）
- [x] 8.2 添加 API 请求函数到 `web/src/lib/api.ts`（listTasks、getTask、getTaskExecutions、getTaskStats、pauseTask、resumeTask、triggerTask）

## 9. 依赖与验证

- [x] 9.1 执行 `go get github.com/robfig/cron/v3` 添加依赖
- [x] 9.2 启动应用验证：引擎启动日志、内建清理任务注册、API 端点可访问
- [x] 9.3 前端验证：任务列表页渲染、详情页渲染、暂停/恢复/触发操作正常
