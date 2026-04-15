## ADDED Requirements

### Requirement: FormDefinition model
The system SHALL store form definitions in `itsm_form_definitions` table with fields: `id` (uint PK), `name` (string, max 128, NOT NULL), `code` (string, max 64, UNIQUE NOT NULL), `description` (string, max 512), `schema` (text, NOT NULL, FormSchema JSON), `version` (int, NOT NULL, default 1), `scope` (string, NOT NULL, default "global"), `service_id` (*uint, INDEX), `is_active` (bool, default true), `created_at`, `updated_at`, `deleted_at`.

#### Scenario: Create form definition
- **WHEN** a valid CreateFormDefinition request is received with name, code, and schema
- **THEN** the system SHALL create a record with version=1 and return the created form definition

#### Scenario: Duplicate code rejected
- **WHEN** a CreateFormDefinition request uses a code that already exists
- **THEN** the system SHALL return HTTP 409 with an error message

### Requirement: Form CRUD API
The system SHALL expose RESTful endpoints at `/api/v1/itsm/forms`:
- `POST /` — create form definition
- `GET /` — list form definitions (supports keyword search, pagination)
- `GET /:id` — get single form definition
- `PUT /:id` — update form definition (bumps version by 1)
- `DELETE /:id` — soft-delete form definition

#### Scenario: List with keyword search
- **WHEN** `GET /api/v1/itsm/forms?keyword=事件` is called
- **THEN** the system SHALL return form definitions whose name or code contains "事件"

#### Scenario: Update bumps version
- **WHEN** a form definition at version 3 is updated via PUT
- **THEN** the system SHALL save the updated schema and set version to 4

#### Scenario: Delete form in use
- **WHEN** a DELETE request targets a form definition that is referenced by a ServiceDefinition.form_id
- **THEN** the system SHALL return HTTP 409 with an error indicating the form is in use

### Requirement: Schema validation on save
The system SHALL validate the FormSchema JSON structure on Create and Update. Validation SHALL check: JSON parseable, version field present and equals 1, all field keys non-empty and unique, all field types in the allowed list, layout section field references exist in the fields array.

#### Scenario: Invalid schema rejected on create
- **WHEN** a CreateFormDefinition request contains schema with an invalid field type
- **THEN** the system SHALL return HTTP 400 with validation errors

#### Scenario: Valid schema accepted
- **WHEN** a CreateFormDefinition request contains a structurally valid schema
- **THEN** the system SHALL accept and store it

### Requirement: Form definition audit trail
Create, Update, and Delete operations SHALL set audit fields (`audit_action`, `audit_resource`, `audit_resource_id`, `audit_summary`) on the Gin context for the Audit middleware.

#### Scenario: Create audit
- **WHEN** a form definition "通用事件表单" is created
- **THEN** audit_action SHALL be "itsm.form.create" and audit_summary SHALL contain the form name

### Requirement: IOC registration
FormDefinition repository, service, and handler SHALL be registered in the ITSM App's `Providers()` method via `do.Provide()`, following the existing three-layer pattern.

#### Scenario: Handler resolves dependencies
- **WHEN** the ITSM App starts and IOC container resolves FormDefHandler
- **THEN** it SHALL successfully inject FormDefService which depends on FormDefRepository
