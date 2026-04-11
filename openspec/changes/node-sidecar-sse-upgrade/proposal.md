## Why

node-management 第一轮实现了完整骨架（节点注册、进程定义、绑定/解绑、命令队列、sidecar 二进制），但控制面通信使用 HTTP 长轮询（每节点每秒 1 次 DB 查询），存在 DB 压力线性增长、命令下发延迟 ~1s、在线状态感知延迟 30s 等问题。同时第一轮存在若干断路功能：探针代码已实现但未接入、`config.update` 命令从未创建、配置只渲染单文件、command-cleanup cron 写错等。

本次改造将控制面下行通道从 HTTP 长轮询升级为 **SSE（Server-Sent Events）**，上行保持 HTTP POST 不变，同时修复所有第一轮遗留的 bug 和断路功能，使 sidecar 达到可用的 supervisor 级别。

## What Changes

### 控制面通信升级

- **新增 SSE 推送通道**：Sidecar 建立 `GET /api/v1/nodes/sidecar/stream` SSE 长连接，Server 实时推送命令和配置变更事件
- **新增 NodeHub**：Server 端内存连接管理器（`map[nodeID]chan`），替代 DB 轮询分发命令
- **移除长轮询端点**：`GET /api/v1/nodes/sidecar/commands` 的 DB 轮询逻辑替换为 SSE
- **在线状态感知**：SSE 连接断开即标记 offline，不再依赖 30s 心跳超时
- **离线命令队列**：节点离线时命令入队 DB，重连后 SSE 首先推送 pending 命令

### Bug 修复与断路功能接通

- **接通探针**：将 `probe.go` 集成到 sidecar heartbeat 循环，ProbeType/ProbeConfig 纳入 start 命令 payload
- **实现 config.update 推送**：ProcessDef 更新时向所有绑定的在线节点推送 `config.update` 事件
- **修复 command-cleanup cron**：从 `0 */5 * * *`（每 5 小时）改为 `*/5 * * * *`（每 5 分钟）
- **修复 Bind() 错误吞没**：`commandRepo.Create()` 错误不再用 `_ =` 忽略
- **多配置文件支持**：Server 端 `RenderConfig` 和 sidecar 端 `ConfigManager` 支持渲染和管理 `configFiles[]` 全部文件
- **OverrideVars 纳入 start payload**：sidecar 启动进程时可直接获取覆盖变量
- **节点删除时下发 process.stop**：删除节点前向 sidecar 推送所有进程的 stop 命令
- **重启计数窗口重置**：进程稳定运行超过阈值后重置 RestartCount

### 日志上报

- **新增日志批量上报端点**：`POST /api/v1/nodes/sidecar/logs`，sidecar 定期批量上传进程 stdout/stderr
- **Sidecar 端日志捕获**：进程 stdout/stderr 写入文件 + 内存环形缓冲区，定期 POST 到 Server
- **Server 端日志存储**：写入 DB 或文件，提供管理端查询 API

### 前端补全

- **Probe 配置表单**：probeType 选择后展开对应配置字段（HTTP URL、TCP 端口、exec 命令）
- **Reload 按钮**：进程操作列新增 Reload，触发热重载而非重启
- **进程日志查看**：节点详情页新增日志 Tab，展示进程 stdout/stderr
- **指令历史分页 + 刷新**
- **ProcessDef 关联节点视图**：进程定义页可查看绑定了哪些节点

## Capabilities

### New Capabilities
- `node-sidecar-sse`: SSE 控制面通信（NodeHub 连接管理、事件推送、离线命令队列、在线状态感知）
- `node-process-logs`: 进程日志捕获、批量上报、存储与查询

### Modified Capabilities
- `sidecar`: 通信协议从 HTTP 长轮询升级为 SSE + HTTP POST 混合；接通探针循环；多配置文件支持；日志捕获
- `node-management`: 修复 bug（cron、错误吞没、删除清理）；config.update 推送；OverrideVars 传递
- `process-def`: 前端 Probe 配置表单；关联节点视图

## Impact

- **后端 `internal/app/node/`**：新增 NodeHub（~150 行）、SSE handler、日志 handler；重构 sidecar_service/handler；修复多处 bug
- **Sidecar `internal/sidecar/`**：通信层从 HTTP polling → SSE client；接通探针循环；日志捕获 + 上报；多配置文件
- **前端 `web/src/apps/node/`**：Probe 配置表单、Reload 按钮、日志 Tab、指令分页、关联节点视图
- **API 变更**：新增 `GET /stream`（SSE）、`POST /logs`；`GET /commands` 长轮询逻辑废弃
- **数据库**：可能新增 `node_process_logs` 表
- **依赖**：无新外部依赖（Gin 原生支持 SSE）
