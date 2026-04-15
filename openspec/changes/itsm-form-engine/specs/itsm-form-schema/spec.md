## ADDED Requirements

### Requirement: Form Schema version identifier
Each FormSchema document SHALL include a `version` field with integer value `1` at the top level.

#### Scenario: Valid schema version
- **WHEN** a FormSchema JSON is parsed
- **THEN** it SHALL contain `"version": 1` at the root level

#### Scenario: Missing version
- **WHEN** a FormSchema JSON omits the `version` field
- **THEN** the system SHALL reject it with a validation error

### Requirement: Field type system
The schema SHALL support exactly 15 field types: `text`, `textarea`, `number`, `email`, `url`, `select`, `multi_select`, `radio`, `checkbox`, `switch`, `date`, `datetime`, `date_range`, `user_picker`, `dept_picker`, `rich_text`.

#### Scenario: All field types recognized
- **WHEN** a schema contains fields of each of the 15 types
- **THEN** all fields SHALL be accepted as valid

#### Scenario: Unknown field type rejected
- **WHEN** a schema contains a field with `"type": "unknown_type"`
- **THEN** the system SHALL reject it with a validation error identifying the invalid type

### Requirement: Field structure
Each field in the `fields` array SHALL have the following required properties: `key` (unique non-empty string), `type` (one of 15 valid types), `label` (non-empty string). Optional properties: `placeholder`, `description`, `defaultValue`, `required` (boolean), `disabled` (boolean), `validation` (array), `options` (array), `visibility` (object), `binding` (string), `permissions` (object), `width` ("full"|"half"|"third"), `props` (object).

#### Scenario: Minimum valid field
- **WHEN** a field contains only `key`, `type`, and `label`
- **THEN** it SHALL be accepted as valid with defaults: required=false, disabled=false, width="full", binding=key

#### Scenario: Duplicate field keys rejected
- **WHEN** two fields in the same schema share the same `key` value
- **THEN** the system SHALL reject the schema with an error indicating the duplicate key

### Requirement: Field options
Fields of type `select`, `multi_select`, `radio`, and `checkbox` (when used as a group) SHALL support an `options` array. Each option SHALL have `label` (string) and `value` (string|number|boolean).

#### Scenario: Select with static options
- **WHEN** a `select` field defines `options: [{"label":"Low","value":"low"},{"label":"High","value":"high"}]`
- **THEN** the renderer SHALL display a dropdown with "Low" and "High" choices

### Requirement: Validation rules
Each field MAY include a `validation` array of rules. Each rule SHALL have `rule` (string) and `message` (string). Supported rules: `required`, `minLength`, `maxLength`, `min`, `max`, `pattern`, `email`, `url`. Rules `minLength`/`maxLength`/`min`/`max`/`pattern` SHALL also include a `value` property.

#### Scenario: Pattern validation rule
- **WHEN** a field has validation `[{"rule":"pattern","value":"^[A-Z]+$","message":"仅限大写字母"}]`
- **THEN** the validator SHALL reject values not matching the regex pattern

#### Scenario: Multiple validation rules
- **WHEN** a field has validation `[{"rule":"required","message":"必填"},{"rule":"minLength","value":3,"message":"至少3字符"}]`
- **THEN** the validator SHALL check all rules and return all failing rule messages

### Requirement: Conditional visibility
Each field MAY include a `visibility` object with `conditions` (array) and `logic` ("and"|"or", default "and"). Each condition SHALL have `field` (referencing another field's key), `operator` (equals|not_equals|in|not_in|is_empty|is_not_empty), and `value` (for comparison operators).

#### Scenario: Field visible when condition met
- **WHEN** field A has visibility `{"conditions":[{"field":"category","operator":"equals","value":"incident"}],"logic":"and"}` and the user sets category="incident"
- **THEN** field A SHALL be visible in the form

#### Scenario: Field hidden when condition not met
- **WHEN** field A has the same visibility rule and category="request"
- **THEN** field A SHALL be hidden and its value excluded from submission

### Requirement: Layout sections
The schema MAY include a `layout` object with `columns` (1|2|3, default 1) and `sections` array. Each section SHALL have `title` (string) and `fields` (array of field keys). Optional: `description`, `collapsible` (boolean). If layout is null/absent, all fields SHALL render in a single column in array order.

#### Scenario: Two-column layout with sections
- **WHEN** layout has `columns: 2` and sections defined
- **THEN** fields within each section SHALL render in a 2-column grid, grouped under section headings

#### Scenario: Section references invalid field
- **WHEN** a section's `fields` array contains a key not present in the `fields` array
- **THEN** the backend schema validator SHALL reject the schema

### Requirement: Variable binding
Each field MAY include a `binding` string that maps to a process variable name. If omitted, binding defaults to the field's `key`.

#### Scenario: Explicit binding
- **WHEN** a field has `key: "urgency_level"` and `binding: "urgency"`
- **THEN** the value SHALL be mapped to process variable `urgency` (not `urgency_level`) by downstream systems

### Requirement: Node-level field permissions
Each field MAY include a `permissions` object where keys are workflow node IDs and values are `"editable"`, `"readonly"`, or `"hidden"`.

#### Scenario: Field readonly in approve node
- **WHEN** FormRenderer is rendering with nodeId="node_approve_1" and field has `permissions: {"node_approve_1": "readonly"}`
- **THEN** the field SHALL render as read-only (visible but not editable)

#### Scenario: No permissions entry defaults to editable
- **WHEN** FormRenderer is rendering with a nodeId not present in the field's permissions
- **THEN** the field SHALL default to editable
