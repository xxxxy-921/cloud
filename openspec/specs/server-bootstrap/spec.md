### Requirement: samber/do IOC container
The system SHALL use samber/do v2 as the dependency injection container for managing service lifecycle. The container SHALL register auth-related providers: UserRepository, RefreshTokenRepository, AuthService, UserService, AuthHandler, UserHandler. Additionally, the container SHALL register RBAC-related providers: Casbin enforcer, RoleRepository, MenuRepository, RoleService, MenuService, CasbinService, RoleHandler, MenuHandler. The container SHALL also register the scheduler.Engine provider, which depends on database.DB. After kernel initialization, the system SHALL iterate all registered Apps (via `app.All()`) and call each App's `Providers()` method to register App-specific providers into the same IOC container.

#### Scenario: Service registration and resolution
- **WHEN** the application starts
- **THEN** all kernel services (database, repositories, services, handlers, scheduler engine) SHALL be registered first, followed by each registered App's providers, all resolved lazily on first use

#### Scenario: App provider registration
- **WHEN** optional Apps are registered in the global registry
- **THEN** main.go SHALL call `a.Providers(injector)` for each App, allowing App services to reference kernel services via `do.MustInvoke`

### Requirement: Graceful shutdown
The system SHALL shut down gracefully on SIGTERM or SIGINT signals.

#### Scenario: Receive termination signal
- **WHEN** the process receives SIGTERM or SIGINT
- **THEN** the system SHALL stop accepting new requests, stop the scheduler engine (waiting for running tasks to complete), complete in-flight requests, close database connections, and exit cleanly

### Requirement: Gin engine with standard middleware
The system SHALL initialize a Gin engine with slog-based request logging, panic recovery middleware, and route groups for public/authenticated endpoints. The authenticated group SHALL use CasbinAuth middleware for permission checking instead of RequireRole middleware. After kernel routes are registered, the system SHALL iterate all registered Apps and call each App's `Routes()` method to register App-specific routes under the authenticated API group.

#### Scenario: Request logging
- **WHEN** any HTTP request is processed
- **THEN** the middleware SHALL log method, path, status code, and latency using slog

#### Scenario: Panic recovery
- **WHEN** a handler panics during request processing
- **THEN** the middleware SHALL recover, log the error, and return a 500 response

#### Scenario: Route group organization
- **WHEN** routes are registered
- **THEN** the system SHALL organize routes into two groups: public (login/refresh/public-site-info) and authenticated (JWTAuth + CasbinAuth middleware). The CasbinAuth middleware SHALL replace the previous RequireRole("admin") middleware for fine-grained permission control.

#### Scenario: App route registration
- **WHEN** optional Apps are registered
- **THEN** main.go SHALL call `a.Routes(apiGroup)` for each App, passing the authenticated API route group

### Requirement: Server port configuration
The system SHALL listen on port 8080 by default, configurable via `SERVER_PORT` environment variable.

#### Scenario: Default port
- **WHEN** no `SERVER_PORT` environment variable is set
- **THEN** the server SHALL listen on port 8080

#### Scenario: Custom port
- **WHEN** `SERVER_PORT=9090` is set
- **THEN** the server SHALL listen on port 9090

### Requirement: Makefile build commands
The Makefile SHALL provide commands for development and production builds. Production builds SHALL support `EDITION` and `APPS` parameters for module selection.

#### Scenario: Dev mode
- **WHEN** `make dev` is run
- **THEN** the Go server SHALL start with all modules (no build tags needed)

#### Scenario: Production build
- **WHEN** `make build` is run
- **THEN** the system SHALL build the frontend (full modules), then compile the Go binary with embedded assets into a single executable

#### Scenario: Edition build
- **WHEN** `make build EDITION=edition_lite APPS=system` is run
- **THEN** the system SHALL generate a minimal frontend registry, build the frontend, then compile the Go binary with `-tags edition_lite`

### Requirement: Scheduler engine startup
The scheduler engine SHALL be started after all task registrations are complete and before the HTTP server begins accepting requests.

#### Scenario: Engine starts on boot
- **WHEN** the application starts and all TaskDefs have been registered
- **THEN** `engine.Start()` SHALL be called, syncing task states to DB, starting cron dispatcher, and starting the queue poller
