## MODIFIED Requirements

### Requirement: Node registration and lifecycle
The system SHALL allow administrators to create, list, update, and delete nodes. Each node SHALL have a unique name, status (pending | online | offline), labels, and system info. A node's status SHALL transition to "online" upon first successful Sidecar registration and to "offline" when the SSE connection is lost. Heartbeat timeout (60s) SHALL serve as a secondary offline detection mechanism.

#### Scenario: Create a new node
- **WHEN** administrator creates a node with name "prod-node-1"
- **THEN** system creates a node with status "pending" and generates a one-time Node Token in `mtk_<32-byte-hex>` format
- **AND** the Token is displayed once and never retrievable again
- **AND** system stores bcrypt hash of the Token and its 8-character prefix

#### Scenario: List nodes with status
- **WHEN** administrator views the node list
- **THEN** system displays all nodes with name, status, labels, last heartbeat time, and running process count

#### Scenario: Node goes offline via SSE disconnect
- **WHEN** a Sidecar's SSE connection is closed or lost
- **THEN** system immediately marks the node status as "offline"

#### Scenario: Node goes offline via heartbeat timeout
- **WHEN** a node's last heartbeat exceeds 60 seconds ago and the node is still marked as online
- **THEN** system marks the node status as "offline" as a fallback mechanism

#### Scenario: Delete a node with cleanup
- **WHEN** administrator deletes a node
- **THEN** system pushes `process.stop` commands for all bound processes via SSE (if node is online)
- **AND** marks all associated NodeProcess records as stopped
- **AND** cleans up pending NodeCommand records
- **AND** soft-deletes the node record and invalidates its Token
