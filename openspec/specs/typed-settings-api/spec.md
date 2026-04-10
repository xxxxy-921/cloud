# Capability: typed-settings-api

## Purpose
Typed settings endpoints providing structured access to system configuration categories (security, scheduler), replacing the generic config CRUD API.

## Requirements

### Requirement: Security settings API
The system SHALL provide `GET /api/v1/settings/security` and `PUT /api/v1/settings/security` with the following fields:

| Field | Type | SystemConfig Key | Default |
|-------|------|-----------------|---------|
| maxConcurrentSessions | int | security.max_concurrent_sessions | 5 |
| sessionTimeoutMinutes | int | security.session_timeout_minutes | 10080 |
| passwordMinLength | int | security.password_min_length | 8 |
| passwordRequireUpper | bool | security.password_require_upper | false |
| passwordRequireLower | bool | security.password_require_lower | false |
| passwordRequireNumber | bool | security.password_require_number | false |
| passwordRequireSpecial | bool | security.password_require_special | false |
| passwordExpiryDays | int | security.password_expiry_days | 0 |
| loginMaxAttempts | int | security.login_max_attempts | 5 |
| loginLockoutMinutes | int | security.login_lockout_minutes | 30 |
| requireTwoFactor | bool | security.require_two_factor | false |
| registrationOpen | bool | security.registration_open | false |
| defaultRoleCode | string | security.default_role_code | "" |
| captchaProvider | string | security.captcha_provider | "none" |
| auditRetentionDaysAuth | int | audit.retention_days_auth | 90 |
| auditRetentionDaysOperation | int | audit.retention_days_operation | 365 |

Both endpoints SHALL be protected by `system:settings:update` (PUT) and `system:settings:list` (GET) permissions.

#### Scenario: Get security settings
- **WHEN** GET /api/v1/settings/security is called
- **THEN** the system SHALL return all 16 fields with values from SystemConfig (or defaults if missing)

#### Scenario: Update security settings
- **WHEN** PUT /api/v1/settings/security with partial or full payload
- **THEN** the system SHALL upsert only the provided SystemConfig keys and return the complete updated settings

#### Scenario: Invalid password min length
- **WHEN** PUT /api/v1/settings/security with passwordMinLength < 1
- **THEN** the system SHALL return 400 (minimum password length must be at least 1)

#### Scenario: Invalid captcha provider
- **WHEN** PUT /api/v1/settings/security with captchaProvider="invalid"
- **THEN** the system SHALL return 400 (valid values: "none", "image")

#### Scenario: Invalid numeric values
- **WHEN** PUT /api/v1/settings/security with any negative number field (except passwordExpiryDays which allows 0)
- **THEN** the system SHALL return 400 with validation error

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
