## MODIFIED Requirements

### Requirement: Route definitions
The application SHALL define kernel routes for config, users, roles, menus, sessions, tasks, announcements, channels, auth-providers, audit-logs, settings, login, oauth callback, and a 404 fallback. The root path `/` SHALL render a DefaultRedirect component that redirects to the first available menu path. Additionally, the router SHALL merge routes from all registered App modules via `getAppRoutes()`.

#### Scenario: Root path redirect
- **WHEN** the user navigates to /
- **THEN** the DefaultRedirect component SHALL read the menu store and redirect (replace) to the first menu child of the first directory in the menu tree

#### Scenario: Root path redirect fallback
- **WHEN** the user navigates to / and the menu tree is empty
- **THEN** the DefaultRedirect component SHALL redirect to /users as fallback

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

#### Scenario: Kernel routes unchanged
- **WHEN** the user navigates to any existing route (/users, /roles, /settings, etc.)
- **THEN** the existing page SHALL render inside the DashboardLayout as before

#### Scenario: App module routes merged
- **WHEN** App modules have registered routes via `registerApp()`
- **THEN** those routes SHALL appear as children of the DashboardLayout route, alongside kernel routes

### Requirement: Breadcrumb from route segments
The header breadcrumb SHALL be generated from the current route pathname segments. The breadcrumb SHALL NOT include a "首页" root entry.

#### Scenario: Nested breadcrumb
- **WHEN** the user is on /config
- **THEN** the breadcrumb SHALL show "系统配置" (without "首页" prefix)

## REMOVED Requirements

### Requirement: Home route
**Reason**: 首页仅显示欢迎文案无实际功能，已被根路径重定向替代
**Migration**: 根路径 `/` 自动重定向到第一个可用菜单
