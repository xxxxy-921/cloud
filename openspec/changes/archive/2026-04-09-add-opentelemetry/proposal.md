## Why

Metis 目前只有 slog 文本日志输出到 stderr，无法做请求链路追踪、DB 查询耗时分析和问题定位。需要接入 OpenTelemetry 将 traces 导出到已部署的 Victoria Traces，通过 .env 开关控制，不配置时零影响。

## What Changes

- 新增 `internal/telemetry/` 包，条件初始化 OTel TracerProvider + OTLP HTTP exporter
- 集成 `otelgin` 中间件，自动追踪所有 HTTP 请求
- 集成 `otelgorm` 插件，自动追踪所有 DB 查询（参数脱敏）
- 集成 `otelslog` bridge，在日志中自动注入 trace_id/span_id
- 新增 `.env` 变量：`OTEL_ENABLED`、`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_SERVICE_NAME`、`OTEL_SAMPLE_RATE`
- 不配置时使用 OTel noop provider，零性能开销

## Capabilities

### New Capabilities
- `telemetry`: OpenTelemetry 集成，包括 TracerProvider 初始化、OTLP HTTP 导出、采样率控制、graceful shutdown

### Modified Capabilities
- `server-bootstrap`: 启动流程增加 telemetry 初始化和 shutdown 步骤
- `database`: GORM 实例注册 otelgorm 插件

## Impact

- **新增依赖**: `go.opentelemetry.io/otel`, `otelgin`, `otelgorm`, `otelslog`, `otlptracehttp`
- **修改文件**: `cmd/server/main.go`, `internal/database/database.go`, `internal/middleware/logger.go`, `.env.example`
- **新增文件**: `internal/telemetry/telemetry.go`
- **API**: 无变化
- **数据库**: 无变化
- **前端**: 无变化
