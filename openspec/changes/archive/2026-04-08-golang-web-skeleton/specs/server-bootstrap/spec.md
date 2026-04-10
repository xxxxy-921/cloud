## ADDED Requirements

### Requirement: samber/do IOC container
The system SHALL use samber/do v2 as the dependency injection container for managing service lifecycle.

#### Scenario: Service registration and resolution
- **WHEN** the application starts
- **THEN** all services (database, repositories, services, handlers) SHALL be registered in the IOC container and resolved lazily on first use

### Requirement: Graceful shutdown
The system SHALL shut down gracefully on SIGTERM or SIGINT signals.

#### Scenario: Receive termination signal
- **WHEN** the process receives SIGTERM or SIGINT
- **THEN** the system SHALL stop accepting new requests, complete in-flight requests, close database connections, and exit cleanly

### Requirement: Gin engine with standard middleware
The system SHALL initialize a Gin engine with slog-based request logging and panic recovery middleware.

#### Scenario: Request logging
- **WHEN** any HTTP request is processed
- **THEN** the middleware SHALL log method, path, status code, and latency using slog

#### Scenario: Panic recovery
- **WHEN** a handler panics during request processing
- **THEN** the middleware SHALL recover, log the error, and return a 500 response

### Requirement: Server port configuration
The system SHALL listen on port 8080 by default, configurable via `SERVER_PORT` environment variable.

#### Scenario: Default port
- **WHEN** no `SERVER_PORT` environment variable is set
- **THEN** the server SHALL listen on port 8080

#### Scenario: Custom port
- **WHEN** `SERVER_PORT=9090` is set
- **THEN** the server SHALL listen on port 9090

### Requirement: Makefile build commands
The Makefile SHALL provide commands for development and production builds.

#### Scenario: Dev mode
- **WHEN** `make dev` is run
- **THEN** the Go server SHALL start with hot-reload capability (or restart instruction)

#### Scenario: Production build
- **WHEN** `make build` is run
- **THEN** the system SHALL build the frontend, then compile the Go binary with embedded assets into a single executable
