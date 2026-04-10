## MODIFIED Requirements

### Requirement: samber/do IOC container
The system SHALL use samber/do v2 as the dependency injection container for managing service lifecycle. The container SHALL register auth-related providers: UserRepository, RefreshTokenRepository, AuthService, UserService, AuthHandler, UserHandler. The container SHALL also register the scheduler.Engine provider, which depends on database.DB.

#### Scenario: Service registration and resolution
- **WHEN** the application starts
- **THEN** all services (database, repositories, services, handlers, scheduler engine) INCLUDING auth-related providers SHALL be registered in the IOC container and resolved lazily on first use

### Requirement: Graceful shutdown
The system SHALL shut down gracefully on SIGTERM or SIGINT signals.

#### Scenario: Receive termination signal
- **WHEN** the process receives SIGTERM or SIGINT
- **THEN** the system SHALL stop accepting new requests, stop the scheduler engine (waiting for running tasks to complete), complete in-flight requests, close database connections, and exit cleanly

## ADDED Requirements

### Requirement: Scheduler engine startup
The scheduler engine SHALL be started after all task registrations are complete and before the HTTP server begins accepting requests.

#### Scenario: Engine starts on boot
- **WHEN** the application starts and all TaskDefs have been registered
- **THEN** `engine.Start()` SHALL be called, syncing task states to DB, starting cron dispatcher, and starting the queue poller
