# Capability: user-auth-frontend

## Purpose
Frontend authentication UI and state management, including login page, auth store with token persistence, API interceptor with auto-refresh, protected route guard, user management page, and user menu in layout.

## Requirements

### Requirement: Login page
The frontend SHALL provide a login page at `/login` with username and password fields, rendered full-screen without DashboardLayout. Below the password field, a captcha area SHALL be conditionally rendered: on page load, call `GET /api/v1/captcha` -- if `enabled=true`, display the captcha image, a text input for the answer, and a refresh button; if `enabled=false`, hide the captcha area. Below the login form, a divider "或" SHALL separate the local login form from dynamically rendered OAuth provider buttons. Below the OAuth section (or below the form if no OAuth), a registration link SHALL be conditionally shown: call `GET /api/v1/auth/registration-status` -- if `registrationOpen=true`, display "还没有账号？立即注册" linking to `/register`. On login failure with captcha enabled, the captcha SHALL auto-refresh. The login form SHALL send `X-Captcha-Id` and `X-Captcha-Answer` headers when captcha is enabled. The OAuth buttons SHALL be fetched from `GET /api/v1/auth/providers` on page load and only displayed when enabled providers exist. If the user is already authenticated, the page SHALL redirect to `/` using a `<Navigate>` component instead of calling `navigate()` during render. **The login page SHALL additionally include an optional email field for domain detection.** On email blur, it SHALL call `GET /api/v1/auth/check-domain?email=xxx` -- if the endpoint returns 404 (no identity App) or no match, the page SHALL behave as current (username+password + OAuth buttons). If a match is returned, the page SHALL display an SSO button. If forceSso=true, the page SHALL hide the password form.

#### Scenario: Successful login
- **WHEN** user enters valid credentials (and valid captcha if enabled) and submits the login form
- **THEN** the system SHALL call POST /api/v1/auth/login (with captcha headers if enabled), store the returned token pair, and redirect to /

#### Scenario: Failed login
- **WHEN** user enters invalid credentials and submits
- **THEN** the system SHALL display the error message from the API response and auto-refresh captcha if enabled

#### Scenario: Captcha displayed
- **WHEN** GET /api/v1/captcha returns enabled=true
- **THEN** the login page SHALL display the captcha image, an input field, and a refresh button below the password field

#### Scenario: Captcha hidden
- **WHEN** GET /api/v1/captcha returns enabled=false
- **THEN** the login page SHALL not display any captcha elements

#### Scenario: Registration link shown
- **WHEN** GET /api/v1/auth/registration-status returns registrationOpen=true
- **THEN** the login page SHALL show "还没有账号？立即注册" linking to /register

#### Scenario: Registration link hidden
- **WHEN** GET /api/v1/auth/registration-status returns registrationOpen=false
- **THEN** the login page SHALL not show the registration link

#### Scenario: 2FA required response
- **WHEN** login returns HTTP 202 with needsTwoFactor=true
- **THEN** the frontend SHALL redirect to /2fa with the twoFactorToken

#### Scenario: Already logged in redirect
- **WHEN** an authenticated user visits `/login`
- **THEN** the page SHALL render `<Navigate to="/" replace />`

#### Scenario: Display OAuth buttons
- **WHEN** the login page loads and GET /api/v1/auth/providers returns providers
- **THEN** the login page SHALL display OAuth buttons below the divider

#### Scenario: No OAuth providers available
- **WHEN** the login page loads and GET /api/v1/auth/providers returns an empty array
- **THEN** the login page SHALL not display the divider or any OAuth buttons

#### Scenario: Account locked error
- **WHEN** login returns HTTP 423
- **THEN** the login page SHALL display the lockout message with remaining minutes

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

#### Scenario: Click OAuth button
- **WHEN** user clicks the "GitHub 登录" button
- **THEN** the system SHALL call GET /api/v1/auth/oauth/github, receive an authURL, and redirect the browser to that URL via window.location.href

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
The frontend SHALL provide a user management page at `/users` accessible only to admin users, with a table listing all users and actions for create, edit, activate/deactivate, reset password, and delete. The table SHALL display a login method indicator column showing icons for local (password) and/or OAuth providers (GitHub/Google).

#### Scenario: Admin views user list
- **WHEN** admin navigates to /users
- **THEN** the page SHALL display a searchable, paginated table of users with columns: username, email, phone, role, login methods (icons), status, actions

#### Scenario: Login method icons
- **WHEN** a user has password set and a GitHub connection
- **THEN** the login methods column SHALL show a key icon (password) and the GitHub icon

#### Scenario: OAuth-only user
- **WHEN** a user has no password and only a Google connection
- **THEN** the login methods column SHALL show only the Google icon

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

### Requirement: Registration page
The frontend SHALL provide a registration page at `/register` rendered full-screen without DashboardLayout. The form SHALL include: username, email (optional), password, confirm password. On submit, call `POST /api/v1/auth/register`. On success, store the returned token pair and redirect to /. On error, display the error message. If registration is closed, the page SHALL show a notice "注册未开放" with a link back to login.

#### Scenario: Successful registration
- **WHEN** user fills the form with valid data and submits
- **THEN** the system SHALL call POST /api/v1/auth/register, store tokens, and redirect to /

#### Scenario: Registration closed
- **WHEN** user navigates to /register and registration_open=false
- **THEN** the page SHALL display "注册未开放" with a "返回登录" link

#### Scenario: Password policy violation
- **WHEN** user enters a password that violates the policy
- **THEN** the frontend SHALL display the violation messages from the API

#### Scenario: Password confirmation mismatch
- **WHEN** user enters mismatched password and confirm password
- **THEN** the frontend SHALL display a client-side validation error before submitting

### Requirement: 2FA verification page
The frontend SHALL provide a 2FA verification page at `/2fa` that accepts a 6-digit TOTP code or an 8-character backup code. The page SHALL receive the twoFactorToken from the login redirect. On submit, call `POST /api/v1/auth/2fa/login` with {twoFactorToken, code}. On success, store the token pair and redirect to /. The page SHALL include a toggle to switch between "验证码" and "恢复码" input modes.

#### Scenario: Successful TOTP verification
- **WHEN** user enters a valid 6-digit TOTP code
- **THEN** the system SHALL call POST /api/v1/auth/2fa/login, store tokens, and redirect to /

#### Scenario: Successful backup code verification
- **WHEN** user switches to backup code mode and enters a valid backup code
- **THEN** the system SHALL call POST /api/v1/auth/2fa/login, store tokens, and redirect to /

#### Scenario: Invalid code
- **WHEN** user enters an invalid code
- **THEN** the page SHALL display the error message

#### Scenario: Token expired
- **WHEN** the twoFactorToken has expired (>5 minutes)
- **THEN** the page SHALL display "验证已过期" and redirect to /login after 3 seconds

#### Scenario: Direct access without token
- **WHEN** user navigates to /2fa without a twoFactorToken
- **THEN** the page SHALL redirect to /login

### Requirement: 2FA setup in profile
The user profile/settings area SHALL include a "两步验证" section. If 2FA is not enabled, it SHALL show a "启用两步验证" button. On click: (1) call POST /api/v1/auth/2fa/setup, (2) display QR code using react-qr-code with the qrUri, (3) show a text input for the 6-digit confirmation code, (4) on confirm call POST /api/v1/auth/2fa/confirm, (5) display the 10 backup codes with a copy button and "我已保存恢复码" confirmation checkbox, (6) complete setup. If 2FA is enabled, show "已启用" badge and a "关闭两步验证" button that prompts for a TOTP code before calling DELETE /api/v1/auth/2fa.

#### Scenario: Setup flow
- **WHEN** user clicks "启用两步验证"
- **THEN** the system SHALL show QR code -> code input -> backup codes -> confirmation

#### Scenario: Disable flow
- **WHEN** user clicks "关闭两步验证" and enters a valid TOTP code
- **THEN** the system SHALL call DELETE /api/v1/auth/2fa and update the UI to show disabled state

### Requirement: Password expiry redirect
The API interceptor SHALL handle HTTP 409 responses with message "password expired" by redirecting to `/change-password` (or opening the change password dialog) and displaying a notification "密码已过期，请修改密码".

#### Scenario: Password expired intercept
- **WHEN** any API request returns 409 with message "password expired"
- **THEN** the frontend SHALL redirect to the change password flow and show a notification

#### Scenario: No redirect loop
- **WHEN** the change password API itself returns an error
- **THEN** the frontend SHALL display the error normally without triggering the 409 intercept

### Requirement: 2FA enforcement redirect
When the login response includes `requireTwoFactorSetup: true`, the auth store SHALL set a flag. The AuthGuard SHALL check this flag and redirect to the 2FA setup page. Only routes /settings, /2fa, /logout SHALL be accessible while the flag is set.

#### Scenario: Forced 2FA setup redirect
- **WHEN** user logs in and response includes requireTwoFactorSetup=true
- **THEN** the frontend SHALL redirect to the 2FA setup section and show a notification "请先启用两步验证"

#### Scenario: Navigation restricted during enforcement
- **WHEN** requireTwoFactorSetup flag is set and user navigates to /users
- **THEN** the frontend SHALL redirect back to the 2FA setup page
