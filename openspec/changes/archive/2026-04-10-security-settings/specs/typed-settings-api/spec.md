## MODIFIED Requirements

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
