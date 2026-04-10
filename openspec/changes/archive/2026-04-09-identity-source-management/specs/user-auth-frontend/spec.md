## MODIFIED Requirements

### Requirement: Login page
The frontend SHALL provide a login page at `/login` with username and password fields. **The login page SHALL additionally include an optional email field for domain detection.** On email blur, it SHALL call `GET /api/v1/auth/check-domain?email=xxx` — if the endpoint returns 404 (no identity App) or no match, the page SHALL behave as current (username+password + OAuth buttons). If a match is returned, the page SHALL display an SSO button. If forceSso=true, the page SHALL hide the password form.

#### Scenario: Successful login
- **WHEN** user enters valid credentials and submits the login form
- **THEN** the system SHALL call POST /api/v1/auth/login, store tokens, redirect to /

#### Scenario: Failed login
- **WHEN** user enters invalid credentials and submits
- **THEN** the system SHALL display the error message

#### Scenario: Already logged in
- **WHEN** a user with a valid token navigates to /login
- **THEN** the system SHALL redirect to /

#### Scenario: Display OAuth buttons
- **WHEN** login page loads and GET /api/v1/auth/providers returns enabled providers
- **THEN** SHALL display OAuth buttons below the divider

#### Scenario: No OAuth providers available
- **WHEN** GET /api/v1/auth/providers returns empty array
- **THEN** SHALL not display divider or OAuth buttons

#### Scenario: Email domain detection (App loaded)
- **WHEN** user enters "john@acme.com" and field blurs, check-domain returns a match
- **THEN** SHALL display SSO login button (e.g., "通过 Okta SSO 登录")

#### Scenario: Email domain detection (no App)
- **WHEN** user enters email and check-domain returns 404
- **THEN** SHALL silently ignore, login page unchanged

#### Scenario: Forced SSO mode
- **WHEN** check-domain returns forceSso=true
- **THEN** SHALL hide username/password form, show only SSO button and "使用其他账号" link

#### Scenario: SSO available mode (not forced)
- **WHEN** check-domain returns forceSso=false
- **THEN** SHALL show SSO button alongside the password form

#### Scenario: Click SSO login button
- **WHEN** user clicks SSO button for OIDC source ID 3
- **THEN** SHALL call GET /api/v1/auth/sso/3/authorize, redirect to authURL

#### Scenario: Reset from forced SSO
- **WHEN** user clicks "使用其他账号"
- **THEN** SHALL clear email, return to default mode

## ADDED Requirements

### Requirement: SSO callback route（App 前端注册）
The identity App frontend SHALL register a route at `/sso/callback` via `registerApp()`. This route handles OIDC SSO redirects.

#### Scenario: Successful SSO callback
- **WHEN** browser redirects to /sso/callback?code=xxx&state=yyy
- **THEN** SHALL call POST /api/v1/auth/sso/callback, store TokenPair, redirect to /

#### Scenario: SSO callback error
- **WHEN** backend returns error (e.g., 409 email conflict)
- **THEN** SHALL redirect to /login with error message

#### Scenario: Missing code or state
- **WHEN** /sso/callback accessed without code or state
- **THEN** SHALL redirect to /login with error
