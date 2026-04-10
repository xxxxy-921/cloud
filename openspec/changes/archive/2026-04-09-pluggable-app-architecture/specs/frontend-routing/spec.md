## MODIFIED Requirements

### Requirement: Route definitions
The application SHALL define kernel routes for home, config, users, roles, menus, sessions, tasks, announcements, channels, auth-providers, audit-logs, settings, login, oauth callback, and a 404 fallback. Additionally, the router SHALL merge routes from all registered App modules via `getAppRoutes()`.

#### Scenario: Kernel routes unchanged
- **WHEN** the user navigates to any existing route (/users, /roles, /settings, etc.)
- **THEN** the existing page SHALL render inside the DashboardLayout as before

#### Scenario: App module routes merged
- **WHEN** App modules have registered routes via `registerApp()`
- **THEN** those routes SHALL appear as children of the DashboardLayout route, alongside kernel routes

#### Scenario: App module route with PermissionGuard
- **WHEN** an App route requires permission checking
- **THEN** the App's module.ts SHALL wrap its route components with PermissionGuard, same as kernel routes
