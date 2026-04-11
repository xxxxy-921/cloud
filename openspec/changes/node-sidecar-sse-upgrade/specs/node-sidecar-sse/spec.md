## ADDED Requirements

### Requirement: SSE stream endpoint
The Server SHALL provide an SSE endpoint at `GET /api/v1/nodes/sidecar/stream` that accepts Node Token authentication and establishes a persistent Server-Sent Events connection. The SSE connection SHALL be the primary channel for Server-to-Sidecar communication.

#### Scenario: Sidecar establishes SSE connection
- **WHEN** Sidecar sends GET request to `/api/v1/nodes/sidecar/stream` with valid Node Token and `Accept: text/event-stream` header
- **THEN** Server registers the connection in NodeHub, sets response headers (`Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`), and begins streaming events
- **AND** any pending commands for this node are immediately pushed as `command` events

#### Scenario: SSE connection with invalid Token
- **WHEN** Sidecar sends GET request to `/api/v1/nodes/sidecar/stream` with invalid Token
- **THEN** Server responds with HTTP 401 and does not establish SSE connection

### Requirement: NodeHub connection manager
The Server SHALL maintain an in-memory NodeHub that maps node IDs to active SSE connections. NodeHub SHALL support registering connections, unregistering on disconnect, sending events to specific nodes, and broadcasting events to multiple nodes.

#### Scenario: Send command to online node
- **WHEN** a command is created for a node that has an active SSE connection
- **THEN** NodeHub delivers the command as an SSE `command` event with sub-second latency
- **AND** the command remains in DB with status "pending" until acknowledged

#### Scenario: Send command to offline node
- **WHEN** a command is created for a node that has no active SSE connection
- **THEN** command is stored in DB with status "pending"
- **AND** will be delivered when the node reconnects and establishes SSE

#### Scenario: Broadcast config update to multiple nodes
- **WHEN** a ProcessDef is updated and multiple online nodes have this process bound
- **THEN** NodeHub broadcasts a `config` event to all affected nodes simultaneously

### Requirement: SSE event types
The Server SHALL push three event types through the SSE connection: `command` for process lifecycle operations, `config` for configuration change notifications, and `ping` for connection keepalive.

#### Scenario: Command event delivery
- **WHEN** Server pushes a `command` event
- **THEN** the event data SHALL contain `id` (command ID), `type` (process.start|stop|restart|config.update), and `payload` (command-specific data)

#### Scenario: Config change notification
- **WHEN** Server pushes a `config` event
- **THEN** the event data SHALL contain `process_def_id`, `process_name`, and `reason` (e.g., "def_updated")
- **AND** Sidecar SHALL respond by downloading updated configuration files via HTTP

#### Scenario: Ping keepalive
- **WHEN** no events have been sent for 15 seconds
- **THEN** Server SHALL send a `ping` event to prevent proxy/load-balancer timeout

### Requirement: SSE-based online status detection
The Server SHALL detect node online/offline status based on SSE connection state. When an SSE connection is closed or lost, the Server SHALL immediately mark the node as offline.

#### Scenario: Node goes offline via SSE disconnect
- **WHEN** a Sidecar's SSE connection is closed (network failure, process exit, etc.)
- **THEN** Server immediately marks the node status as "offline" via NodeHub.Unregister()
- **AND** the status change is reflected in the admin UI without waiting for heartbeat timeout

#### Scenario: Sidecar reconnects after disconnect
- **WHEN** Sidecar re-establishes SSE connection after a disconnect
- **THEN** Server marks the node as "online"
- **AND** pushes any pending commands accumulated during the offline period

### Requirement: Sidecar SSE client with reconnection
The Sidecar SHALL replace HTTP long-polling with an SSE client for receiving commands. The SSE client SHALL implement automatic reconnection with exponential backoff and random jitter.

#### Scenario: Sidecar receives command via SSE
- **WHEN** Server pushes a `command` event through SSE
- **THEN** Sidecar parses the event, dispatches to ProcessManager or ConfigManager, and acknowledges via HTTP POST

#### Scenario: SSE connection lost
- **WHEN** SSE connection is interrupted
- **THEN** Sidecar waits a random duration (1-5s jitter) before reconnecting
- **AND** uses exponential backoff (max 60s) for repeated failures

#### Scenario: Server restart causes mass reconnection
- **WHEN** Server restarts and all Sidecar SSE connections drop
- **THEN** each Sidecar reconnects with independent random jitter to avoid thundering herd
