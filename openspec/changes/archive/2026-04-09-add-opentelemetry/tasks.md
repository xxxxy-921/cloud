## 1. 依赖安装

- [x] 1.1 `go get` 安装 OTel 核心包：`go.opentelemetry.io/otel`、`go.opentelemetry.io/otel/sdk`、`go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp`
- [x] 1.2 `go get` 安装自动插桩包：`go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin`、`github.com/uptrace/opentelemetry-go-extra/otelgorm`
- [x] 1.3 `go get` 安装 slog bridge：`go.opentelemetry.io/contrib/bridges/otelslog`

## 2. Telemetry 包

- [x] 2.1 创建 `internal/telemetry/telemetry.go`，实现 `Init(ctx) (shutdown func(context.Context), error)` 函数：读取 `OTEL_ENABLED`，false 时直接返回 noop shutdown；true 时初始化 resource（service.name 从 `OTEL_SERVICE_NAME` 读取，默认 `metis`）、OTLP HTTP trace exporter（从 `OTEL_EXPORTER_OTLP_ENDPOINT` 读取）、TraceIDRatioBased sampler（从 `OTEL_SAMPLE_RATE` 读取，默认 1.0）、BatchSpanProcessor、TracerProvider，设置 W3C TraceContext propagator
- [x] 2.2 在 `Init` 中，当 `OTEL_ENABLED=true` 时，使用 otelslog 包装当前 slog default handler 并替换为新的 default logger，使日志自动注入 trace_id/span_id

## 3. 集成到启动流程

- [x] 3.1 在 `cmd/server/main.go` 中调用 `telemetry.Init(ctx)`，将返回的 shutdown 函数集成到 graceful shutdown 流程中（在 HTTP server shutdown 之后、IOC shutdown 之前调用）
- [x] 3.2 在 Gin engine 上注册 `otelgin.Middleware("metis")` 作为全局中间件（在 Logger 和 Recovery 之后）
- [x] 3.3 在 `internal/database/database.go` 中，GORM 初始化后注册 `otelgorm.NewPlugin(otelgorm.WithoutQueryVariables())`

## 4. Logger 改造

- [x] 4.1 将 `internal/middleware/logger.go` 中的 `slog.Info(...)` 改为 `slog.InfoContext(c.Request.Context(), ...)`，使日志能关联 trace context

## 5. 环境变量文档

- [x] 5.1 更新 `.env.example`，添加 `OTEL_ENABLED`、`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_SERVICE_NAME`、`OTEL_SAMPLE_RATE` 及注释说明
