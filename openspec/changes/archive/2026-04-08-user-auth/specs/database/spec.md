## MODIFIED Requirements

### Requirement: AutoMigrate on startup
The system SHALL run GORM AutoMigrate for all registered models during database initialization, including User and RefreshToken models.

#### Scenario: First run with empty database
- **WHEN** the application starts with a new empty database
- **THEN** all model tables INCLUDING users and refresh_tokens SHALL be created automatically

#### Scenario: Subsequent run with existing schema
- **WHEN** the application starts with an existing database
- **THEN** AutoMigrate SHALL add any new columns or indexes without data loss
