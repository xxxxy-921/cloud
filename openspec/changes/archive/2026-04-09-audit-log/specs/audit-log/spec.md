## ADDED Requirements

### Requirement: AuditLog model
The system SHALL store audit logs in a single `audit_logs` table with the following fields: ID (auto-increment primary key), CreatedAt (timestamp, indexed), Category (enum: auth/operation/application, indexed), UserID (nullable uint, FK to users, indexed), Username (varchar 64, snapshot of operator name), Action (varchar 64, indexed, e.g. "login_success" or "user.create"), Resource (varchar 32, e.g. "session", "user"), ResourceID (varchar 64, nullable), Summary (text, human-readable Chinese description), Level (enum: info/warn/error, default info), IPAddress (varchar 45), UserAgent (varchar 512), Detail (nullable text, JSON string for extension data). The model SHALL NOT embed BaseModel — it has no UpdatedAt or DeletedAt fields. Records are append-only and never soft-deleted.

#### Scenario: Create auth audit log
- **WHEN** a login_success event occurs for user "admin" from IP 10.0.0.1
- **THEN** the system SHALL insert a record with category="auth", action="login_success", username="admin", ip_address="10.0.0.1", user_agent from request header, level="info", and summary="用户 admin 登录成功"

#### Scenario: Create operation audit log
- **WHEN** admin creates a new user "alice"
- **THEN** the system SHALL insert a record with category="operation", action="user.create", resource="user", resource_id=<new user ID>, username="admin", summary="创建用户 alice", level="info"

#### Scenario: UserID nullable on failed login
- **WHEN** a login attempt fails for non-existent username "hacker"
- **THEN** the system SHALL insert a record with user_id=NULL, username="hacker", category="auth", action="login_failed", level="warn"

#### Scenario: Detail JSON stores extension data
- **WHEN** a login_failed event occurs with reason "invalid_password"
- **THEN** the detail field MAY contain `{"reason": "invalid_password"}` as a JSON string

### Requirement: Audit log database indexes
The system SHALL create the following indexes on the audit_logs table: composite index on (category, created_at) for Tab queries, index on (user_id, created_at) for per-user queries, index on action for filtering, and composite index on (resource, resource_id) for resource history queries.

#### Scenario: Tab query performance
- **WHEN** querying audit logs filtered by category="auth" ordered by created_at DESC with pagination
- **THEN** the query SHALL use the (category, created_at) composite index

### Requirement: Audit log query API
The system SHALL provide `GET /api/v1/audit-logs` returning paginated audit logs filtered by category (required), with optional filters: keyword (searches username for auth, searches summary for operation), action, resource, ip_address, date_from, date_to. The endpoint SHALL be protected by `system:audit-log:list` Casbin permission. Response format SHALL follow the standard ListResult pattern with `{code: 0, data: {items: [...], total: N}}`.

#### Scenario: Query auth logs
- **WHEN** GET /api/v1/audit-logs?category=auth&page=1&pageSize=20
- **THEN** the system SHALL return paginated auth audit logs ordered by created_at DESC

#### Scenario: Query operation logs with keyword
- **WHEN** GET /api/v1/audit-logs?category=operation&keyword=alice&page=1&pageSize=20
- **THEN** the system SHALL return operation logs where summary contains "alice"

#### Scenario: Filter by action
- **WHEN** GET /api/v1/audit-logs?category=auth&action=login_failed
- **THEN** the system SHALL return only auth logs with action="login_failed"

#### Scenario: Filter by date range
- **WHEN** GET /api/v1/audit-logs?category=operation&date_from=2026-04-01&date_to=2026-04-09
- **THEN** the system SHALL return operation logs within the specified date range

#### Scenario: Missing category parameter
- **WHEN** GET /api/v1/audit-logs without category parameter
- **THEN** the system SHALL return 400 with validation error

### Requirement: AuditService.Log method
The system SHALL provide an AuditService with a `Log(ctx, entry)` method that inserts an audit log record. The method SHALL execute asynchronously (goroutine) to avoid blocking the caller. Logging failures SHALL be logged to slog but SHALL NOT propagate errors to the caller.

#### Scenario: Async non-blocking write
- **WHEN** AuditService.Log() is called from a handler
- **THEN** the audit record SHALL be written asynchronously without blocking the HTTP response

#### Scenario: Write failure resilience
- **WHEN** the audit log INSERT fails (e.g., database error)
- **THEN** the error SHALL be logged via slog.Error and the caller SHALL NOT receive an error

### Requirement: Audit middleware for operation capture
The system SHALL provide a Gin middleware that runs after handler execution. If the handler has set `audit_action` in the Gin context AND the response status is 2xx, the middleware SHALL call AuditService.Log() with the audit metadata from context keys: audit_action, audit_resource, audit_resource_id (optional), audit_summary. UserID, Username, and IPAddress SHALL be extracted from the JWT context and request headers respectively.

#### Scenario: Successful operation triggers audit
- **WHEN** a handler sets audit_action="user.create" and returns HTTP 200
- **THEN** the audit middleware SHALL create an operation audit log with the declared metadata

#### Scenario: Failed operation skips audit
- **WHEN** a handler sets audit_action="user.create" but returns HTTP 400
- **THEN** the audit middleware SHALL NOT create an audit log

#### Scenario: Handler without audit metadata
- **WHEN** a handler does NOT set audit_action in context (e.g., a GET handler)
- **THEN** the audit middleware SHALL do nothing

### Requirement: Auth event audit logging
The system SHALL record audit logs for the following auth events: login_success (level=info), login_failed (level=warn), logout (level=info). Auth events SHALL be recorded by calling AuditService.Log() directly from the auth handler, with category="auth", appropriate action, username, IP address, and user agent.

#### Scenario: Login success audit
- **WHEN** a user successfully logs in via POST /api/v1/auth/login
- **THEN** the system SHALL record an auth audit log with action="login_success", the user's ID, username, IP, and user agent

#### Scenario: Login failure audit
- **WHEN** a login attempt fails (wrong password or user not found)
- **THEN** the system SHALL record an auth audit log with action="login_failed", level="warn", username from request body, user_id=NULL if user not found, and detail containing the failure reason

#### Scenario: Logout audit
- **WHEN** a user logs out via POST /api/v1/auth/logout
- **THEN** the system SHALL record an auth audit log with action="logout", the user's ID and username

### Requirement: Audit log frontend page
The system SHALL provide a frontend page at /audit-logs displaying audit logs in a tabbed layout with two tabs: "登录活动" (auth) and "操作记录" (operation). Each tab SHALL display a paginated table with category-specific columns and filters. The page SHALL be protected by `system:audit-log:list` permission.

#### Scenario: Auth tab display
- **WHEN** the user views the "登录活动" tab
- **THEN** the table SHALL show columns: 时间, 用户, 事件 (badge with color), IP 地址, 设备 (truncated user agent)
- **AND** filters SHALL include: keyword search (username), action multi-select (login_success/login_failed/logout), date range

#### Scenario: Operation tab display
- **WHEN** the user views the "操作记录" tab
- **THEN** the table SHALL show columns: 时间, 操作者, 操作 (badge with color by verb), 资源类型, 摘要
- **AND** filters SHALL include: keyword search (summary), resource type select, date range

#### Scenario: Event badge colors for auth
- **WHEN** displaying auth events
- **THEN** login_success SHALL be green, login_failed SHALL be red, logout SHALL be gray

#### Scenario: Action badge colors for operation
- **WHEN** displaying operation events
- **THEN** create verbs SHALL be green, update verbs SHALL be blue, delete verbs SHALL be red, other verbs SHALL be gray

### Requirement: Audit log menu and permissions
The system SHALL seed a "审计日志" menu item under the "系统管理" directory with path "/audit-logs", icon "ClipboardList", and permission code "system:audit-log:list". Casbin policies SHALL grant admin role GET access to /api/v1/audit-logs.

#### Scenario: Menu visibility
- **WHEN** a user with admin role views the sidebar
- **THEN** the "审计日志" menu item SHALL appear under "系统管理"

#### Scenario: Permission enforcement
- **WHEN** a user without audit-log:list permission accesses GET /api/v1/audit-logs
- **THEN** the system SHALL return 403 forbidden

### Requirement: Audit log retention cleanup task
The system SHALL register a scheduled task "audit_log_cleanup" with cron expression "0 3 * * *" (daily at 3:00 AM). The task SHALL read retention days for each category from SystemConfig, delete records older than the configured retention period per category, and log the cleanup result as an application-category audit log. A retention value of 0 means never clean that category.

#### Scenario: Daily cleanup execution
- **WHEN** the cleanup task runs at 3:00 AM
- **THEN** the system SHALL delete auth logs older than audit.retention_days_auth, operation logs older than audit.retention_days_operation, and application logs older than audit.retention_days_application

#### Scenario: Zero retention means no cleanup
- **WHEN** audit.retention_days_auth is set to 0
- **THEN** the cleanup task SHALL skip auth log deletion

#### Scenario: Cleanup self-audit
- **WHEN** the cleanup task completes
- **THEN** the system SHALL record an application-category audit log with summary containing the number of deleted records per category
