# Capability: menu-management-service-test

## Purpose
Defines service-layer test requirements and scenarios for the menu management module.

## Requirements

### Requirement: Menu service test infrastructure
The system SHALL provide a test harness for `MenuService` using an in-memory SQLite database, real GORM repositories, and a real Casbin enforcer, consistent with other kernel service tests.

#### Scenario: Setup test database
- **WHEN** a menu service test initializes
- **THEN** it SHALL migrate `Menu` and `CasbinRule` tables into a shared-memory SQLite database

#### Scenario: Setup DI container
- **WHEN** a menu service test needs dependencies
- **THEN** it SHALL use `samber/do` to provide the database, enforcer, `MenuRepo`, and `MenuService`

### Requirement: Test menu tree retrieval
The service-layer test suite SHALL verify that `GetTree` returns the complete menu tree sorted by `Sort` at each level.

#### Scenario: Get full tree sorted
- **WHEN** menus are seeded with mixed sort values across multiple levels
- **THEN** `GetTree` returns all menus in a nested tree with children sorted by `Sort` ascending

### Requirement: Test user menu tree with permission filtering
The service-layer test suite SHALL verify that `GetUserTree` returns only permitted menus, retains parent directories when descendants are permitted, and returns all menus for the admin role.

#### Scenario: Admin gets full tree
- **WHEN** `GetUserTree` is called with role code "admin"
- **THEN** it returns the complete menu tree regardless of Casbin policies

#### Scenario: Role sees only permitted menus
- **WHEN** a role has only the "system:user:list" permission
- **THEN** `GetUserTree` returns the directory ancestor, the "用户管理" menu, and any button descendants with matching permission

#### Scenario: Parent directory retained for descendant permission
- **WHEN** a role has a button permission but not the parent menu or directory permission
- **THEN** the directory and menu ancestors are still included so the tree structure is valid

#### Scenario: Hidden menus included in user tree
- **WHEN** a permitted menu has `IsHidden=true`
- **THEN** `GetUserTree` still includes it (hidden is a UI concern, not an access concern)

### Requirement: Test user permissions list
The service-layer test suite SHALL verify that `GetUserPermissions` returns all unique permission strings for a role's accessible menus.

#### Scenario: Get permissions for role
- **WHEN** `GetUserPermissions` is called for a role with access to several menus and buttons
- **THEN** it returns a list of all non-empty permission strings from those menus

### Requirement: Test menu creation
The service-layer test suite SHALL verify that `Create` inserts a menu and assigns the correct parent relationship.

#### Scenario: Create menu successfully
- **WHEN** `Create` is called with name, type, parentID, path, permission, and sort
- **THEN** it returns the menu with an assigned ID and the provided fields persisted

#### Scenario: Create root directory
- **WHEN** `Create` is called with type "directory" and parentID nil
- **THEN** it returns a root-level menu with no parent

### Requirement: Test menu update
The service-layer test suite SHALL verify that `Update` modifies allowed fields and returns the updated menu.

#### Scenario: Update name and sort
- **WHEN** `Update` is called with new name and sort for an existing menu
- **THEN** it returns the menu with updated fields

#### Scenario: Update parent
- **WHEN** `Update` is called with a new parentID
- **THEN** the menu is moved under the new parent

#### Scenario: Update not found
- **WHEN** `Update` is called for a non-existent menu ID
- **THEN** it returns `ErrMenuNotFound`

### Requirement: Test menu reordering
The service-layer test suite SHALL verify that `ReorderMenus` batch-updates the `Sort` field for multiple menus.

#### Scenario: Reorder multiple menus
- **WHEN** `ReorderMenus` is called with a list of `{id, sort}` pairs
- **THEN** each menu's `Sort` is updated to the new value

### Requirement: Test menu deletion
The service-layer test suite SHALL verify that `Delete` removes a leaf menu and prevents deletion when children exist.

#### Scenario: Delete leaf menu
- **WHEN** `Delete` is called for a menu with no children
- **THEN** the menu is soft-deleted

#### Scenario: Prevent deletion with children
- **WHEN** `Delete` is called for a menu that has child menus
- **THEN** it returns `ErrMenuHasChildren`

#### Scenario: Delete not found
- **WHEN** `Delete` is called for a non-existent menu ID
- **THEN** it returns `ErrMenuNotFound`
