## MODIFIED Requirements

### Requirement: Security settings API
The system SHALL provide `GET /api/v1/settings/security` returning `{maxConcurrentSessions: number, auditRetentionDaysAuth: number, auditRetentionDaysOperation: number}` and `PUT /api/v1/settings/security` accepting the same shape. Both endpoints SHALL be protected by `system:settings:update` permission (PUT) and `system:settings:list` permission (GET).

#### Scenario: Get security settings
- **WHEN** GET /api/v1/settings/security is called
- **THEN** the system SHALL return `{code: 0, data: {maxConcurrentSessions: <value>, auditRetentionDaysAuth: <value>, auditRetentionDaysOperation: <value>}}` where values are read from SystemConfig keys `security.max_concurrent_sessions` (default 5), `audit.retention_days_auth` (default 90), and `audit.retention_days_operation` (default 365)

#### Scenario: Update security settings
- **WHEN** PUT /api/v1/settings/security with `{maxConcurrentSessions: 10, auditRetentionDaysAuth: 180, auditRetentionDaysOperation: 730}`
- **THEN** the system SHALL upsert the corresponding SystemConfig keys and return the updated settings

#### Scenario: Invalid value
- **WHEN** PUT /api/v1/settings/security with any negative number field
- **THEN** the system SHALL return 400 with validation error (value must be >= 0)
