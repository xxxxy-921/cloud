## Context

Metis 是 Go + Gin + GORM + SQLite 的单体应用。当前可观测性仅有 slog 文本日志输出到 stderr。用户已部署 Victoria Traces 在 `http://127.0.0.1:10428`（OTLP HTTP），需要将分布式追踪数据导出到该端点。

核心约束：不配置 OTel 时，Metis 必须完全正常运行，零额外开销。

## Goals / Non-Goals

**Goals:**
- 通过 `OTEL_ENABLED` 环境变量控制 OTel 开关
- 自动追踪所有 HTTP 请求（otelgin middleware）
- 自动追踪所有 DB 查询（otelgorm plugin），SQL 参数脱敏
- slog 日志自动注入 trace_id/span_id（otelslog bridge）
- 采样率可通过 `OTEL_SAMPLE_RATE` 配置
- Graceful shutdown 正确刷新 pending spans

**Non-Goals:**
- Metrics（MeterProvider）— 后续再做
- 手动埋点（业务代码中不新增 tracer.Start 调用）
- 自定义 span attributes（除自动插桩提供的外）
- 日志导出到 OTel（只做 trace_id 注入，不做 log export）

## Decisions

### D1: 始终注册 middleware/plugin，靠 noop provider 丢弃

**选择**: 始终添加 otelgin middleware 和 otelgorm plugin，不管 OTEL_ENABLED 是否为 true。

**理由**: OTel 默认的 noop TracerProvider 开销为纳秒级（空 struct 操作）。代码更简洁，不需要条件分支。这是 OTel 官方推荐的"instrument once, configure at deploy time"模式。

**替代方案**: 条件注册 — 真正零开销，但增加 if/else 复杂度，收益微乎其微。

### D2: OTLP HTTP (非 gRPC)

**选择**: 使用 `otlptracehttp` exporter。

**理由**: Victoria Traces 端点是 HTTP；Metis 不依赖 gRPC，无需引入 gRPC 栈；HTTP 配置更简单。

### D3: slog 桥接方式 — otelslog handler wrap

**选择**: 使用 `go.opentelemetry.io/contrib/bridges/otelslog` 包装现有 slog handler。在 OTEL_ENABLED=true 时将默认 slog handler 替换为 otelslog 包装后的 handler。

**理由**: 零业务代码改动（只需 logger middleware 中 `slog.Info` → `slog.InfoContext` 一处改动）。自动从 context 中提取 trace_id/span_id 注入到日志字段。

### D4: 新增 `internal/telemetry/` 包封装初始化

**选择**: 创建独立的 telemetry 包，暴露 `Init(ctx) (shutdown func(ctx), error)` 函数。

**理由**: 职责清晰，main.go 只需一次调用。shutdown 函数集成到现有 graceful shutdown 流程。所有 OTel 配置逻辑集中一处。

### D5: 采样率通过自定义 `OTEL_SAMPLE_RATE` 配置

**选择**: 使用自定义环境变量 `OTEL_SAMPLE_RATE`（0.0~1.0），映射到 `sdktrace.TraceIDRatioBased` sampler，外层包裹 `ParentBased`。

**理由**: 比 OTel 标准的 `OTEL_TRACES_SAMPLER` + `OTEL_TRACES_SAMPLER_ARG` 组合更直观。默认 1.0（全采），适合 Metis 这种小流量单体应用。

### D6: otelgorm 使用 WithoutQueryVariables

**选择**: 注册 otelgorm plugin 时启用 `WithoutQueryVariables()`。

**理由**: SQL 参数值中可能包含密码、token 等敏感数据。Trace 中只记录 SQL 模板（参数替换为 `?`），足够定位性能问题。

## Risks / Trade-offs

- **[Victoria Traces 不可用]** → OTLP HTTP exporter 内置重试 + 超时，BatchSpanProcessor 异步运行，不阻塞请求。队列满后丢弃 span，不影响业务。
- **[新增 6 个 Go 依赖]** → OTel 生态包较多但都是轻量级。编译产物体积增加约 2-3MB，可接受。
- **[slog.InfoContext 改动]** → 仅需改 logger middleware 一处，将 `slog.Info(...)` 改为 `slog.InfoContext(c.Request.Context(), ...)`。其他 slog 调用不改也不影响功能，只是没有 trace 关联。
