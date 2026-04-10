## REMOVED Requirements

### Requirement: Get config by key
**Reason**: Public API access to arbitrary config keys is a security risk. Config access is now internal-only via SettingsService.
**Migration**: Use typed settings APIs: GET /api/v1/settings/security or GET /api/v1/settings/scheduler.

### Requirement: Set config
**Reason**: Public API for arbitrary key upserts replaced by typed settings endpoints with validation.
**Migration**: Use PUT /api/v1/settings/security or PUT /api/v1/settings/scheduler.

### Requirement: List all configs
**Reason**: Listing all raw config entries is no longer exposed. Each settings category has its own typed GET endpoint.
**Migration**: Use GET /api/v1/settings/security and GET /api/v1/settings/scheduler.

### Requirement: Delete config by key
**Reason**: Arbitrary key deletion is no longer supported. Config keys are lifecycle-managed by the application.
**Migration**: No migration needed.

## MODIFIED Requirements

### Requirement: SystemConfig K/V table
The system SHALL provide a SystemConfig table with Key (primary key), Value, Remark, CreatedAt, and UpdatedAt fields. The table SHALL NOT be exposed via public API routes. Access SHALL be through internal service methods only (SettingsService, SiteInfoHandler, SchedulerTask).

#### Scenario: Table structure
- **WHEN** the database is initialized
- **THEN** the system_configs table SHALL exist with columns: key (varchar 255, PK), value (text), remark (varchar 500), created_at, updated_at

#### Scenario: Internal access only
- **WHEN** any code needs to read or write a system config value
- **THEN** it SHALL use the SysConfigService or SettingsService methods, NOT direct HTTP API calls

#### Scenario: Seed default security config
- **WHEN** the seed command runs
- **THEN** the system SHALL ensure key `security.max_concurrent_sessions` exists with default value "5" and remark "每用户最大并发会话数，0 表示不限制"
