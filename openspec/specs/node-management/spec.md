# Node Management

## Purpose

Manage distributed nodes that run Sidecar agents, including node creation, token-based authentication, status lifecycle, and installation guidance.

## Requirements

### Requirement: Node registration and lifecycle
The system SHALL allow administrators to create, list, update, and delete nodes. Each node SHALL have a unique name, status (pending | online | offline), labels, and system info. A node's status SHALL transition to "online" upon first successful Sidecar registration and to "offline" when heartbeat is missed for 30 seconds.

#### Scenario: Create a new node
- **WHEN** administrator creates a node with name "prod-node-1"
- **THEN** system creates a node with status "pending" and generates a one-time Node Token in `mtk_<32-byte-hex>` format
- **AND** the Token is displayed once and never retrievable again
- **AND** system stores bcrypt hash of the Token and its 8-character prefix

#### Scenario: List nodes with status
- **WHEN** administrator views the node list
- **THEN** system displays all nodes with name, status, labels, last heartbeat time, and running process count

#### Scenario: Node goes offline
- **WHEN** a node's last heartbeat exceeds 30 seconds ago
- **THEN** system marks the node status as "offline"

#### Scenario: Delete a node
- **WHEN** administrator deletes a node
- **THEN** system soft-deletes the node record and invalidates its Token
- **AND** all associated NodeProcess records are marked as stopped

### Requirement: Node Token authentication
The system SHALL provide a Token-based authentication mechanism for Sidecar communication, independent of the JWT system. Node Tokens SHALL be long-lived machine credentials using `mtk_<32-byte-hex>` format with bcrypt storage.

#### Scenario: Sidecar authenticates with valid Token
- **WHEN** Sidecar sends a request with `Authorization: Bearer mtk_<hex>` header
- **THEN** system validates the Token against stored bcrypt hashes
- **AND** resolves the associated node_id for the request context

#### Scenario: Sidecar authenticates with invalid Token
- **WHEN** Sidecar sends a request with an invalid or revoked Token
- **THEN** system responds with HTTP 401 Unauthorized

#### Scenario: Administrator rotates a Node Token
- **WHEN** administrator triggers Token rotation for a node
- **THEN** system generates a new Token, displays it once, invalidates the old Token
- **AND** existing Sidecar connection fails on next request, requiring reconfiguration with the new Token

### Requirement: Node installation guidance
The system SHALL display an installation guide after node creation, including the one-time Token, download command for the Sidecar binary, sample sidecar.yaml configuration, and startup command.

#### Scenario: View installation guide
- **WHEN** administrator creates a new node
- **THEN** system displays a guide page with: Token (shown once), sidecar binary download URL, sidecar.yaml template with server URL and Token fields, and systemd/manual startup command
