# Capability: menu-system

## Purpose
Provides the menu model, tree-structured menu API, and frontend dynamic menu loading for sidebar navigation based on user permissions.

## Requirements

### Requirement: Menu model
The system SHALL store menus in a self-referencing tree structure with ParentID (*uint, NULL for root), Name, Type (enum: "directory"/"menu"/"button"), Path (route path for menu type), Icon (icon name string), Permission (unique permission identifier like "system:user:list"), Sort (ordering weight), and IsHidden (hide from sidebar). The Menu model SHALL embed BaseModel.

#### Scenario: Create root directory menu
- **WHEN** a menu is created with type "directory", name "系统管理", parentID nil
- **THEN** the system SHALL store a root-level directory menu entry

#### Scenario: Create child menu under directory
- **WHEN** a menu is created with type "menu", name "用户管理", parentID=1, path "/users", permission "system:user:list"
- **THEN** the system SHALL store a child menu entry linked to its parent

#### Scenario: Create button under menu
- **WHEN** a menu is created with type "button", name "新增用户", parentID=2, permission "system:user:create"
- **THEN** the system SHALL store a button entry (no path, no icon) under the parent menu

#### Scenario: Permission uniqueness
- **WHEN** a menu with permission "system:user:list" already exists and another menu is created with the same permission
- **THEN** the system SHALL return a unique constraint violation error (only for non-empty permission values)

### Requirement: Menu tree API
The system SHALL provide endpoints for menu management and user-specific menu tree retrieval.

#### Scenario: Get full menu tree (admin)
- **WHEN** GET /api/v1/menus/tree
- **THEN** the system SHALL return the complete menu tree with nested children, sorted by Sort field at each level

#### Scenario: Get user menu tree
- **WHEN** GET /api/v1/menus/user-tree for an authenticated user
- **THEN** the system SHALL return only the menus that the user's role has permission to access, structured as a tree with nested children

#### Scenario: User menu tree filtering logic
- **WHEN** building the user menu tree for a role
- **THEN** the system SHALL include a menu if the role has a Casbin policy for any of its descendant permissions; directory nodes SHALL be included if any of their children are included

#### Scenario: Create menu
- **WHEN** POST /api/v1/menus with `{name: "日志管理", type: "menu", parentId: 1, path: "/logs", permission: "system:log:list"}`
- **THEN** the system SHALL create the menu and return the menu record

#### Scenario: Update menu
- **WHEN** PUT /api/v1/menus/:id with `{name: "审计日志", sort: 10}`
- **THEN** the system SHALL update the specified fields

#### Scenario: Delete menu
- **WHEN** DELETE /api/v1/menus/:id
- **THEN** the system SHALL soft-delete the menu and all its descendant menus

#### Scenario: Cannot delete menu with children
- **WHEN** DELETE /api/v1/menus/:id and the menu has child menus
- **THEN** the system SHALL return 400 with message "cannot delete menu with children, delete children first"

### Requirement: Frontend dynamic menu loading
The frontend SHALL load the user's menu tree from the backend after login and use it to render the sidebar navigation.

#### Scenario: Menu store initialization
- **WHEN** a user logs in successfully
- **THEN** the frontend SHALL call GET /api/v1/menus/user-tree and store the menu tree in a Zustand menuStore

#### Scenario: Sidebar rendering from menu tree
- **WHEN** the sidebar renders
- **THEN** it SHALL iterate the menu tree from menuStore, rendering directory nodes as collapsible groups and menu nodes as navigable links

#### Scenario: Menu refresh on page reload
- **WHEN** the user refreshes the page
- **THEN** the authStore.init() flow SHALL also trigger menuStore re-initialization from the API

#### Scenario: Hidden menu handling
- **WHEN** a menu has IsHidden=true
- **THEN** the sidebar SHALL not display it, but the route SHALL still be accessible if the user has permission

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
