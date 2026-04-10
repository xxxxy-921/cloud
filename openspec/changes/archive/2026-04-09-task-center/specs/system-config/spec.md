## ADDED Requirements

### Requirement: Scheduler history retention config
The system SHALL support a `scheduler.history_retention_days` config key with default value `30`. This config SHALL be readable by the scheduler engine to determine how many days of task execution history to retain.

#### Scenario: Config seeded on first run
- **WHEN** the application starts for the first time
- **THEN** the `scheduler.history_retention_days` key SHALL exist in system_configs with value `30` and remark `任务执行历史保留天数，0 表示永不清理`

#### Scenario: Admin updates retention
- **WHEN** an admin updates `scheduler.history_retention_days` to `7` via the config API
- **THEN** the next cleanup task execution SHALL delete records older than 7 days
