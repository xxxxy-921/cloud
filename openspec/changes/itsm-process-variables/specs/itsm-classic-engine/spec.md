## MODIFIED Requirements

### Requirement: Gateway condition evaluation uses process variables
The gateway condition evaluator SHALL read process variables from the `itsm_process_variables` table instead of parsing form_data JSON from the latest activity. Variables SHALL be accessible via the `var.<key>` prefix in condition fields. For backward compatibility, `form.<key>` SHALL also be populated from process variables during a transition period.

#### Scenario: Gateway evaluates variable-based condition
- **WHEN** a gateway has a condition with field="var.urgency", operator="equals", value="high"
- **AND** the ticket has a process variable key="urgency", value="high"
- **THEN** the condition evaluates to true

#### Scenario: Backward-compatible form prefix
- **WHEN** a gateway has a condition with field="form.urgency", operator="equals", value="high"
- **AND** the ticket has a process variable key="urgency", value="high"
- **THEN** the condition evaluates to true (form.* maps to var.* for compatibility)

#### Scenario: Ticket fields still accessible
- **WHEN** a gateway has a condition with field="ticket.priority_id"
- **THEN** the condition evaluates against the ticket's priority_id field (unchanged behavior)

#### Scenario: Activity outcome still accessible
- **WHEN** a gateway has a condition with field="activity.outcome"
- **THEN** the condition evaluates against the latest completed activity's transition_outcome (unchanged behavior)

#### Scenario: Variable not found
- **WHEN** a gateway condition references var.nonexistent
- **AND** no such process variable exists
- **THEN** the condition evaluates to false (field not found)

#### Scenario: Fallback for tickets without variables
- **WHEN** a pre-existing ticket (created before this change) reaches a gateway
- **AND** no process variables exist for the ticket
- **THEN** the evaluator SHALL fall back to parsing form_data from ticket and latest activity (legacy behavior preserved)
