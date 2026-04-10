## ADDED Requirements

### Requirement: usePermission hook
The frontend SHALL provide a `usePermission` hook that checks whether the current user has a specific permission code.

#### Scenario: User has permission
- **WHEN** usePermission("system:user:create") is called and the user's permission list includes "system:user:create"
- **THEN** the hook SHALL return true

#### Scenario: User lacks permission
- **WHEN** usePermission("system:user:delete") is called and the user's permission list does not include it
- **THEN** the hook SHALL return false

### Requirement: Button-level permission control
The frontend SHALL conditionally render action buttons based on the user's menu permissions (button-type menu entries).

#### Scenario: Show create button with permission
- **WHEN** the user has "system:user:create" permission and visits the users page
- **THEN** the "新增用户" button SHALL be visible

#### Scenario: Hide delete button without permission
- **WHEN** the user does not have "system:user:delete" permission and visits the users page
- **THEN** the "删除用户" button SHALL not be rendered

### Requirement: PermissionGuard component
The frontend SHALL provide a `PermissionGuard` component that replaces AdminGuard, checking permissions instead of role strings.

#### Scenario: Route with required permission
- **WHEN** a route is wrapped with `<PermissionGuard permission="system:user:list">` and the user has the permission
- **THEN** the child page SHALL render normally

#### Scenario: Route without required permission
- **WHEN** a route is wrapped with `<PermissionGuard permission="system:user:list">` and the user does not have the permission
- **THEN** the component SHALL display a 403 "无权访问" (access denied) page
