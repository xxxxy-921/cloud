## ADDED Requirements

### Requirement: Seed CLI subcommand
The system SHALL provide a `metis seed` CLI subcommand that idempotently initializes built-in roles, menus, and Casbin policies.

#### Scenario: First run seed
- **WHEN** `metis seed` is run on a fresh database (after AutoMigrate)
- **THEN** the system SHALL create built-in roles (admin, user), the default menu tree, and Casbin policies for admin (full access), then exit with a summary

#### Scenario: Idempotent re-run
- **WHEN** `metis seed` is run and built-in roles already exist
- **THEN** the system SHALL skip existing records (match by role code or menu permission), only create missing entries, and report "X created, Y skipped"

#### Scenario: Seed output summary
- **WHEN** seed completes
- **THEN** the CLI SHALL print a summary: "Roles: N created, M skipped. Menus: N created, M skipped. Policies: N added."

### Requirement: Built-in roles seed data
The seed SHALL create two system roles: admin (code="admin", name="管理员", isSystem=true, sort=0) and user (code="user", name="普通用户", isSystem=true, sort=1).

#### Scenario: Admin role created
- **WHEN** seed runs and no role with code "admin" exists
- **THEN** the system SHALL create the admin role with IsSystem=true

#### Scenario: User role created
- **WHEN** seed runs and no role with code "user" exists
- **THEN** the system SHALL create the user role with IsSystem=true

### Requirement: Built-in menu tree seed data
The seed SHALL create the default menu tree including: top-level "首页" (menu, path "/"), "系统管理" (directory) with children "用户管理" (menu, path "/users", permission "system:user:list"), "角色管理" (menu, path "/roles", permission "system:role:list"), "菜单管理" (menu, path "/menus", permission "system:menu:list"), "系统配置" (menu, path "/config", permission "system:config:list"), "系统设置" (menu, path "/settings", permission "system:settings:list"). Each menu type entry SHALL also have button children for CRUD operations.

#### Scenario: Menu tree created with button permissions
- **WHEN** seed runs and the "用户管理" menu is created
- **THEN** it SHALL have button children: "新增用户" (permission "system:user:create"), "编辑用户" (permission "system:user:update"), "删除用户" (permission "system:user:delete"), "重置密码" (permission "system:user:reset-password")

### Requirement: Built-in Casbin policies seed
The seed SHALL create Casbin policies granting admin role full access to all API endpoints and all menu permissions. The user role SHALL get policies for basic access (home page, auth endpoints, view own profile).

#### Scenario: Admin full API access
- **WHEN** seed runs
- **THEN** admin role SHALL have Casbin policies for all registered API paths and methods

#### Scenario: Admin full menu access
- **WHEN** seed runs
- **THEN** admin role SHALL have Casbin policies for all menu permission identifiers with action "read"

#### Scenario: User basic access
- **WHEN** seed runs
- **THEN** user role SHALL have Casbin policies for home page access and basic auth endpoints only

### Requirement: User migration in seed
The seed SHALL migrate existing users from the old Role string field to the new RoleID foreign key.

#### Scenario: Migrate existing admin user
- **WHEN** seed runs and a user has Role="admin" but RoleID=0
- **THEN** the system SHALL set the user's RoleID to the admin role's ID

#### Scenario: Migrate existing regular user
- **WHEN** seed runs and a user has Role="user" but RoleID=0
- **THEN** the system SHALL set the user's RoleID to the user role's ID
