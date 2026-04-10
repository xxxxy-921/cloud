## MODIFIED Requirements

### Requirement: samber/do IOC container
The system SHALL use samber/do v2 as the dependency injection container for managing service lifecycle. The container SHALL register auth-related providers: UserRepository, RefreshTokenRepository, AuthService, UserService, AuthHandler, UserHandler.

#### Scenario: Service registration and resolution
- **WHEN** the application starts
- **THEN** all services (database, repositories, services, handlers) INCLUDING auth-related providers SHALL be registered in the IOC container and resolved lazily on first use

### Requirement: Gin engine with standard middleware
The system SHALL initialize a Gin engine with slog-based request logging, panic recovery middleware, and route groups for public/authenticated/admin endpoints.

#### Scenario: Request logging
- **WHEN** any HTTP request is processed
- **THEN** the middleware SHALL log method, path, status code, and latency using slog

#### Scenario: Panic recovery
- **WHEN** a handler panics during request processing
- **THEN** the middleware SHALL recover, log the error, and return a 500 response

#### Scenario: Route group organization
- **WHEN** routes are registered
- **THEN** the system SHALL organize routes into three groups: public (login/refresh), authenticated (JWTAuth middleware), and admin (JWTAuth + RequireRole("admin") middleware)
