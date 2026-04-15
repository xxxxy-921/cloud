## ADDED Requirements

### Requirement: ProcessVariable Model
The system SHALL store process variables in the `itsm_process_variables` table with fields: id, ticket_id, scope_id, key, value (TEXT), value_type, source, created_at, updated_at. The combination (ticket_id, scope_id, key) SHALL be unique.

#### Scenario: Variable creation
- **WHEN** the engine writes a variable with ticket_id=42, scope_id="root", key="urgency", value="high", value_type="string", source="form:1"
- **THEN** a row is inserted into itsm_process_variables with those values

#### Scenario: Variable update (upsert)
- **WHEN** a variable with the same (ticket_id, scope_id, key) already exists and the engine writes a new value
- **THEN** the existing row's value, value_type, source, and updated_at SHALL be updated (not duplicated)

### Requirement: Variable value types
The system SHALL support 5 value_type values: "string", "number", "boolean", "json", "date". The value column SHALL always store a JSON-serialized TEXT representation. Reading a variable SHALL deserialize according to value_type.

#### Scenario: String variable
- **WHEN** a variable is stored with value_type="string" and value=`"hello"`
- **THEN** reading the variable returns the string "hello"

#### Scenario: Number variable
- **WHEN** a variable is stored with value_type="number" and value=`42`
- **THEN** reading the variable returns the number 42

#### Scenario: Boolean variable
- **WHEN** a variable is stored with value_type="boolean" and value=`true`
- **THEN** reading the variable returns the boolean true

### Requirement: Variable scope
The system SHALL support a scope_id field on each variable. For this change, only scope_id="root" (process-level) is used. The scope mechanism SHALL be extensible for future subprocess scopes.

#### Scenario: Root scope variables
- **WHEN** a variable is written without explicit scope
- **THEN** scope_id SHALL default to "root"

### Requirement: Variable Repository CRUD
The system SHALL provide a VariableRepository with: SetVariable (upsert), GetVariable (by ticket+scope+key), ListByTicket (all variables for a ticket), DeleteByTicket (cleanup).

#### Scenario: List variables for a ticket
- **WHEN** a ticket has 3 variables and ListByTicket is called
- **THEN** all 3 variables are returned ordered by key

### Requirement: Variable Query API
The system SHALL expose `GET /api/v1/itsm/tickets/:id/variables` returning all process variables for the given ticket. The response SHALL include key, value (deserialized), valueType, source, and updatedAt.

#### Scenario: Fetch variables for existing ticket
- **WHEN** a GET request is made to `/api/v1/itsm/tickets/42/variables` and ticket 42 has 5 variables
- **THEN** the response contains an array of 5 variable objects with all fields

#### Scenario: Fetch variables for ticket with no variables
- **WHEN** a GET request is made for a ticket with no variables
- **THEN** the response contains an empty array
