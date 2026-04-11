## ADDED Requirements

### Requirement: Sidecar log capture
The Sidecar SHALL capture stdout and stderr of all managed processes, writing output to local log files with size-based rotation and maintaining an in-memory ring buffer for batch upload.

#### Scenario: Process stdout captured to file
- **WHEN** a managed process writes to stdout
- **THEN** Sidecar writes the output to `generate/<process_name>/logs/stdout.log`
- **AND** rotates the file when it exceeds 10MB, keeping at most 3 backup files

#### Scenario: Process stderr captured to file
- **WHEN** a managed process writes to stderr
- **THEN** Sidecar writes the output to `generate/<process_name>/logs/stderr.log`
- **AND** rotates the file when it exceeds 10MB, keeping at most 3 backup files

### Requirement: Log batch upload
The Sidecar SHALL periodically upload captured log lines to the Server via HTTP POST. Uploads SHALL be batched to minimize request frequency.

#### Scenario: Periodic log upload
- **WHEN** 10 seconds have elapsed since the last upload and the ring buffer contains new lines
- **THEN** Sidecar sends POST to `/api/v1/nodes/sidecar/logs` with buffered log lines
- **AND** clears the uploaded portion of the ring buffer

#### Scenario: Buffer full triggers early upload
- **WHEN** the ring buffer reaches capacity (4096 lines) before the 10-second interval
- **THEN** Sidecar immediately uploads the buffered lines

#### Scenario: Server unreachable during upload
- **WHEN** log upload POST fails due to network error
- **THEN** Sidecar retains the lines in the buffer and retries on the next interval
- **AND** if the buffer is full, oldest lines are dropped to make room for new output

### Requirement: Server log ingestion endpoint
The Server SHALL provide an endpoint at `POST /api/v1/nodes/sidecar/logs` that accepts batched log lines from Sidecar, authenticated via Node Token.

#### Scenario: Receive log batch
- **WHEN** Sidecar sends a log batch with node_id, process_def_id, stream (stdout|stderr), and content
- **THEN** Server stores the log entries in the `node_process_logs` table

### Requirement: Log query API
The Server SHALL provide a management API endpoint for querying process logs with pagination and filtering.

#### Scenario: Query logs for a specific process on a node
- **WHEN** administrator requests `GET /api/v1/nodes/:id/processes/:defId/logs` with optional stream filter and pagination
- **THEN** Server returns paginated log entries sorted by timestamp descending

### Requirement: Log retention cleanup
The Server SHALL automatically clean up old log entries based on a configurable retention period (default 7 days).

#### Scenario: Automatic log cleanup
- **WHEN** the log cleanup scheduled task runs
- **THEN** all log entries older than the configured retention period are deleted from the database

### Requirement: Frontend log viewer
The admin UI SHALL provide a log viewing interface in the node detail page for each bound process.

#### Scenario: View process logs
- **WHEN** administrator opens the logs tab for a process on a node detail page
- **THEN** system displays recent log entries with stream type indicators (stdout/stderr)
- **AND** supports manual refresh and stream filtering
