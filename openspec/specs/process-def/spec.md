# Process Definition

## Purpose

Define and manage process definitions that describe deployable processes, their configuration templates, and the command queue for asynchronous node operations.

## Requirements

### Requirement: Process definition CRUD
The system SHALL allow administrators to create, list, update, and delete process definitions. Each ProcessDef SHALL describe a manageable process with its binary path, arguments, environment variables, configuration templates, health probe, restart policy, and resource limits.

#### Scenario: Create a process definition
- **WHEN** administrator creates a ProcessDef with name "ai-agent", binary "metis-ai-agent", and restart_policy "always"
- **THEN** system stores the definition with all specified fields
- **AND** the definition is available for binding to nodes

#### Scenario: Update a process definition
- **WHEN** administrator updates a ProcessDef's configuration template
- **THEN** system stores the new template
- **AND** Server enqueues "config.update" commands for all nodes running this process

#### Scenario: Delete a process definition
- **WHEN** administrator deletes a ProcessDef
- **THEN** system soft-deletes the definition
- **AND** enqueues "process.stop" commands for all nodes running this process

### Requirement: Node-process binding
The system SHALL allow administrators to bind process definitions to specific nodes, creating NodeProcess records. Each binding SHALL track the process runtime status, PID, config version hash, and latest probe result.

#### Scenario: Bind a process to a node
- **WHEN** administrator assigns ProcessDef "ai-agent" to node "prod-node-1"
- **THEN** system creates a NodeProcess record with status "pending_config"
- **AND** enqueues a "process.start" command for the node

#### Scenario: Unbind a process from a node
- **WHEN** administrator removes a process binding from a node
- **THEN** system enqueues a "process.stop" command
- **AND** marks the NodeProcess as "stopped" after acknowledgment

#### Scenario: View process status on a node
- **WHEN** administrator views a node's detail page
- **THEN** system displays all bound processes with their status, PID, uptime, config version, and last probe result

### Requirement: Command queue
The system SHALL maintain a per-node command queue for asynchronous operations. Commands SHALL have types (process.start | process.stop | process.restart | config.update), payloads, and status tracking (pending | acked | failed).

#### Scenario: Enqueue a command
- **WHEN** an administrative action requires a node operation (e.g., start process)
- **THEN** system creates a NodeCommand with type, payload, and status "pending"

#### Scenario: Command acknowledgment
- **WHEN** Sidecar acknowledges a command with success
- **THEN** system updates command status to "acked" with timestamp

#### Scenario: Command failure
- **WHEN** Sidecar acknowledges a command with failure
- **THEN** system updates command status to "failed" with error details
- **AND** the failure is visible in the node's command history

#### Scenario: Stale command cleanup
- **WHEN** a command has been in "pending" status for more than 5 minutes and the node is offline
- **THEN** system marks the command as "failed" with reason "node_offline_timeout"

### Requirement: Configuration template rendering
The system SHALL render ProcessDef configuration templates using Go template syntax before download by Sidecar. Template variables SHALL include node labels, process override variables, and system config values.

#### Scenario: Render configuration with node labels
- **WHEN** Sidecar requests configuration for process "telegraf"
- **THEN** Server renders the config template with node's labels, override_vars, and relevant SystemConfig values
- **AND** returns the rendered content with a content hash header

#### Scenario: Template rendering error
- **WHEN** a configuration template contains invalid Go template syntax
- **THEN** Server returns HTTP 500 with template error details
- **AND** logs the error for administrator review
