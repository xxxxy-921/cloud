## ADDED Requirements

### Requirement: React Router SPA routing
The application SHALL use React Router 7 with createBrowserRouter for client-side routing.

#### Scenario: Dashboard layout wrapping
- **WHEN** the user navigates to any protected route (/, /users, /config)
- **THEN** the DashboardLayout (TopNav + Sidebar + Header + Content) SHALL wrap the page content

#### Scenario: Login page without layout
- **WHEN** the user navigates to /login
- **THEN** the page SHALL render full-screen without the DashboardLayout

### Requirement: Route definitions
The application SHALL define routes for home, config, and a 404 fallback.

#### Scenario: Home route
- **WHEN** the user navigates to /
- **THEN** the home page SHALL render inside the DashboardLayout

#### Scenario: Config route
- **WHEN** the user navigates to /config
- **THEN** the system config page SHALL render inside the DashboardLayout

#### Scenario: Unknown route
- **WHEN** the user navigates to an undefined path
- **THEN** a 404 not-found page SHALL be displayed

### Requirement: Breadcrumb from route segments
The header breadcrumb SHALL be generated from the current route pathname segments.

#### Scenario: Nested breadcrumb
- **WHEN** the user is on /config
- **THEN** the breadcrumb SHALL show "首页 / 系统配置"
