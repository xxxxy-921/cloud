## MODIFIED Requirements

### Requirement: Route definitions
The application SHALL define routes for home, config, users, roles, menus, settings, login, and a 404 fallback. Admin-only routes SHALL be wrapped with PermissionGuard instead of AdminGuard.

#### Scenario: Home route
- **WHEN** the user navigates to /
- **THEN** the home page SHALL render inside the DashboardLayout

#### Scenario: Config route
- **WHEN** the user navigates to /config
- **THEN** the system config page SHALL render inside the DashboardLayout, wrapped with PermissionGuard requiring "system:config:list"

#### Scenario: Users route
- **WHEN** a user with "system:user:list" permission navigates to /users
- **THEN** the user management page SHALL render inside the DashboardLayout

#### Scenario: Roles route
- **WHEN** a user with "system:role:list" permission navigates to /roles
- **THEN** the role management page SHALL render inside the DashboardLayout

#### Scenario: Menus route
- **WHEN** a user with "system:menu:list" permission navigates to /menus
- **THEN** the menu management page SHALL render inside the DashboardLayout

#### Scenario: Login route
- **WHEN** the user navigates to /login
- **THEN** the login page SHALL render full-screen without DashboardLayout

#### Scenario: Unknown route
- **WHEN** the user navigates to an undefined path
- **THEN** a 404 not-found page SHALL be displayed

#### Scenario: Unauthenticated redirect
- **WHEN** an unauthenticated user navigates to any route except /login
- **THEN** the system SHALL redirect to /login

#### Scenario: Unauthorized route access
- **WHEN** an authenticated user navigates to a route they lack permission for
- **THEN** the PermissionGuard SHALL display a 403 "无权访问" page
