## MODIFIED Requirements

### Requirement: Login page
The frontend SHALL provide a login page at `/login` with username and password fields, rendered full-screen without DashboardLayout. Below the password field, a captcha area SHALL be conditionally rendered: on page load, call `GET /api/v1/captcha` — if `enabled=true`, display the captcha image, a text input for the answer, and a refresh button; if `enabled=false`, hide the captcha area. Below the login form, a divider "或" SHALL separate the local login form from dynamically rendered OAuth provider buttons. Below the OAuth section (or below the form if no OAuth), a registration link SHALL be conditionally shown: call `GET /api/v1/auth/registration-status` — if `registrationOpen=true`, display "还没有账号？立即注册" linking to `/register`. On login failure with captcha enabled, the captcha SHALL auto-refresh. The login form SHALL send `X-Captcha-Id` and `X-Captcha-Answer` headers when captcha is enabled.

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

#### Scenario: Account locked error
- **WHEN** login returns HTTP 423
- **THEN** the login page SHALL display the lockout message with remaining minutes

## ADDED Requirements

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
- **THEN** the system SHALL show QR code → code input → backup codes → confirmation

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
