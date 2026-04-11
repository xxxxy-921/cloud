## Context

node-management 第一轮已实现：节点注册表（Node CRUD + Token 鉴权）、进程定义（ProcessDef CRUD）、节点-进程绑定（NodeProcess）、命令队列（NodeCommand）、sidecar 二进制（进程管理 + 配置下载 + 探针代码）。

当前控制面通信架构：
- 下行（Server→Sidecar）：Sidecar HTTP 长轮询 `GET /commands`，Server 端每秒查 DB，30s 超时
- 上行（Sidecar→Server）：HTTP POST（heartbeat、ack）

存在的问题：
1. DB 轮询压力：N 节点 × 1 query/s = N queries/s，线性增长
2. 命令延迟 ~1s（DB 轮询间隔）
3. 在线状态感知延迟 30s（依赖心跳超时）
4. 探针代码已实现但未集成到 heartbeat 循环
5. `config.update` 命令常量存在但从未创建
6. 配置只渲染 `configFiles[0]`，多配置文件不工作
7. command-cleanup cron 写成每 5 小时而非 5 分钟
8. `Bind()` 中 command 创建错误被 `_ =` 吞掉
9. 节点删除时不下发 `process.stop`
10. OverrideVars 不在 start payload 中
11. 进程 stdout/stderr 直接继承 sidecar，无日志管理

基础设施约束不变：SQLite/PostgreSQL 双引擎，CGO_ENABLED=0，Gin HTTP 框架，无外部消息中间件。

## Goals / Non-Goals

**Goals:**
- 控制面下行通道从 HTTP 长轮询升级为 SSE，消除 DB 轮询，命令实时推送
- 修复所有第一轮 bug 和断路功能
- 接通探针到 heartbeat 循环
- 实现 config.update 推送链路
- 支持多配置文件
- 进程日志捕获、批量 HTTP POST 上报、Server 端存储和查询
- 前端补全：Probe 配置表单、Reload 按钮、日志查看、指令分页

**Non-Goals:**
- WebSocket 双向通信（SSE + HTTP POST 混合已满足需求）
- NATS 等消息中间件引入（当前规模不需要）
- 多 Server 实例 SSE 广播（未来可加 Redis PubSub）
- Sidecar 自动升级
- 实时日志流（日志通过定时批量 POST，非 streaming）

## Decisions

### 1. 控制面下行使用 SSE 替代 HTTP 长轮询

**选择**: Sidecar 建立 `GET /api/v1/nodes/sidecar/stream` SSE 连接，Server 通过内存 channel 实时推送事件。

**替代方案**:
- WebSocket：真双向，但引入 `gorilla/websocket` 依赖，连接管理更复杂，代理兼容性不如 SSE
- NATS：彻底解耦，天然支持多实例，但引入外部组件，对 <100 节点规模 overkill
- 优化现有长轮询：减少 DB 查询频率，但根本问题未解决

**理由**: SSE 是标准 HTTP，Gin 原生 `c.SSEvent()` 支持，零新依赖。单向推送满足控制面需求（命令下发、配置变更通知），上行仍用 HTTP POST。代理/防火墙友好度优于 WebSocket。未来需要多实例时可加 Redis PubSub 而无需改协议。

### 2. NodeHub 内存连接管理器

**选择**: Server 端维护 `NodeHub` 结构：

```
NodeHub
├── mu sync.RWMutex
├── connections map[uint]*NodeConn   // nodeID → SSE channel
│   └── NodeConn
│       ├── eventCh chan SSEEvent    // buffered channel
│       ├── doneCh chan struct{}     // 连接关闭信号
│       └── connectedAt time.Time
└── methods
    ├── Register(nodeID) → *NodeConn
    ├── Unregister(nodeID)
    ├── Send(nodeID, event) → error  // 在线直推
    ├── Broadcast(nodeIDs, event)    // 批量推送
    └── IsOnline(nodeID) → bool
```

命令下发流程变为：
1. Service 创建 NodeCommand 入 DB（持久化）
2. 调用 `NodeHub.Send(nodeID, event)`
3. 节点在线 → 通过 SSE 直推，Sidecar 收到后执行并 ack
4. 节点离线 → Send 返回 error，命令留在 DB 等重连

**理由**: 内存 channel 替代 DB 轮询，延迟从 ~1s → 实时，DB 压力从 N queries/s → 0。DB 命令队列作为持久化 fallback，保证离线节点不丢命令。

### 3. SSE 事件类型设计

```
event: command
data: {"id":123, "type":"process.start", "payload":{...}}

event: config
data: {"process_def_id":456, "process_name":"nginx", "reason":"def_updated"}

event: ping
data: {}
```

- `command`：命令下发（process.start/stop/restart/config.update）
- `config`：配置变更通知（ProcessDef 更新时推送，Sidecar 收到后主动拉取配置）
- `ping`：保活心跳（每 15s），防止代理/负载均衡器超时断开

Sidecar 重连后，SSE handler 首先查询该节点 pending 命令，逐条推送，然后进入实时推送模式。

### 4. 在线状态双重保障

**选择**: SSE 连接状态 + 心跳双重判断。

- SSE 连接断开 → `NodeHub.Unregister()` → 立即更新 DB 状态为 offline
- 心跳仍保留（每 5s POST）→ 上报进程状态 + 探针结果
- 离线检测定时任务保留但降低频率（从 30s → 60s）→ 作为 SSE 断开检测的兜底

**理由**: SSE 断开可能被 TCP 层延迟感知（尤其在 NAT 环境），心跳作为应用层活性证明。两者取最快者标记离线。

### 5. 探针集成方案

**选择**: Sidecar 端新增 `probeLoop` goroutine，独立于 heartbeat 循环运行。

```
Agent.Run()
├── heartbeatLoop()     // 每 5s，上报进程状态 + 最新探针结果
├── sseLoop()           // SSE 连接，接收命令
└── probeLoop()         // 每个进程独立探针 ticker
    └── for each managed process:
        └── go runProbeForProcess(mp) // 按 probe_interval 执行
```

- `process.start` 命令 payload 新增 `probe_type`、`probe_config`、`probe_interval` 字段
- 探针结果存入 `ManagedProcess.LastProbe`
- `heartbeat` POST 上报时包含 `last_probe` 字段（已有，只需填充）
- 探针失败连续 N 次 → 自动重启进程（可配置，默认关闭）

### 6. 多配置文件支持

**选择**: Server `RenderConfig` 接收 `filename` 参数，渲染指定文件。Sidecar `ConfigManager` 遍历所有 configFiles。

当前：
- Server `GET /configs/:name` → 渲染 `configFiles[0]`（硬编码）
- Sidecar 写入 `generate/<process_name>/config`（固定文件名）

改为：
- Server `GET /configs/:name?file=<filename>` → 按 filename 查找并渲染
- Sidecar `SyncConfig()` 遍历 `ProcessDef.ConfigFiles[]`，逐个下载，写入 `generate/<process_name>/<filename>`
- hash 按文件名分别追踪，任一文件变化触发 reload

### 7. 日志捕获与上报

**选择**: Sidecar 端捕获进程 stdout/stderr → 写入本地日志文件 + 内存环形缓冲区 → 定时批量 POST 到 Server。

```
Sidecar 端：
  进程 stdout/stderr → io.MultiWriter
    ├── 本地文件 generate/<name>/logs/stdout.log（轮转，默认 10MB × 3）
    └── 环形缓冲区 Ring(4096 lines)
        └── 每 10s 或缓冲区满 → POST /api/v1/nodes/sidecar/logs

Server 端：
  POST /logs → 写入 node_process_logs 表
  GET /api/v1/nodes/:id/processes/:defId/logs → 分页查询（管理端 API）
```

**日志模型**:
```
NodeProcessLog
├── ID (BaseModel)
├── NodeID
├── ProcessDefID
├── Stream (stdout | stderr)
├── Content (TEXT, 单次上报的一批日志行)
├── Timestamp (日志产生时间)
└── CreatedAt
```

**考量**: 日志量可能很大，DB 存储有上限。增加定时清理任务（默认保留 7 天），同时提供 `maxLogRetentionDays` 配置。

### 8. 前端 Probe 配置表单

ProcessDef Sheet 中，`probeType` 选择后动态展开对应配置字段：

| probeType | 展开字段 |
|-----------|---------|
| `http` | URL, Expected Status (默认 200), Timeout (默认 5s), Interval (默认 30s) |
| `tcp` | Host:Port, Timeout (默认 5s), Interval (默认 30s) |
| `exec` | Command, Timeout (默认 10s), Interval (默认 30s) |
| `none` | 不展开 |

字段序列化为 `probeConfig` JSON 存入 ProcessDef。

### 9. OverrideVars 传递方案

**选择**: `process.start` 命令 payload 新增 `override_vars` 字段。

当前 `Bind()` 创建 start 命令时只包含 ProcessDef 数据，不含 NodeProcess 的 overrideVars。改为：

```go
payload := map[string]any{
    "process_def_id": np.ProcessDefID,
    "process_def":    def,
    "override_vars":  np.OverrideVars,  // 新增
}
```

Sidecar 收到 start 命令时将 overrideVars 存入 ManagedProcess，`ConfigManager.SyncConfig()` 渲染模板时传入。

### 10. 节点删除清理

**选择**: `NodeService.Delete()` 改为：

1. 查询该节点所有 NodeProcess
2. 对每个进程通过 NodeHub 推送 `process.stop`（如果在线）
3. 批量更新 NodeProcess 状态为 stopped
4. 软删除 Node 记录
5. 清理该节点的 pending NodeCommand 记录

## Risks / Trade-offs

**[SSE 连接持有 goroutine]** → 每个在线节点占用一个 server goroutine 维持 SSE 连接。100 节点 = 100 goroutines，可接受。千级节点需评估。→ 暂不处理，Go goroutine 足够轻量。

**[SSE 与反向代理]** → Nginx 默认会缓冲 SSE 响应，需要配置 `proxy_buffering off` 和 `X-Accel-Buffering: no`。→ 在文档中说明。SSE handler 设置 `X-Accel-Buffering: no` header。

**[日志存储膨胀]** → 100 节点 × 3 进程 × 每 10s 上报 = 2.6M 行/天。→ 日志保留 7 天自动清理 + 批量插入优化 + 可选关闭日志上报。

**[SSE 重连 thundering herd]** → Server 重启时所有 Sidecar 同时重连。→ Sidecar 端重连增加随机 jitter（0-5s）。

**[DB 命令队列与 SSE 双写一致性]** → 命令先入 DB 再推 SSE，如果推 SSE 成功但 DB 写失败会丢命令。→ 先写 DB 再推 SSE，保证持久化优先。SSE 推送失败（离线）无影响，命令已在 DB。

**[Sidecar 端日志文件轮转]** → 需要自实现简单的日志轮转逻辑（按大小）。→ 保持简单：单文件 + 大小阈值 + 最多 N 个备份。
