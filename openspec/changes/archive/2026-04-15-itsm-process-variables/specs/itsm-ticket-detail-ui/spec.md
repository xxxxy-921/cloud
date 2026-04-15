## MODIFIED Requirements

### Requirement: Variables panel in ticket detail
The ticket detail page SHALL display a "Variables" panel showing all process variables for the current ticket. The panel SHALL be read-only and display: variable key, deserialized value, value type, source, and last updated time.

#### Scenario: Ticket with variables
- **WHEN** a user views a ticket that has 5 process variables
- **THEN** the variables panel displays a table with 5 rows showing key, value, type, source, updatedAt

#### Scenario: Ticket with no variables
- **WHEN** a user views a ticket that has no process variables
- **THEN** the variables panel displays an empty state message

#### Scenario: Variable value display
- **WHEN** a variable has value_type="json" and value=`["a","b"]`
- **THEN** the panel displays the value as formatted JSON text
