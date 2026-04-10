package telemetry

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

// Init initializes OpenTelemetry when OTEL_ENABLED=true.
// Returns a shutdown function that flushes pending spans.
// When disabled, returns a no-op shutdown and nil error.
func Init(ctx context.Context) (func(context.Context), error) {
	if os.Getenv("OTEL_ENABLED") != "true" {
		return func(context.Context) {}, nil
	}

	// Service name
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "metis"
	}

	// Resource
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)

	// OTLP HTTP exporter
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4318"
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	// Build exporter options: host:port from URL, path appended with /v1/traces
	urlPath := u.Path
	if urlPath == "" || urlPath == "/" {
		urlPath = "/v1/traces"
	} else {
		urlPath = urlPath + "/v1/traces"
	}
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(u.Host),
		otlptracehttp.WithURLPath(urlPath),
		otlptracehttp.WithTimeout(5 * time.Second),
	}
	if u.Scheme == "http" {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	// Create exporter in goroutine — New() calls Start() which may block
	type exporterResult struct {
		exporter sdktrace.SpanExporter
		err      error
	}
	ch := make(chan exporterResult, 1)
	go func() {
		exp, err := otlptracehttp.New(ctx, opts...)
		ch <- exporterResult{exp, err}
	}()

	var exporter sdktrace.SpanExporter
	select {
	case r := <-ch:
		if r.err != nil {
			slog.Warn("opentelemetry exporter failed, tracing disabled", "error", r.err)
			return func(context.Context) {}, nil
		}
		exporter = r.exporter
	case <-time.After(5 * time.Second):
		slog.Warn("opentelemetry exporter timed out, tracing disabled", "endpoint", endpoint)
		return func(context.Context) {}, nil
	}

	// Sampler
	sampler := sdktrace.ParentBased(sdktrace.AlwaysSample())
	if rateStr := os.Getenv("OTEL_SAMPLE_RATE"); rateStr != "" {
		if rate, err := strconv.ParseFloat(rateStr, 64); err == nil && rate >= 0 && rate <= 1 {
			sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(rate))
		}
	}

	// TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(tp)

	// W3C TraceContext propagator
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Wrap slog with trace-aware handler that writes to stderr
	slog.SetDefault(slog.New(&traceHandler{inner: slog.NewTextHandler(os.Stderr, nil)}))

	slog.Info("opentelemetry initialized",
		"service", serviceName,
		"endpoint", endpoint,
		"path", urlPath,
	)

	return func(ctx context.Context) {
		if err := tp.Shutdown(ctx); err != nil {
			slog.Error("otel shutdown error", "error", err)
		}
	}, nil
}

// traceHandler wraps a slog.Handler to inject trace_id and span_id from context.
type traceHandler struct {
	inner slog.Handler
}

func (h *traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *traceHandler) Handle(ctx context.Context, record slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, record)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{inner: h.inner.WithGroup(name)}
}
