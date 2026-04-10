## MODIFIED Requirements

### Requirement: Login page
The login page at `/login` SHALL display a form with username and password fields. On successful login, the page SHALL redirect to `/`. If the user is already authenticated, the page SHALL redirect to `/` using a `<Navigate>` component instead of calling `navigate()` during render.

#### Scenario: Already logged in redirect
- **WHEN** an authenticated user visits `/login`
- **THEN** the page SHALL render `<Navigate to="/" replace />` instead of calling `navigate()` imperatively during render

#### Scenario: Successful login
- **WHEN** user submits valid credentials
- **THEN** the system SHALL call POST /api/v1/auth/login, store tokens, and navigate to `/`
