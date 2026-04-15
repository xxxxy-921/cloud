## MODIFIED Requirements

### Requirement: ServiceDefinition model form reference
ServiceDefinition SHALL reference forms via `form_id` (*uint, FK to itsm_form_definitions) instead of storing inline `form_schema` (JSONField). The `form_schema` column SHALL be removed from the model. ServiceDefinitionResponse SHALL include `formId` (*uint) instead of `formSchema`.

#### Scenario: Create service with form reference
- **WHEN** a CreateServiceDefinition request includes `formId: 5`
- **THEN** the system SHALL store `form_id=5` and return the created service with `formId: 5`

#### Scenario: Create service without form
- **WHEN** a CreateServiceDefinition request omits `formId`
- **THEN** the system SHALL accept the request with `form_id=NULL`

#### Scenario: Referenced form not found
- **WHEN** a CreateServiceDefinition request references a formId that does not exist
- **THEN** the system SHALL return HTTP 400 with an error message

### Requirement: Workflow node form reference
WorkflowDef NodeData SHALL reference forms via `formId` (string, FormDefinition code) instead of inline `formSchema` (json.RawMessage). The `form_schema` field SHALL be removed from NodeData struct. The engine SHALL resolve formId to the full schema at activity creation time and snapshot it into `activity.form_schema`.

#### Scenario: Classic engine resolves form
- **WHEN** the ClassicEngine encounters a form node with `formId: "form_general_incident"`
- **THEN** it SHALL look up the FormDefinition by code, snapshot its schema into the created activity's form_schema field

#### Scenario: Form not found at runtime
- **WHEN** the engine encounters a formId that does not exist in FormDefinition table
- **THEN** it SHALL record a timeline warning and create the activity without form_schema
