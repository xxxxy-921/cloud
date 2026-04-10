## ADDED Requirements

### Requirement: Security settings API
The system SHALL provide `GET /api/v1/settings/security` returning `{maxConcurrentSessions: number}` and `PUT /api/v1/settings/security` accepting the same shape. Both endpoints SHALL be protected by `system:settings:update` permission (PUT) and `system:settings:list` permission (GET).

#### Scenario: Get security settings
- **WHEN** GET /api/v1/settings/security is called
- **THEN** the system SHALL return `{code: 0, data: {maxConcurrentSessions: <value>}}` where value is read from SystemConfig key `security.max_concurrent_sessions` (default 5)

#### Scenario: Update security settings
- **WHEN** PUT /api/v1/settings/security with `{maxConcurrentSessions: 10}`
- **THEN** the system SHALL upsert SystemConfig key `security.max_concurrent_sessions` with value "10" and return the updated settings

#### Scenario: Invalid value
- **WHEN** PUT /api/v1/settings/security with `{maxConcurrentSessions: -1}`
- **THEN** the system SHALL return 400 with validation error (value must be >= 0)

### Requirement: Scheduler settings API
The system SHALL provide `GET /api/v1/settings/scheduler` returning `{historyRetentionDays: number}` and `PUT /api/v1/settings/scheduler` accepting the same shape. Both endpoints SHALL be protected by `system:settings:update` permission (PUT) and `system:settings:list` permission (GET).

#### Scenario: Get scheduler settings
- **WHEN** GET /api/v1/settings/scheduler is called
- **THEN** the system SHALL return `{code: 0, data: {historyRetentionDays: <value>}}` where value is read from SystemConfig key `scheduler.history_retention_days` (default 30)

#### Scenario: Update scheduler settings
- **WHEN** PUT /api/v1/settings/scheduler with `{historyRetentionDays: 60}`
- **THEN** the system SHALL upsert SystemConfig key `scheduler.history_retention_days` with value "60" and return the updated settings

#### Scenario: Invalid value
- **WHEN** PUT /api/v1/settings/scheduler with `{historyRetentionDays: -5}`
- **THEN** the system SHALL return 400 with validation error (value must be >= 0)

### Requirement: Settings service internal abstraction
The system SHALL provide a SettingsService that encapsulates reading/writing typed settings from the SystemConfig table. The generic config CRUD API (`/api/v1/config`) SHALL be removed; all config access goes through typed settings APIs or internal service calls.

#### Scenario: Service reads config key
- **WHEN** SettingsService.GetSecuritySettings() is called
- **THEN** the service SHALL read SystemConfig key `security.max_concurrent_sessions`, parse it as int, and return a SecuritySettings struct with the parsed value (or default if key is missing)

#### Scenario: Service writes config key
- **WHEN** SettingsService.UpdateSecuritySettings(req) is called
- **THEN** the service SHALL upsert the SystemConfig key `security.max_concurrent_sessions` with the string representation of the new value
