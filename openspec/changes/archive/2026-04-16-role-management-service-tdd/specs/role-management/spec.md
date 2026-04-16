## ADDED Requirements

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
