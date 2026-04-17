## MODIFIED Requirements

### Requirement: Progress method advances workflow
ClassicEngine.Progress() SHALL read form schema from the workflow node's inline `formSchema` field when creating the next activity. The engine SHALL NOT query FormDefinition table. When writing form bindings from a completed activity, the engine SHALL parse the schema from `activity.FormSchema` (already snapshotted) as before.

#### Scenario: Activity creation reads inline formSchema
- **WHEN** the engine creates an activity for a form/user_task node
- **THEN** it SHALL copy `node.FormSchema` directly into `activity.FormSchema`
- **AND** it SHALL NOT perform any database query to resolve the form

#### Scenario: Form binding write unchanged
- **WHEN** a user completes an activity with form data
- **THEN** the engine SHALL parse bindings from `activity.FormSchema` and write process variables
- **AND** the binding behavior SHALL be identical to the current implementation
