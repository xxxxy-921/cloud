## ADDED Requirements

### Requirement: BaseModel provides common fields
The system SHALL define a BaseModel struct that provides ID, CreatedAt, UpdatedAt, and DeletedAt fields for all business tables.

#### Scenario: New record creation
- **WHEN** a record embedding BaseModel is created via GORM
- **THEN** the ID SHALL be auto-incremented, CreatedAt SHALL be set to current time, and UpdatedAt SHALL be set to current time

#### Scenario: Record update
- **WHEN** a record embedding BaseModel is updated via GORM
- **THEN** UpdatedAt SHALL be automatically updated to current time

#### Scenario: Soft delete
- **WHEN** a record embedding BaseModel is deleted via GORM
- **THEN** DeletedAt SHALL be set (soft delete) and the record SHALL not appear in normal queries

### Requirement: BaseModel JSON serialization
All BaseModel fields SHALL have JSON tags for API response serialization.

#### Scenario: JSON output format
- **WHEN** a model with BaseModel is serialized to JSON
- **THEN** fields SHALL use camelCase names: `id`, `createdAt`, `updatedAt`, `deletedAt`
