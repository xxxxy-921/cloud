## MODIFIED Requirements

### Requirement: Login page
The frontend SHALL provide a login page at `/login` with username and password fields, rendered full-screen without DashboardLayout. Below the login form, a divider "或" SHALL separate the local login form from dynamically rendered OAuth provider buttons. The OAuth buttons SHALL be fetched from `GET /api/v1/auth/providers` on page load and only displayed when enabled providers exist.

#### Scenario: Successful login
- **WHEN** user enters valid credentials and submits the login form
- **THEN** the system SHALL call POST /api/v1/auth/login, store the returned token pair, and redirect to /

#### Scenario: Failed login
- **WHEN** user enters invalid credentials and submits
- **THEN** the system SHALL display the error message from the API response

#### Scenario: Already logged in
- **WHEN** a user with a valid token navigates to /login
- **THEN** the system SHALL redirect to /

#### Scenario: Display OAuth buttons
- **WHEN** the login page loads and GET /api/v1/auth/providers returns [{providerKey: "github", displayName: "GitHub"}]
- **THEN** the login page SHALL display a "GitHub 登录" button with the GitHub icon below the divider

#### Scenario: No OAuth providers available
- **WHEN** the login page loads and GET /api/v1/auth/providers returns an empty array
- **THEN** the login page SHALL not display the divider or any OAuth buttons

#### Scenario: Click OAuth button
- **WHEN** user clicks the "GitHub 登录" button
- **THEN** the system SHALL call GET /api/v1/auth/oauth/github, receive an authURL, and redirect the browser to that URL via window.location.href

### Requirement: OAuth callback route
The frontend SHALL provide a route at `/oauth/callback` that handles the OAuth provider's redirect, extracts `code` and `state` from URL query parameters, calls the backend callback API, and stores the resulting TokenPair.

#### Scenario: Successful OAuth callback
- **WHEN** the browser is redirected to /oauth/callback?code=xxx&state=yyy
- **THEN** the frontend SHALL call POST /api/v1/auth/oauth/callback with {code, state}, store the returned TokenPair in auth store, and redirect to /

#### Scenario: OAuth callback error
- **WHEN** the backend returns an error for the OAuth callback (e.g., email conflict 409)
- **THEN** the frontend SHALL redirect to /login and display the error message

#### Scenario: Missing code or state
- **WHEN** /oauth/callback is accessed without code or state parameters
- **THEN** the frontend SHALL redirect to /login with an error message

### Requirement: Auth store (Zustand)
The frontend SHALL maintain an auth store with token pair, current user info, and methods for login, logout, refresh, OAuth login, and token persistence.

#### Scenario: Token persistence
- **WHEN** a user logs in successfully (via password or OAuth)
- **THEN** the store SHALL save accessToken and refreshToken to localStorage

#### Scenario: Token restoration on load
- **WHEN** the application loads
- **THEN** the store SHALL restore tokens from localStorage and fetch current user info via GET /api/v1/auth/me

#### Scenario: Logout clears state
- **WHEN** user logs out
- **THEN** the store SHALL call POST /api/v1/auth/logout, clear tokens from localStorage, clear user state, and redirect to /login

#### Scenario: OAuth login method
- **WHEN** the OAuth callback page calls the store's oauthLogin method with a TokenPair
- **THEN** the store SHALL save the tokens, fetch user info via GET /api/v1/auth/me, and update the menu store

## ADDED Requirements

### Requirement: Account connections management in settings
The frontend SHALL provide an "账号关联" card in the settings page displaying the user's bound external accounts with bind/unbind actions.

#### Scenario: Display bound accounts
- **WHEN** user navigates to settings and has a GitHub connection
- **THEN** the settings page SHALL show an "账号关联" card with GitHub listed as bound, showing the external username and an "解绑" button

#### Scenario: Display available providers
- **WHEN** user navigates to settings and Google is enabled but not bound
- **THEN** the settings page SHALL show Google as available with a "绑定" button

#### Scenario: Bind external account
- **WHEN** user clicks "绑定" on Google
- **THEN** the frontend SHALL call POST /api/v1/auth/connections/google, receive an authURL, and redirect to the OAuth flow. After callback, the connection list SHALL refresh.

#### Scenario: Unbind external account
- **WHEN** user clicks "解绑" on GitHub
- **THEN** the frontend SHALL call DELETE /api/v1/auth/connections/github and refresh the connection list

#### Scenario: Cannot unbind last login method
- **WHEN** user attempts to unbind their only connection and has no password
- **THEN** the frontend SHALL display the error message "cannot unbind last login method, please set a password first"
