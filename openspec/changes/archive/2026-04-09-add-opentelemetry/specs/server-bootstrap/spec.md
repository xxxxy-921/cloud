## MODIFIED Requirements

### Requirement: Gin engine with standard middleware
The system SHALL initialize a Gin engine with slog-based request logging, panic recovery middleware, OpenTelemetry tracing middleware (otelgin), and route groups for public/authenticated/admin endpoints.

#### Scenario: Request logging
- **WHEN** any HTTP request is processed
- **THEN** the middleware SHALL log method, path, status code, and latency using slog with request context (slog.InfoContext) to enable trace ID correlation

#### Scenario: Panic recovery
- **WHEN** a handler panics during request processing
- **THEN** the middleware SHALL recover, log the error, and return a 500 response

#### Scenario: Route group organization
- **WHEN** routes are registered
- **THEN** the system SHALL organize routes into three groups: public (login/refresh), authenticated (JWTAuth middleware), and admin (JWTAuth + RequireRole("admin") middleware)

#### Scenario: OpenTelemetry tracing middleware
- **WHEN** Gin engine is initialized
- **THEN** otelgin.Middleware SHALL be registered as a global middleware, automatically creating spans for all HTTP requests

### Requirement: Graceful shutdown
The system SHALL shut down gracefully on SIGTERM or SIGINT signals, including flushing OpenTelemetry trace data.

#### Scenario: Receive termination signal
- **WHEN** the process receives SIGTERM or SIGINT
- **THEN** the system SHALL stop accepting new requests, flush pending OTel spans, complete in-flight requests, close database connections, and exit cleanly
