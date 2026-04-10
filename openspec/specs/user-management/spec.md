# Capability: user-management

## Purpose
Admin-only user CRUD API endpoints and CLI tool for managing users, including listing, creating, updating, deleting, password reset, activation/deactivation, and initial admin creation.

## Requirements

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

### Requirement: User management page table actions
The user management page table SHALL display row actions in a `DropdownMenu` triggered by a `MoreHorizontal` icon button, instead of inline buttons. The menu SHALL contain: "编辑", "停用/启用", a separator, and "删除" (in destructive styling). The delete action SHALL still trigger an AlertDialog for confirmation.

#### Scenario: User clicks action menu
- **WHEN** user clicks the MoreHorizontal button on a user row
- **THEN** a DropdownMenu SHALL open with edit, toggle-active, and delete options

#### Scenario: Delete from action menu
- **WHEN** user selects "删除" from the action menu
- **THEN** an AlertDialog SHALL appear for confirmation before deletion

### Requirement: Role management page table actions
The role management page table SHALL display row actions in a `DropdownMenu` triggered by a `MoreHorizontal` icon button. The menu SHALL contain: "权限", "编辑", a separator, and "删除" (in destructive styling, disabled for system roles).

#### Scenario: Role action menu
- **WHEN** user clicks the MoreHorizontal button on a role row
- **THEN** a DropdownMenu SHALL open with permission-assign, edit, and delete options

### Requirement: Form selects use shadcn Select
All form select inputs in Sheet forms SHALL use shadcn `Select` component instead of native `<select>` elements, ensuring visual consistency with other shadcn form components.

#### Scenario: Menu sheet parent and type selects
- **WHEN** the menu sheet form is displayed
- **THEN** the parent menu selector and type selector SHALL use shadcn `Select` components

#### Scenario: User sheet role select
- **WHEN** the user sheet form is displayed
- **THEN** the role selector SHALL use a shadcn `Select` component

### Requirement: Role sheet cancel button
The role edit/create Sheet SHALL include a "取消" (cancel) button in the footer alongside the "保存" button, consistent with the permission dialog footer.

#### Scenario: Role sheet footer buttons
- **WHEN** the role sheet is open
- **THEN** the footer SHALL display both "取消" and "保存" buttons
