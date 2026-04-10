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
