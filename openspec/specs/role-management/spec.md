# Capability: role-management

## Purpose
Provides the Role model, CRUD API, User-Role association, and frontend role management page.

## Requirements

### Requirement: Role model
The system SHALL store roles with Name (display name), Code (unique identifier used as Casbin subject), Description, Sort (ordering weight), and IsSystem (flag preventing deletion of built-in roles). The Role model SHALL embed BaseModel.

#### Scenario: Create role record
- **WHEN** a new role is created with name "编辑员", code "editor"
- **THEN** the system SHALL store a Role record with auto-generated ID/timestamps and IsSystem=false

#### Scenario: Code uniqueness
- **WHEN** a role with code "admin" already exists and another role is created with the same code
- **THEN** the system SHALL return a unique constraint violation error

### Requirement: Role CRUD API
The system SHALL provide REST endpoints for role management, all requiring authentication and appropriate permission.

#### Scenario: List roles
- **WHEN** GET /api/v1/roles
- **THEN** the system SHALL return `{code: 0, data: {items: [...], total, page, pageSize}}` with role records sorted by Sort field

#### Scenario: Create role
- **WHEN** POST /api/v1/roles with `{name: "编辑员", code: "editor", description: "内容编辑"}`
- **THEN** the system SHALL create the role and return the role record

#### Scenario: Duplicate code on create
- **WHEN** POST /api/v1/roles with a code that already exists
- **THEN** the system SHALL return 400 with message "role code already exists"

#### Scenario: Get role detail
- **WHEN** GET /api/v1/roles/:id
- **THEN** the system SHALL return the role record

#### Scenario: Role not found
- **WHEN** GET /api/v1/roles/999 and role does not exist
- **THEN** the system SHALL return 404 with message "role not found"

#### Scenario: Update role
- **WHEN** PUT /api/v1/roles/:id with `{name: "高级编辑", description: "updated"}`
- **THEN** the system SHALL update the specified fields and return the updated role record

#### Scenario: Cannot update system role code
- **WHEN** PUT /api/v1/roles/:id for a system role (IsSystem=true) with a code change
- **THEN** the system SHALL return 400 with message "cannot modify system role code"

#### Scenario: Delete role
- **WHEN** DELETE /api/v1/roles/:id for a non-system role
- **THEN** the system SHALL soft-delete the role and remove all Casbin policies for this role's code

#### Scenario: Cannot delete system role
- **WHEN** DELETE /api/v1/roles/:id for a system role (IsSystem=true)
- **THEN** the system SHALL return 400 with message "cannot delete system role"

#### Scenario: Cannot delete role with assigned users
- **WHEN** DELETE /api/v1/roles/:id and users are still assigned to this role
- **THEN** the system SHALL return 400 with message "cannot delete role with assigned users"

### Requirement: User-Role association
The User model SHALL replace the `Role string` field with `RoleID uint` foreign key referencing the Role table. User API responses SHALL include the role object instead of a role string.

#### Scenario: User with role object in response
- **WHEN** GET /api/v1/auth/me or GET /api/v1/users/:id
- **THEN** the user response SHALL include `role: {id, name, code}` instead of `role: "admin"`

#### Scenario: Create user with roleID
- **WHEN** POST /api/v1/users with `{username: "bob", password: "pass", roleId: 2}`
- **THEN** the system SHALL create the user associated with the role of ID 2

#### Scenario: Invalid roleID on create
- **WHEN** POST /api/v1/users with a roleId that does not exist
- **THEN** the system SHALL return 400 with message "role not found"

### Requirement: Role management frontend page
The system SHALL provide a role management page at /roles with list view, create/edit dialog, and delete confirmation.

#### Scenario: View role list
- **WHEN** user navigates to /roles
- **THEN** the page SHALL display a table of roles with columns: name, code, description, sort, isSystem, actions

#### Scenario: Create role via dialog
- **WHEN** user clicks "新增角色" button and fills the form
- **THEN** a new role SHALL be created and the list SHALL refresh

#### Scenario: System role indicators
- **WHEN** a role has IsSystem=true
- **THEN** the delete button SHALL be hidden and the code field SHALL be read-only in edit mode
