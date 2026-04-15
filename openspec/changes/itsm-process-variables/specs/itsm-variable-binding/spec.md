## ADDED Requirements

### Requirement: Form binding writes variables on activity completion
When an activity completes and its form schema contains fields with a `binding` property, the engine SHALL extract the corresponding values from form_data and write them as process variables. The variable key SHALL be the binding value, the variable value SHALL be the form field value, and value_type SHALL be inferred from the field type.

#### Scenario: Text field with binding
- **WHEN** a Form activity completes with form_data=`{"title":"VPN Request"}` and the form schema has a field with key="title" and binding="title"
- **THEN** a process variable is written: key="title", value="VPN Request", value_type="string", source="form:<activity_id>"

#### Scenario: Number field with binding
- **WHEN** a Form activity completes with form_data=`{"amount":5000}` and the form schema has a field with key="amount", type="number", binding="amount"
- **THEN** a process variable is written: key="amount", value=5000, value_type="number"

#### Scenario: Field without binding
- **WHEN** a Form activity completes and a form field does NOT have a binding property
- **THEN** no process variable is written for that field (data remains only in activity.form_data)

#### Scenario: Select field with binding
- **WHEN** a Form activity completes with form_data=`{"urgency":"high"}` and the form schema has a select field with key="urgency" and binding="urgency"
- **THEN** a process variable is written: key="urgency", value="high", value_type="string"

### Requirement: Form binding writes variables on ticket creation
When a ticket is created via the classic engine and the service's start form has fields with binding, the engine SHALL write the initial form data as process variables using the same mechanism.

#### Scenario: Ticket creation initializes variables
- **WHEN** a ticket is created for a service with a start form containing 3 fields with binding
- **THEN** 3 process variables are created with scope_id="root" and source="form:start"

#### Scenario: Ticket creation with no bindings
- **WHEN** a ticket is created and the start form has no fields with binding
- **THEN** no process variables are created (form_data is still stored on ticket.form_data as before)

### Requirement: Field type to value_type mapping
The system SHALL map form field types to variable value_types as follows: text/textarea/email/url/select/radio/rich_text → "string"; number → "number"; switch/checkbox(no options) → "boolean"; date/datetime → "date"; multi_select/checkbox(with options)/date_range → "json".

#### Scenario: Switch field binding
- **WHEN** a switch field with binding="approved" is submitted with value=true
- **THEN** the variable is written with value_type="boolean"

#### Scenario: Multi-select field binding
- **WHEN** a multi_select field with binding="tags" is submitted with value=["a","b"]
- **THEN** the variable is written with value_type="json"
