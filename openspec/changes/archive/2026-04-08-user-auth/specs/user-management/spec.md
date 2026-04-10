## ADDED Requirements

### Requirement: List users (admin)
The system SHALL provide `GET /api/v1/users` (requires admin role) returning a paginated list of users with search and filter support.

#### Scenario: List all users
- **WHEN** admin GET /api/v1/users
- **THEN** the system SHALL return `{code: 0, data: {items: [...], total, page, pageSize}}` with user records (excluding password hash)

#### Scenario: Search users
- **WHEN** admin GET /api/v1/users?keyword=alice
- **THEN** the system SHALL return users whose username, email, or phone contains "alice"

#### Scenario: Filter by status
- **WHEN** admin GET /api/v1/users?isActive=true
- **THEN** the system SHALL return only active users

### Requirement: Create user (admin)
The system SHALL provide `POST /api/v1/users` (requires admin role) to create a new user with specified username, password, email, phone, role.

#### Scenario: Create user successfully
- **WHEN** admin POST /api/v1/users with `{username: "bob", password: "pass123", role: "user"}`
- **THEN** the system SHALL create a user with bcrypt-hashed password and return the user record (excluding password hash)

#### Scenario: Duplicate username
- **WHEN** admin POST /api/v1/users with a username that already exists
- **THEN** the system SHALL return 400 with message "username already exists"

### Requirement: Get user detail (admin)
The system SHALL provide `GET /api/v1/users/:id` (requires admin role) returning a single user's full profile.

#### Scenario: User exists
- **WHEN** admin GET /api/v1/users/1
- **THEN** the system SHALL return the user record (excluding password hash)

#### Scenario: User not found
- **WHEN** admin GET /api/v1/users/999 and user does not exist
- **THEN** the system SHALL return 404 with message "user not found"

### Requirement: Update user (admin)
The system SHALL provide `PUT /api/v1/users/:id` (requires admin role) to update user's email, phone, avatar, role, is_active.

#### Scenario: Update user fields
- **WHEN** admin PUT /api/v1/users/1 with `{email: "new@example.com", role: "admin"}`
- **THEN** the system SHALL update the specified fields and return the updated user record

#### Scenario: Cannot update own role
- **WHEN** admin PUT /api/v1/users/:id where :id is their own user ID and the payload includes a role change
- **THEN** the system SHALL return 400 with message "cannot change own role"

### Requirement: Delete user (admin)
The system SHALL provide `DELETE /api/v1/users/:id` (requires admin role) to soft-delete a user.

#### Scenario: Delete user
- **WHEN** admin DELETE /api/v1/users/2
- **THEN** the system SHALL soft-delete the user (GORM DeletedAt) and revoke all their refresh tokens

#### Scenario: Cannot delete self
- **WHEN** admin DELETE /api/v1/users/:id where :id is their own user ID
- **THEN** the system SHALL return 400 with message "cannot delete self"

### Requirement: Reset user password (admin)
The system SHALL provide `POST /api/v1/users/:id/reset-password` (requires admin role) to set a new password for a user.

#### Scenario: Reset password
- **WHEN** admin POST /api/v1/users/2/reset-password with `{password: "newpass123"}`
- **THEN** the system SHALL update the user's password hash and revoke all their refresh tokens

### Requirement: Activate user (admin)
The system SHALL provide `POST /api/v1/users/:id/activate` (requires admin role) to set is_active=true.

#### Scenario: Activate disabled user
- **WHEN** admin POST /api/v1/users/2/activate for a user with is_active=false
- **THEN** the system SHALL set is_active=true and return the updated user record

### Requirement: Deactivate user (admin)
The system SHALL provide `POST /api/v1/users/:id/deactivate` (requires admin role) to set is_active=false and revoke all tokens.

#### Scenario: Deactivate user
- **WHEN** admin POST /api/v1/users/2/deactivate
- **THEN** the system SHALL set is_active=false and revoke all refresh tokens for the user

#### Scenario: Cannot deactivate self
- **WHEN** admin POST /api/v1/users/:id/deactivate where :id is their own user ID
- **THEN** the system SHALL return 400 with message "cannot deactivate self"

### Requirement: CLI create-admin subcommand
The binary SHALL support a `create-admin` subcommand: `metis create-admin --username=xxx --password=xxx`.

#### Scenario: Create admin via CLI
- **WHEN** running `metis create-admin --username=admin --password=admin123`
- **THEN** the system SHALL initialize the database, create a user with role "admin" and the specified credentials, then exit

#### Scenario: Username already exists via CLI
- **WHEN** running `metis create-admin --username=admin` and "admin" already exists
- **THEN** the system SHALL print an error message and exit with non-zero code

#### Scenario: Missing required flags
- **WHEN** running `metis create-admin` without --username or --password
- **THEN** the system SHALL print usage information and exit with non-zero code
