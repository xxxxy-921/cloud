## 1. Bug 修复与断路功能修复

- [x] 1.1 修复 `app.go` 中 `node-command-cleanup` cron 表达式：从 `0 */5 * * *`（每5小时）改为 `*/5 * * * *`（每5分钟）
- [x] 1.2 修复 `node_process_service.go` 中 `Bind()` 的 `commandRepo.Create()` 错误吞没：移除 `_ =`，正确返回错误
- [x] 1.3 修复 `node_service.go` 中 `Delete()`：在软删除前下发 `process.stop` 命令给所有绑定进程，清理 pending NodeCommand 记录
- [x] 1.4 修改 `node_process_service.go` 中 `Start()/Bind()` 的 start 命令 payload：加入 `override_vars`、`probe_type`、`probe_config` 字段
- [x] 1.5 离线检测定时任务超时时间从 30s 调整为 60s（作为 SSE 断开检测的兜底）
- [x] 1.6 验证编译通过 `go build -tags dev ./cmd/server/`

## 2. NodeHub 内存连接管理器

- [x] 2.1 创建 `internal/app/node/node_hub.go`：NodeHub 结构体（`sync.RWMutex` + `map[uint]*NodeConn`），NodeConn 结构体（eventCh chan、doneCh chan、connectedAt）
- [x] 2.2 实现 `Register(nodeID)` / `Unregister(nodeID)` / `Send(nodeID, event)` / `Broadcast(nodeIDs, event)` / `IsOnline(nodeID)` 方法
- [x] 2.3 `Unregister()` 中自动更新 DB 节点状态为 offline
- [x] 2.4 在 `app.go` 的 `Providers()` 中注册 NodeHub 为 IOC 单例
- [x] 2.5 验证编译通过

## 3. SSE 推送端点（Server 端）

- [x] 3.1 在 `sidecar_handler.go` 新增 `Stream()` handler：接受 SSE 连接，注册到 NodeHub，循环读取 eventCh 写入 SSE 响应，连接关闭时 Unregister
- [x] 3.2 设置 SSE 响应 headers：`Content-Type: text/event-stream`、`Cache-Control: no-cache`、`Connection: keep-alive`、`X-Accel-Buffering: no`
- [x] 3.3 SSE 连接建立后立即查询该节点 pending 命令，逐条推送
- [x] 3.4 实现 ping 保活：每 15s 发送 `event: ping`
- [x] 3.5 在 `app.go` 的 sidecar 路由组中注册 `GET /stream` 端点
- [x] 3.6 验证编译通过

## 4. 命令下发改造

- [x] 4.1 重构 `sidecar_service.go`：创建命令后调用 `NodeHub.Send()` 推送，Send 失败（离线）时命令留在 DB 等重连
- [x] 4.2 移除 `sidecar_handler.go` 中旧的 `PollCommands()` handler 的 DB 轮询逻辑（保留端点作为 fallback 或直接移除）
- [x] 4.3 `node_process_service.go` 中 `Start()`、`Stop()`、`Restart()`、`Bind()`、`Unbind()` 统一走 NodeHub 推送
- [x] 4.4 验证编译通过

## 5. config.update 推送链路

- [x] 5.1 在 `process_def_service.go` 的 `Update()` 方法中：查询所有绑定此 ProcessDef 的 NodeProcess，对在线节点通过 NodeHub 推送 `config` 事件，对离线节点入队 `config.update` DB 命令
- [x] 5.2 验证编译通过

## 6. 多配置文件支持

- [x] 6.1 修改 `sidecar_service.go` 中 `RenderConfig()`：接受 `filename` 参数，在 `configFiles[]` 中按 filename 查找对应文件渲染，无 filename 时 fallback 渲染 `configFiles[0]`
- [x] 6.2 修改 `sidecar_handler.go` 中 `DownloadConfig()`：从 query parameter 读取 `file` 参数传给 `RenderConfig()`
- [x] 6.3 验证编译通过

## 7. 探针集成（Server 端）

- [x] 7.1 确认 `model.go` 中 `ProcessDef` 的 `ProbeType`、`ProbeConfig` 字段已正确定义（包含 interval、timeout 等）
- [x] 7.2 确认 `sidecar_service.go` 中 `Heartbeat()` 正确同步 `last_probe` 字段到 NodeProcess 记录
- [x] 7.3 验证编译通过

## 8. Sidecar 端 SSE 客户端

- [ ] 8.1 创建 `internal/sidecar/sse_client.go`：SSE 客户端，解析 `event:` + `data:` 字段，返回结构化事件
- [ ] 8.2 实现重连逻辑：随机 jitter（1-5s）+ 指数退避（max 60s）+ 重连成功后重置退避
- [ ] 8.3 修改 `agent.go`：将 `commandPollLoop()` 替换为 `sseLoop()`，从 SSE 读取事件分发给 ProcessManager / ConfigManager
- [ ] 8.4 处理 `config` 事件：收到后调用 `ConfigManager.SyncConfig()` 触发配置同步
- [ ] 8.5 移除 `client.go` 中 `PollCommands()` 方法（不再需要）
- [ ] 8.6 验证 `go build ./cmd/sidecar/` 编译通过

## 9. Sidecar 端探针集成

- [ ] 9.1 修改 `agent.go`：新增 `probeLoop()` goroutine，为每个已启动进程按 `probe_interval` 执行探针
- [ ] 9.2 修改 `process_manager.go`：`HandleCommand()` 处理 `process.start` 时提取 `probe_type`、`probe_config` 存入 `ManagedProcess`
- [ ] 9.3 修改 `process_manager.go`：`GetStatus()` 返回每个进程的 `last_probe` 结果
- [ ] 9.4 修改 heartbeat 上报：将探针结果包含在进程状态中
- [ ] 9.5 验证 `go build ./cmd/sidecar/` 编译通过

## 10. Sidecar 端多配置文件

- [ ] 10.1 修改 `config_manager.go`：`SyncConfig()` 遍历 `ProcessDef.ConfigFiles[]`，逐个调用 `client.DownloadConfig(name, filename)` 下载
- [ ] 10.2 修改 `config_manager.go`：每个文件写入 `generate/<process_name>/<filename>`，hash 按文件名分别追踪
- [ ] 10.3 修改 `client.go`：`DownloadConfig()` 支持 `file` query parameter
- [ ] 10.4 修改 `process_manager.go`：`HandleCommand()` 处理 `process.start` 时提取 `override_vars` 存入 `ManagedProcess`
- [ ] 10.5 验证 `go build ./cmd/sidecar/` 编译通过

## 11. Sidecar 端重启计数优化

- [ ] 11.1 修改 `process_manager.go` 中 `monitor()`：进程稳定运行超过 5 分钟后重置 `RestartCount` 为 0
- [ ] 11.2 修复 `Reload()` 中的锁逻辑：避免持锁调用 `Restart()` 导致的脆弱解锁模式

## 12. 日志捕获与上报（Sidecar 端）

- [ ] 12.1 创建 `internal/sidecar/log_writer.go`：`LogWriter` 结构体，实现 `io.Writer` 接口，写入本地日志文件 + 内存环形缓冲区
- [ ] 12.2 实现日志文件轮转：单文件超过 10MB 时轮转，保留最多 3 个备份
- [ ] 12.3 修改 `process_manager.go`：进程启动时 `cmd.Stdout` / `cmd.Stderr` 设置为 `LogWriter`（通过 `io.MultiWriter` 同时输出到原始 stdout 和 LogWriter）
- [ ] 12.4 创建 `internal/sidecar/log_uploader.go`：每 10s 或缓冲区满时批量 POST `/api/v1/nodes/sidecar/logs`
- [ ] 12.5 修改 `client.go`：新增 `UploadLogs()` 方法
- [ ] 12.6 修改 `agent.go`：新增 `logUploadLoop()` goroutine
- [ ] 12.7 验证 `go build ./cmd/sidecar/` 编译通过

## 13. 日志存储与查询（Server 端）

- [ ] 13.1 在 `model.go` 新增 `NodeProcessLog` 模型（NodeID、ProcessDefID、Stream、Content、Timestamp），加入 `app.go` 的 `Models()` 返回值
- [ ] 13.2 创建 `internal/app/node/node_process_log_repository.go`：Create（批量插入）、List（分页查询，支持按 nodeID + processDefID + stream 过滤）、DeleteBefore（按时间删除）
- [ ] 13.3 创建 `internal/app/node/node_process_log_service.go`：封装日志写入和查询逻辑
- [ ] 13.4 在 `sidecar_handler.go` 新增 `UploadLogs()` handler：`POST /api/v1/nodes/sidecar/logs`，Node Token 认证
- [ ] 13.5 新增管理端 API：`GET /api/v1/nodes/:id/processes/:defId/logs`，支持分页和 stream 过滤
- [ ] 13.6 在 `app.go` 的 `Tasks()` 中注册日志清理定时任务（默认保留 7 天）
- [ ] 13.7 在 `app.go` 的 `Providers()` 和 `Routes()` 中注册新的 repo、service、handler
- [ ] 13.8 验证编译通过 `go build -tags dev ./cmd/server/`

## 14. 前端：Probe 配置表单

- [ ] 14.1 修改 `process-def-sheet.tsx`：probeType 选择后动态展开对应配置字段（HTTP: url/expectedStatus/timeout/interval，TCP: address/timeout/interval，exec: command/timeout/interval）
- [ ] 14.2 将展开的配置字段序列化为 `probeConfig` JSON 提交
- [ ] 14.3 编辑时从 `probeConfig` 反序列化填充表单
- [ ] 14.4 更新 i18n 翻译文件

## 15. 前端：节点详情页增强

- [ ] 15.1 进程列表操作列新增 Reload 按钮（调用 `POST /api/v1/nodes/:id/processes/:defId/reload`，对应后端下发 config.update 命令）
- [ ] 15.2 指令历史 Tab 添加分页（pageSize=20）和刷新按钮
- [ ] 15.3 新增日志 Tab：展示进程日志（调用 `GET /api/v1/nodes/:id/processes/:defId/logs`），支持 stream 筛选和手动刷新
- [ ] 15.4 更新 i18n 翻译文件

## 16. 前端：进程定义关联节点视图

- [ ] 16.1 新增后端 API：`GET /api/v1/process-defs/:id/nodes`，返回绑定了该进程定义的所有节点及其 NodeProcess 状态
- [ ] 16.2 在 `process-defs/index.tsx` 中为每个进程定义添加"查看节点"操作，展示关联节点列表（可用 Sheet 或 Dialog）
- [ ] 16.3 更新 i18n 翻译文件
