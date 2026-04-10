## ADDED Requirements

### Requirement: OTel 总开关
系统 SHALL 通过 `OTEL_ENABLED` 环境变量控制 OpenTelemetry 功能。未设置或值不为 `true` 时，所有 OTel 组件使用 noop provider，零性能开销。

#### Scenario: OTel 禁用（默认）
- **WHEN** `OTEL_ENABLED` 未设置或值为 `false`
- **THEN** 系统 SHALL 使用 OTel 默认的 noop TracerProvider，不创建 exporter，不发送任何 trace 数据，应用正常运行

#### Scenario: OTel 启用
- **WHEN** `OTEL_ENABLED=true`
- **THEN** 系统 SHALL 初始化 TracerProvider、OTLP HTTP exporter 和 W3C TraceContext propagator

### Requirement: OTLP HTTP Trace 导出
当 OTel 启用时，系统 SHALL 通过 OTLP HTTP 协议将 trace 数据导出到指定端点。

#### Scenario: 配置导出端点
- **WHEN** `OTEL_ENABLED=true` 且 `OTEL_EXPORTER_OTLP_ENDPOINT` 已设置
- **THEN** 系统 SHALL 使用该端点作为 OTLP HTTP trace exporter 的目标地址

#### Scenario: 导出端点不可用
- **WHEN** OTLP HTTP 端点不可达
- **THEN** BatchSpanProcessor SHALL 异步重试，不阻塞 HTTP 请求处理

### Requirement: 服务资源标识
系统 SHALL 在 trace 数据中包含 service.name 资源属性，可通过 `OTEL_SERVICE_NAME` 配置。

#### Scenario: 自定义服务名
- **WHEN** `OTEL_SERVICE_NAME=my-metis`
- **THEN** 所有导出的 span SHALL 包含 `service.name=my-metis` 资源属性

#### Scenario: 默认服务名
- **WHEN** `OTEL_SERVICE_NAME` 未设置
- **THEN** 系统 SHALL 使用 `metis` 作为默认 service.name

### Requirement: 采样率配置
系统 SHALL 支持通过 `OTEL_SAMPLE_RATE` 环境变量配置 trace 采样率。

#### Scenario: 全量采样（默认）
- **WHEN** `OTEL_SAMPLE_RATE` 未设置
- **THEN** 系统 SHALL 使用 1.0（100%）采样率

#### Scenario: 部分采样
- **WHEN** `OTEL_SAMPLE_RATE=0.1`
- **THEN** 系统 SHALL 使用 ParentBased(TraceIDRatioBased(0.1)) sampler，约 10% 的根 trace 被采样

### Requirement: HTTP 请求自动追踪
系统 SHALL 通过 otelgin middleware 自动为所有 HTTP 请求创建 span。

#### Scenario: 正常请求追踪
- **WHEN** 一个 HTTP 请求到达 Gin router
- **THEN** otelgin middleware SHALL 自动创建一个 span，包含 http.method、http.route、http.status_code 等属性

### Requirement: DB 查询自动追踪
系统 SHALL 通过 otelgorm plugin 自动为所有 GORM 数据库操作创建 span，且 SQL 参数 SHALL 被脱敏。

#### Scenario: 查询追踪
- **WHEN** 业务代码通过 GORM 执行数据库查询
- **THEN** otelgorm plugin SHALL 创建一个 child span，包含 db.system 和 db.statement（参数替换为 `?`）

#### Scenario: 参数脱敏
- **WHEN** SQL 语句包含用户密码等敏感参数
- **THEN** trace 中的 db.statement SHALL 将所有参数值替换为 `?`，不泄露实际值

### Requirement: slog 日志关联 trace
当 OTel 启用时，系统 SHALL 通过 otelslog bridge 在日志输出中自动注入 trace_id 和 span_id 字段。

#### Scenario: 请求日志包含 trace 信息
- **WHEN** HTTP 请求被处理且 OTel 已启用
- **THEN** logger middleware 的日志输出 SHALL 包含当前请求的 trace_id 和 span_id 字段

#### Scenario: OTel 禁用时日志不变
- **WHEN** `OTEL_ENABLED` 未启用
- **THEN** slog 日志输出 SHALL 与当前行为完全一致，无额外字段

### Requirement: Graceful shutdown 刷新 spans
系统 SHALL 在关闭时正确刷新所有 pending span 数据。

#### Scenario: 收到终止信号
- **WHEN** 系统收到 SIGTERM/SIGINT
- **THEN** TracerProvider.Shutdown() SHALL 被调用，将 BatchSpanProcessor 中的 pending span 刷新到 exporter
