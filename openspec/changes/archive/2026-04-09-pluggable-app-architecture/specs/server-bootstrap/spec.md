## MODIFIED Requirements

### Requirement: samber/do IOC container
The system SHALL use samber/do v2 as the dependency injection container for managing service lifecycle. The container SHALL register auth-related providers: UserRepository, RefreshTokenRepository, AuthService, UserService, AuthHandler, UserHandler. After kernel initialization, the system SHALL iterate all registered Apps (via `app.All()`) and call each App's `Providers()` method to register App-specific providers into the same IOC container.

#### Scenario: Service registration and resolution
- **WHEN** the application starts
- **THEN** all kernel services (database, repositories, services, handlers) SHALL be registered first, followed by each registered App's providers, all resolved lazily on first use

#### Scenario: App provider registration
- **WHEN** optional Apps are registered in the global registry
- **THEN** main.go SHALL call `a.Providers(injector)` for each App, allowing App services to reference kernel services via `do.MustInvoke`

### Requirement: Gin engine with standard middleware
The system SHALL initialize a Gin engine with slog-based request logging, panic recovery middleware, and route groups for public/authenticated/admin endpoints. After kernel routes are registered, the system SHALL iterate all registered Apps and call each App's `Routes()` method to register App-specific routes under the authenticated API group.

#### Scenario: Request logging
- **WHEN** any HTTP request is processed
- **THEN** the middleware SHALL log method, path, status code, and latency using slog

#### Scenario: Panic recovery
- **WHEN** a handler panics during request processing
- **THEN** the middleware SHALL recover, log the error, and return a 500 response

#### Scenario: Route group organization
- **WHEN** routes are registered
- **THEN** the system SHALL organize routes into three groups: public (login/refresh), authenticated (JWTAuth middleware), and admin (JWTAuth + RequireRole("admin") middleware)

#### Scenario: App route registration
- **WHEN** optional Apps are registered
- **THEN** main.go SHALL call `a.Routes(apiGroup)` for each App, passing the authenticated API route group

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
