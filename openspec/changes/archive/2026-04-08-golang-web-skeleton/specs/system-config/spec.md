## ADDED Requirements

### Requirement: SystemConfig K/V table
The system SHALL provide a SystemConfig table with Key (primary key), Value, Remark, CreatedAt, and UpdatedAt fields.

#### Scenario: Table structure
- **WHEN** the database is initialized
- **THEN** the system_configs table SHALL exist with columns: key (varchar 255, PK), value (text), remark (varchar 500), created_at, updated_at

### Requirement: Get config by key
The system SHALL expose `GET /api/v1/config/:key` to retrieve a system config entry.

#### Scenario: Key exists
- **WHEN** a GET request is made to `/api/v1/config/app.name` and the key exists
- **THEN** the response SHALL be 200 with JSON body containing key, value, remark, createdAt, updatedAt

#### Scenario: Key not found
- **WHEN** a GET request is made to `/api/v1/config/nonexistent`
- **THEN** the response SHALL be 404 with an error message

### Requirement: Set config
The system SHALL expose `PUT /api/v1/config` to create or update a system config entry.

#### Scenario: Create new config
- **WHEN** a PUT request with `{"key": "app.name", "value": "Metis", "remark": "应用名称"}` is made and the key does not exist
- **THEN** the response SHALL be 200 and the config SHALL be created

#### Scenario: Update existing config
- **WHEN** a PUT request with an existing key and new value is made
- **THEN** the response SHALL be 200, value SHALL be updated, and updatedAt SHALL change

### Requirement: List all configs
The system SHALL expose `GET /api/v1/config` to list all system config entries.

#### Scenario: Configs exist
- **WHEN** a GET request is made to `/api/v1/config` and entries exist
- **THEN** the response SHALL be 200 with a JSON array of all config entries

#### Scenario: No configs
- **WHEN** a GET request is made to `/api/v1/config` and no entries exist
- **THEN** the response SHALL be 200 with an empty JSON array

### Requirement: Delete config by key
The system SHALL expose `DELETE /api/v1/config/:key` to permanently delete a config entry.

#### Scenario: Delete existing key
- **WHEN** a DELETE request is made to `/api/v1/config/app.name` and the key exists
- **THEN** the response SHALL be 200 and the entry SHALL be permanently removed (hard delete)

#### Scenario: Delete nonexistent key
- **WHEN** a DELETE request is made to `/api/v1/config/nonexistent`
- **THEN** the response SHALL be 404
