## MODIFIED Requirements

### Requirement: Ticket creation triggers variable initialization
When the classic engine creates a ticket (Start → first node), if the start form has fields with binding properties, the engine SHALL write those form values as process variables before proceeding to the first activity node.

#### Scenario: Start with bound form fields
- **WHEN** a ticket is created for a service whose start form has fields with binding
- **AND** the user submits form data
- **THEN** process variables are initialized from bound fields before the first activity is created

#### Scenario: Start without form
- **WHEN** a ticket is created for a service that has no start form (FormID is empty)
- **THEN** no process variables are created at ticket creation time (variables may still be created when subsequent form nodes complete)
