## MODIFIED Requirements

### Requirement: List users (admin)
The system SHALL provide `GET /api/v1/users` (requires admin role) returning a paginated list of users with search and filter support. Each user record SHALL include a `connections` field listing bound OAuth providers.

#### Scenario: List all users
- **WHEN** admin GET /api/v1/users
- **THEN** the system SHALL return `{code: 0, data: {items: [...], total, page, pageSize}}` with user records (excluding password hash), each including `connections: [{provider, externalName}]` and `hasPassword: bool`

#### Scenario: Search users
- **WHEN** admin GET /api/v1/users?keyword=alice
- **THEN** the system SHALL return users whose username, email, or phone contains "alice"

#### Scenario: Filter by status
- **WHEN** admin GET /api/v1/users?isActive=true
- **THEN** the system SHALL return only active users

### Requirement: User management page (admin)
The frontend SHALL provide a user management page at `/users` accessible only to admin users, with a table listing all users and actions for create, edit, activate/deactivate, reset password, and delete. The table SHALL display a login method indicator column showing icons for local (password) and/or OAuth providers (GitHub/Google).

#### Scenario: Admin views user list
- **WHEN** admin navigates to /users
- **THEN** the page SHALL display a searchable, paginated table of users with columns: username, email, phone, role, login methods (icons), status, actions

#### Scenario: Login method icons
- **WHEN** a user has password set and a GitHub connection
- **THEN** the login methods column SHALL show a key icon (password) and the GitHub icon

#### Scenario: OAuth-only user
- **WHEN** a user has no password and only a Google connection
- **THEN** the login methods column SHALL show only the Google icon

#### Scenario: Non-admin access
- **WHEN** a user with role "user" navigates to /users
- **THEN** the system SHALL show a 403 forbidden message or redirect to /
