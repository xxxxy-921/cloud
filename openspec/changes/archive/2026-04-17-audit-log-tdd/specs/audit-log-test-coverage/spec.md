## ADDED Requirements

### Requirement: Model layer tests
The `AuditLog` model SHALL have a unit test verifying that `ToResponse()` correctly maps all fields.

#### Scenario: ToResponse maps all fields
- **WHEN** `ToResponse()` is called on an `AuditLog` with all fields populated
- **THEN** the returned `AuditLogResponse` SHALL contain identical values for ID, CreatedAt, Category, UserID, Username, Action, Resource, ResourceID, Summary, Level, IPAddress, UserAgent, and Detail

### Requirement: Handler layer tests
The `AuditLogHandler` SHALL have integration-style tests verifying request binding, status codes, pagination, filtering, and JSON responses.

#### Scenario: List audit logs success
- **WHEN** `GET /api/v1/audit-logs?category=operation` is requested
- **THEN** the response SHALL have status 200 and contain `items`, `total`, `page`, and `pageSize`

#### Scenario: List requires category
- **WHEN** `GET /api/v1/audit-logs` is requested without a category query parameter
- **THEN** the response SHALL have status 400 with message "category is required"

#### Scenario: List validates category
- **WHEN** `GET /api/v1/audit-logs?category=invalid` is requested
- **THEN** the response SHALL have status 400 with message "invalid category"

#### Scenario: List supports keyword filter
- **WHEN** `GET /api/v1/audit-logs?category=operation&keyword=alice` is requested and matching logs exist
- **THEN** the response SHALL contain only logs whose summary matches the keyword

#### Scenario: List supports action filter
- **WHEN** `GET /api/v1/audit-logs?category=operation&action=create_user` is requested
- **THEN** the response SHALL contain only logs with that action

#### Scenario: List supports resource filter
- **WHEN** `GET /api/v1/audit-logs?category=operation&resource=user` is requested
- **THEN** the response SHALL contain only logs with that resource

#### Scenario: List supports date range
- **WHEN** `GET /api/v1/audit-logs?category=operation&dateFrom=2024-01-01&dateTo=2024-01-31` is requested
- **THEN** the response SHALL contain only logs within the parsed date range

#### Scenario: List response items use AuditLogResponse
- **WHEN** `GET /api/v1/audit-logs?category=operation` returns items
- **THEN** each item SHALL be serialized as `AuditLogResponse`

### Requirement: Middleware tests
The `Audit` middleware SHALL have tests for edge cases not yet covered.

#### Scenario: Records ClientIP and User-Agent
- **WHEN** a request with `X-Forwarded-For` and `User-Agent` headers triggers a successful audited handler
- **THEN** the persisted audit log SHALL contain the captured IP address and user agent string

#### Scenario: Handles non-string audit_action
- **WHEN** a handler sets `audit_action` to a non-string value and returns 2xx
- **THEN** the middleware SHALL still write the audit log using the string zero value for action

### Requirement: Scheduler tests
The `SetAuditLogCleanupHandler` function SHALL have a unit test verifying handler wiring.

#### Scenario: Cleanup handler invokes cleaner
- **WHEN** `SetAuditLogCleanupHandler` is called with a task and a cleaner function
- **THEN** the task's Handler SHALL invoke the cleaner and return nil error
