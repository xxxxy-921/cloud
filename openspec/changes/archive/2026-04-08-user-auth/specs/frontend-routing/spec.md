## MODIFIED Requirements

### Requirement: Route definitions
The application SHALL define routes for home, config, users (admin-only), login, and a 404 fallback.

#### Scenario: Home route
- **WHEN** the user navigates to /
- **THEN** the home page SHALL render inside the DashboardLayout

#### Scenario: Config route
- **WHEN** the user navigates to /config
- **THEN** the system config page SHALL render inside the DashboardLayout

#### Scenario: Users route (admin only)
- **WHEN** an admin user navigates to /users
- **THEN** the user management page SHALL render inside the DashboardLayout

#### Scenario: Login route
- **WHEN** the user navigates to /login
- **THEN** the login page SHALL render full-screen without DashboardLayout

#### Scenario: Unknown route
- **WHEN** the user navigates to an undefined path
- **THEN** a 404 not-found page SHALL be displayed

#### Scenario: Unauthenticated redirect
- **WHEN** an unauthenticated user navigates to any route except /login
- **THEN** the system SHALL redirect to /login
