# Capability: system-config

## Purpose
Provides the SystemConfig K/V table for internal system configuration storage. Access is restricted to internal service methods only (SettingsService, SiteInfoHandler, SchedulerTask) — no public API routes.

## Requirements

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

### Requirement: Scheduler history retention config
The system SHALL support a `scheduler.history_retention_days` config key with default value `30`. This config SHALL be readable by the scheduler engine to determine how many days of task execution history to retain.

#### Scenario: Config seeded on first run
- **WHEN** the application starts for the first time
- **THEN** the `scheduler.history_retention_days` key SHALL exist in system_configs with value `30` and remark `任务执行历史保留天数，0 表示永不清理`

#### Scenario: Admin updates retention
- **WHEN** an admin updates `scheduler.history_retention_days` to `7` via the config API
- **THEN** the next cleanup task execution SHALL delete records older than 7 days
