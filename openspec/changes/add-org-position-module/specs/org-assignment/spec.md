## ADDED Requirements

### Requirement: Assign multiple positions to a single user
The system SHALL allow a user to hold multiple department/position combinations via a `UserPosition` association table.

#### Scenario: Assign a primary and secondary position
- **WHEN** an admin assigns two positions to a user, marking one as primary
- **THEN** both associations are persisted and exactly one is marked primary

### Requirement: Enforce single primary position per user
The system SHALL guarantee that a user has at most one primary position at any time.

#### Scenario: Switch primary position
- **WHEN** an admin assigns a new primary position to a user who already has one
- **THEN** the previous primary position is automatically demoted to non-primary and the new one becomes primary

### Requirement: Personnel assignment API
The system SHALL expose endpoints to retrieve and update a user’s position assignments.

#### Scenario: Get user positions
- **WHEN** a client calls `GET /api/v1/org/users/:id/positions`
- **THEN** the system returns the full list of departments and positions assigned to that user

#### Scenario: Batch update user positions
- **WHEN** a client calls `PUT /api/v1/org/users/:id/positions` with a list of assignments
- **THEN** the system replaces all existing assignments for that user with the provided list

### Requirement: Scope helper for department-based data filtering
The system SHALL provide a service helper to retrieve all department IDs accessible to a user, including sub-departments.

#### Scenario: Retrieve user department scope
- **WHEN** a module requests the department scope for a user with a primary position in a parent department
- **THEN** the helper returns the parent department ID and all its descendant department IDs
