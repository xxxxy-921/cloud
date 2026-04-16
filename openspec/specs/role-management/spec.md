# Capability: role-management

## Purpose
Provides the Role model, CRUD API, User-Role association, and frontend role management page.

## Requirements

### Requirement: Role model
The system SHALL store roles with Name (display name), Code (unique identifier used as Casbin subject), Description, Sort (ordering weight), IsSystem (flag preventing deletion of built-in roles), and **DataScope** (data visibility scope enum: `all` | `dept_and_sub` | `dept` | `self` | `custom`, default `all`). The Role model SHALL embed BaseModel. For `custom` DataScope, the role SHALL have an associated `RoleDeptScope` collection.

#### Scenario: Create role record
- **WHEN** a new role is created with name "编辑员", code "editor"
- **THEN** the system SHALL store a Role record with auto-generated ID/timestamps, IsSystem=false, and DataScope="all"

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
The system SHALL provide a role management page at /roles with list view, create/edit Sheet, delete confirmation, and data scope configuration entry.

#### Scenario: View role list
- **WHEN** user navigates to /roles
- **THEN** the page SHALL display a table of roles with columns: name, code, description, sort, isSystem, dataScope (badge), actions

#### Scenario: Create role via Sheet
- **WHEN** user clicks "新增角色" button and fills the form
- **THEN** a new role SHALL be created and the list SHALL refresh

#### Scenario: System role indicators
- **WHEN** a role has IsSystem=true
- **THEN** the delete button SHALL be hidden and the code field SHALL be read-only in edit mode

#### Scenario: DataScope badge in role list
- **WHEN** user views the role list
- **THEN** each role row SHALL display a badge indicating the current dataScope (e.g., "全部数据", "本部门及下级", "仅本人")

### Requirement: Role dataScope field in CRUD API
The system SHALL include the `dataScope` field in all Role CRUD API responses and accept it in create/update requests.

#### Scenario: List roles includes dataScope
- **WHEN** GET /api/v1/roles
- **THEN** each role in the response SHALL include `dataScope` field (e.g., `"all"`, `"dept_and_sub"`)

#### Scenario: Create role with dataScope
- **WHEN** POST /api/v1/roles with `{name: "运维经理", code: "ops-manager", dataScope: "dept_and_sub"}`
- **THEN** the system SHALL create the role with the specified dataScope

#### Scenario: Role detail includes custom deptIds
- **WHEN** GET /api/v1/roles/:id for a role with dataScope `custom`
- **THEN** the response SHALL include `deptIds: [...]` with the configured department IDs

### Requirement: Test role creation
The role service test suite SHALL verify that role creation succeeds with valid input and rejects duplicate codes.

#### Scenario: Create role successfully
- **WHEN** `Create` is called with a unique name, code, description, and sort
- **THEN** it returns a role with `IsSystem=false` and `DataScope=all`

#### Scenario: Reject duplicate code
- **WHEN** `Create` is called with a code that already exists
- **THEN** it returns `ErrRoleCodeExists`

### Requirement: Test role retrieval
The test suite SHALL verify that roles can be retrieved by ID, with or without custom department scope, and that missing roles return `ErrRoleNotFound`.

#### Scenario: Get role by ID successfully
- **WHEN** `GetByID` is called with an existing role ID
- **THEN** it returns the role

#### Scenario: Get role by ID returns not found
- **WHEN** `GetByID` is called with a non-existent role ID
- **THEN** it returns `ErrRoleNotFound`

#### Scenario: Get role with custom department scope
- **WHEN** `GetByIDWithDeptScope` is called for a role with `DataScope=custom`
- **THEN** it returns the role and the associated department IDs

### Requirement: Test role update
The test suite SHALL verify that updates apply allowed fields, guard against system role code changes, reject duplicate codes, and migrate Casbin policies when the code changes.

#### Scenario: Update role successfully
- **WHEN** `Update` is called with new name and description for a non-system role
- **THEN** it returns the updated role with the new values persisted

#### Scenario: Reject duplicate code on update
- **WHEN** `Update` attempts to change a role's code to one that already exists
- **THEN** it returns `ErrRoleCodeExists`

#### Scenario: Prevent system role code change
- **WHEN** `Update` attempts to change the code of a system role (`IsSystem=true`)
- **THEN** it returns `ErrSystemRole`

#### Scenario: Migrate Casbin policies on code change
- **WHEN** `Update` successfully changes a non-system role's code
- **THEN** all existing Casbin policies for the old code are migrated to the new code

### Requirement: Test role data scope update
The test suite SHALL verify that data scope updates validate the scope value, replace custom department sets, and protect the admin system role.

#### Scenario: Update data scope successfully
- **WHEN** `UpdateDataScope` is called with `DataScope=custom` and a list of department IDs
- **THEN** it returns the role with the new scope and the department IDs persisted

#### Scenario: Clear custom department IDs when scope is not custom
- **WHEN** `UpdateDataScope` is called with `DataScope=all` for a role that previously had `DataScope=custom`
- **THEN** all `RoleDeptScope` entries for that role are removed

#### Scenario: Reject invalid data scope
- **WHEN** `UpdateDataScope` is called with an invalid scope value
- **THEN** it returns `ErrDataScopeInvalid`

#### Scenario: Prevent admin data scope change
- **WHEN** `UpdateDataScope` is called for the admin system role
- **THEN** it returns `ErrSystemRole`

### Requirement: Test role deletion
The test suite SHALL verify that deletion removes the role, cleans up Casbin policies and custom department scopes, and guards against deleting system roles or roles with assigned users.

#### Scenario: Delete role successfully
- **WHEN** `Delete` is called for a non-system role with no assigned users
- **THEN** the role is removed, its Casbin policies are deleted, and its `RoleDeptScope` entries are cleared

#### Scenario: Prevent system role deletion
- **WHEN** `Delete` is called for a system role
- **THEN** it returns `ErrSystemRoleDel`

#### Scenario: Prevent deletion when users are assigned
- **WHEN** `Delete` is called for a role that still has users assigned
- **THEN** it returns `ErrRoleHasUsers`
