## ADDED Requirements

### Requirement: Login page
The frontend SHALL provide a login page at `/login` with username and password fields, rendered full-screen without DashboardLayout.

#### Scenario: Successful login
- **WHEN** user enters valid credentials and submits the login form
- **THEN** the system SHALL call POST /api/v1/auth/login, store the returned token pair, and redirect to /

#### Scenario: Failed login
- **WHEN** user enters invalid credentials and submits
- **THEN** the system SHALL display the error message from the API response

#### Scenario: Already logged in
- **WHEN** a user with a valid token navigates to /login
- **THEN** the system SHALL redirect to /

### Requirement: Auth store (Zustand)
The frontend SHALL maintain an auth store with token pair, current user info, and methods for login, logout, refresh, and token persistence.

#### Scenario: Token persistence
- **WHEN** a user logs in successfully
- **THEN** the store SHALL save accessToken and refreshToken to localStorage

#### Scenario: Token restoration on load
- **WHEN** the application loads
- **THEN** the store SHALL restore tokens from localStorage and fetch current user info via GET /api/v1/auth/me

#### Scenario: Logout clears state
- **WHEN** user logs out
- **THEN** the store SHALL call POST /api/v1/auth/logout, clear tokens from localStorage, clear user state, and redirect to /login

### Requirement: API interceptor with auto-refresh
The API client SHALL automatically attach Bearer token to all requests and handle 401 responses by attempting token refresh.

#### Scenario: Attach Bearer token
- **WHEN** any API request is made and a valid access token exists
- **THEN** the API client SHALL include `Authorization: Bearer <accessToken>` header

#### Scenario: Auto-refresh on 401
- **WHEN** an API request returns 401 and a refresh token exists
- **THEN** the API client SHALL call POST /api/v1/auth/refresh, update stored tokens, and retry the original request

#### Scenario: Refresh fails
- **WHEN** token refresh also returns 401 (refresh token expired/revoked)
- **THEN** the API client SHALL clear auth state and redirect to /login

#### Scenario: Concurrent 401 handling
- **WHEN** multiple requests return 401 simultaneously
- **THEN** the API client SHALL only trigger one refresh request, queue other requests, and retry all after refresh completes

### Requirement: Protected route guard
The frontend SHALL redirect unauthenticated users to /login for all routes except /login itself.

#### Scenario: Unauthenticated access
- **WHEN** user navigates to / without a valid token
- **THEN** the system SHALL redirect to /login

#### Scenario: Authenticated access
- **WHEN** user navigates to / with a valid token
- **THEN** the system SHALL render the page normally

### Requirement: User management page (admin)
The frontend SHALL provide a user management page at `/users` accessible only to admin users, with a table listing all users and actions for create, edit, activate/deactivate, reset password, and delete.

#### Scenario: Admin views user list
- **WHEN** admin navigates to /users
- **THEN** the page SHALL display a searchable, paginated table of users with columns: username, email, phone, role, status, actions

#### Scenario: Non-admin access
- **WHEN** a user with role "user" navigates to /users
- **THEN** the system SHALL show a 403 forbidden message or redirect to /

#### Scenario: Create user
- **WHEN** admin clicks "create user" and fills the form
- **THEN** the system SHALL call POST /api/v1/users and refresh the list

#### Scenario: Edit user
- **WHEN** admin clicks edit on a user row
- **THEN** the system SHALL open a form sheet pre-filled with user data, and on submit call PUT /api/v1/users/:id

### Requirement: User menu in layout
The DashboardLayout SHALL display the current user's username and a dropdown menu with "change password" and "logout" options.

#### Scenario: Display username
- **WHEN** user is logged in
- **THEN** the layout header/topnav SHALL show the current user's username or avatar

#### Scenario: Change password
- **WHEN** user clicks "change password" in the dropdown
- **THEN** a dialog SHALL appear with old password, new password, and confirm password fields

#### Scenario: Logout
- **WHEN** user clicks "logout" in the dropdown
- **THEN** the system SHALL call the auth store's logout method

### Requirement: Navigation update for user management
The navigation SHALL include a "用户管理" entry under the "系统" section, visible only to admin users.

#### Scenario: Admin sees user management nav
- **WHEN** an admin user is logged in
- **THEN** the sidebar SHALL show "用户管理" under "系统" section

#### Scenario: Non-admin nav
- **WHEN** a non-admin user is logged in
- **THEN** the sidebar SHALL NOT show "用户管理"
