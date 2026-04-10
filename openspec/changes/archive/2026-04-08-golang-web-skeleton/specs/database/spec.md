## ADDED Requirements

### Requirement: Database initialization with GORM
The system SHALL initialize a GORM database connection on startup using the configured driver and DSN.

#### Scenario: Default SQLite initialization
- **WHEN** no `DB_DRIVER` environment variable is set
- **THEN** the system SHALL open a SQLite database at `metis.db` with foreign keys enabled and WAL journal mode

#### Scenario: SQLite with custom path
- **WHEN** `DB_DRIVER=sqlite` and `DB_DSN` is set to a custom path
- **THEN** the system SHALL open SQLite at the specified path with foreign keys and WAL mode

#### Scenario: PostgreSQL initialization
- **WHEN** `DB_DRIVER=postgres` and `DB_DSN` contains a valid PostgreSQL connection string
- **THEN** the system SHALL open a PostgreSQL connection using the provided DSN

#### Scenario: Unsupported driver
- **WHEN** `DB_DRIVER` is set to an unsupported value
- **THEN** the system SHALL return an error indicating the driver is not supported

### Requirement: Pure Go SQLite driver
The system SHALL use the `github.com/glebarez/sqlite` driver (no CGO dependency) for SQLite connections.

#### Scenario: Build without CGO
- **WHEN** the project is built with `CGO_ENABLED=0`
- **THEN** the binary SHALL compile and run successfully with SQLite support

### Requirement: AutoMigrate on startup
The system SHALL run GORM AutoMigrate for all registered models during database initialization.

#### Scenario: First run with empty database
- **WHEN** the application starts with a new empty database
- **THEN** all model tables SHALL be created automatically

#### Scenario: Subsequent run with existing schema
- **WHEN** the application starts with an existing database
- **THEN** AutoMigrate SHALL add any new columns or indexes without data loss
