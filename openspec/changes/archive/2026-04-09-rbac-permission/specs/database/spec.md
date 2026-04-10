## MODIFIED Requirements

### Requirement: AutoMigrate on startup
The system SHALL run GORM AutoMigrate for all registered models during database initialization, including User, RefreshToken, Role, and Menu models. The Casbin GORM adapter SHALL auto-create its `casbin_rule` table separately.

#### Scenario: First run with empty database
- **WHEN** the application starts with a new empty database
- **THEN** all model tables INCLUDING users, refresh_tokens, roles, and menus SHALL be created automatically. The casbin_rule table SHALL be created by the GORM adapter initialization.

#### Scenario: Subsequent run with existing schema
- **WHEN** the application starts with an existing database
- **THEN** AutoMigrate SHALL add any new columns or indexes without data loss
