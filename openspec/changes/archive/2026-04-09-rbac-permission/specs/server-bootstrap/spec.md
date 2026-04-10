## MODIFIED Requirements

### Requirement: samber/do IOC container
The system SHALL use samber/do v2 as the dependency injection container for managing service lifecycle. The container SHALL register auth-related providers: UserRepository, RefreshTokenRepository, AuthService, UserService, AuthHandler, UserHandler. Additionally, the container SHALL register RBAC-related providers: Casbin enforcer, RoleRepository, MenuRepository, RoleService, MenuService, CasbinService, RoleHandler, MenuHandler.

#### Scenario: Service registration and resolution
- **WHEN** the application starts
- **THEN** all services (database, repositories, services, handlers) INCLUDING auth-related AND RBAC-related providers (Casbin enforcer, Role/Menu repos, Role/Menu/Casbin services, Role/Menu handlers) SHALL be registered in the IOC container and resolved lazily on first use

### Requirement: Gin engine with standard middleware
The system SHALL initialize a Gin engine with slog-based request logging, panic recovery middleware, and route groups for public/authenticated endpoints. The authenticated group SHALL use CasbinAuth middleware for permission checking instead of RequireRole middleware.

#### Scenario: Request logging
- **WHEN** any HTTP request is processed
- **THEN** the middleware SHALL log method, path, status code, and latency using slog

#### Scenario: Panic recovery
- **WHEN** a handler panics during request processing
- **THEN** the middleware SHALL recover, log the error, and return a 500 response

#### Scenario: Route group organization
- **WHEN** routes are registered
- **THEN** the system SHALL organize routes into two groups: public (login/refresh/public-site-info) and authenticated (JWTAuth + CasbinAuth middleware). The CasbinAuth middleware SHALL replace the previous RequireRole("admin") middleware for fine-grained permission control.
