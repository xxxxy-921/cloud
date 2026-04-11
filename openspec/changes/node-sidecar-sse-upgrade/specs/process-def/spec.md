## MODIFIED Requirements

### Requirement: Process definition CRUD
The system SHALL allow administrators to create, list, update, and delete process definitions. Each ProcessDef SHALL describe a manageable process with start/stop/reload commands, environment variables, configuration templates, health probe configuration, restart policy, and resource limits. Updating a ProcessDef SHALL trigger config update notifications to all online nodes running this process.

#### Scenario: Create a process definition
- **WHEN** administrator creates a ProcessDef with name "ai-agent", startCommand, and restart_policy "always"
- **THEN** system stores the definition with all specified fields
- **AND** the definition is available for binding to nodes

#### Scenario: Update a process definition pushes config update
- **WHEN** administrator updates a ProcessDef's configuration or commands
- **THEN** system stores the updated definition
- **AND** pushes `config.update` events via SSE to all online nodes running this process
- **AND** for offline nodes, enqueues `config.update` commands in DB for delivery on reconnection

#### Scenario: Delete a process definition
- **WHEN** administrator deletes a ProcessDef
- **THEN** system soft-deletes the definition
- **AND** pushes "process.stop" commands to all nodes running this process

#### Scenario: View associated nodes for a process definition
- **WHEN** administrator views a process definition's detail or associated nodes
- **THEN** system displays all nodes that have this process bound, with per-node status

### Requirement: Node-process binding
The system SHALL allow administrators to bind process definitions to specific nodes, creating NodeProcess records. Each binding SHALL track the process runtime status, PID, config version hash, override variables, and latest probe result. Binding errors SHALL be surfaced to the administrator.

#### Scenario: Bind a process to a node
- **WHEN** administrator assigns ProcessDef "ai-agent" to node "prod-node-1" with optional override_vars
- **THEN** system creates a NodeProcess record with status "pending_config"
- **AND** enqueues a "process.start" command containing full ProcessDef data, probe configuration, and override_vars
- **AND** if command creation fails, the error SHALL be returned to the caller (not silently ignored)

#### Scenario: Unbind a process from a node
- **WHEN** administrator removes a process binding from a node
- **THEN** system enqueues a "process.stop" command
- **AND** marks the NodeProcess as "stopped" after acknowledgment

#### Scenario: View process status on a node
- **WHEN** administrator views a node's detail page
- **THEN** system displays all bound processes with their status, PID, uptime, config version, last probe result
- **AND** provides Start, Stop, Restart, and Reload action buttons

### Requirement: Command queue
The system SHALL maintain a per-node command queue for asynchronous operations. Commands SHALL have types (process.start | process.stop | process.restart | config.update), payloads, and status tracking (pending | acked | failed). Commands SHALL be delivered via SSE when the node is online, and queued in DB when offline.

#### Scenario: Enqueue a command for online node
- **WHEN** an administrative action requires a node operation and the node has an active SSE connection
- **THEN** system creates a NodeCommand in DB with status "pending"
- **AND** immediately pushes the command via SSE

#### Scenario: Enqueue a command for offline node
- **WHEN** an administrative action requires a node operation and the node is offline
- **THEN** system creates a NodeCommand in DB with status "pending"
- **AND** the command will be delivered when the node reconnects

#### Scenario: Command acknowledgment
- **WHEN** Sidecar acknowledges a command with success
- **THEN** system updates command status to "acked" with timestamp

#### Scenario: Command failure
- **WHEN** Sidecar acknowledges a command with failure
- **THEN** system updates command status to "failed" with error details
- **AND** the failure is visible in the node's command history

#### Scenario: Stale command cleanup
- **WHEN** a command has been in "pending" status for more than 5 minutes
- **THEN** a scheduled task running every 5 minutes marks the command as "failed" with reason "timeout"

### Requirement: Configuration template rendering
The system SHALL render ProcessDef configuration templates using Go template syntax before download by Sidecar. Template variables SHALL include node labels, process override variables, and system config values. The system SHALL support rendering individual files from a multi-file configuration.

#### Scenario: Render specific configuration file
- **WHEN** Sidecar requests configuration via `GET /configs/:name?file=<filename>`
- **THEN** Server finds the matching file in the ProcessDef's configFiles array by filename
- **AND** renders the template with node labels, override_vars, and relevant SystemConfig values
- **AND** returns the rendered content with a SHA256 content hash header

#### Scenario: Render default configuration file (backward compatible)
- **WHEN** Sidecar requests configuration via `GET /configs/:name` without file parameter
- **THEN** Server renders the first file in configFiles array (backward compatible behavior)

#### Scenario: Template rendering error
- **WHEN** a configuration template contains invalid Go template syntax
- **THEN** Server returns HTTP 500 with template error details
- **AND** logs the error for administrator review

### Requirement: Probe configuration UI
The admin UI SHALL provide a dynamic form for configuring health probe parameters based on the selected probe type.

#### Scenario: Configure HTTP probe
- **WHEN** administrator selects probe_type "http" in the ProcessDef form
- **THEN** form expands to show URL, expected status code (default 200), timeout (default 5s), and interval (default 30s) fields

#### Scenario: Configure TCP probe
- **WHEN** administrator selects probe_type "tcp" in the ProcessDef form
- **THEN** form expands to show host:port, timeout (default 5s), and interval (default 30s) fields

#### Scenario: Configure exec probe
- **WHEN** administrator selects probe_type "exec" in the ProcessDef form
- **THEN** form expands to show command, timeout (default 10s), and interval (default 30s) fields

### Requirement: Process reload action
The admin UI SHALL provide a Reload action button for running processes, allowing administrators to trigger hot-reload without full restart.

#### Scenario: Reload a running process
- **WHEN** administrator clicks Reload on a running process
- **THEN** system enqueues a "config.update" command for the process on that node
- **AND** Sidecar re-downloads configuration and executes reload_command or sends reload signal

### Requirement: Command history pagination
The admin UI SHALL display command history with pagination and refresh capability.

#### Scenario: Paginated command history
- **WHEN** administrator views the commands tab on a node detail page
- **THEN** system displays commands with pagination (default 20 per page)
- **AND** provides a refresh button to reload the latest commands
