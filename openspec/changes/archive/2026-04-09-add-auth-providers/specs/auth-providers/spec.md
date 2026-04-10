## ADDED Requirements

### Requirement: AuthProvider model
The system SHALL store OAuth provider configurations in an `auth_providers` table with BaseModel (ID, timestamps, soft delete), ProviderKey (string, unique, e.g. "github"/"google"), DisplayName (string), Enabled (bool, default false), ClientID (string), ClientSecret (string), Scopes (string), CallbackURL (string), and SortOrder (int, default 0).

#### Scenario: Create GitHub provider configuration
- **WHEN** an admin creates an auth provider with ProviderKey "github", ClientID "abc", ClientSecret "xyz", Scopes "user:email"
- **THEN** the system SHALL store the configuration with Enabled=false and SortOrder=0

#### Scenario: ProviderKey uniqueness
- **WHEN** an auth provider with ProviderKey "github" already exists and another is created with the same key
- **THEN** the system SHALL return a unique constraint violation error

### Requirement: AuthProvider response hiding secrets
The AuthProvider model SHALL expose a `ToResponse()` method that returns all fields except ClientSecret (replaced with a masked indicator like "••••••" if non-empty, empty string if empty).

#### Scenario: API returns masked secret
- **WHEN** GET /api/v1/admin/auth-providers returns a provider with ClientSecret set
- **THEN** the response SHALL show ClientSecret as "••••••" instead of the actual value

### Requirement: UserConnection model
The system SHALL store user-provider bindings in a `user_connections` table with BaseModel, UserID (uint, FK to users with cascade delete), Provider (string, e.g. "github"/"google"), ExternalID (string, provider's unique user identifier), ExternalName (string, provider's display name/username), ExternalEmail (string), and AvatarURL (string). The table SHALL have unique constraints on (Provider, ExternalID) and (UserID, Provider).

#### Scenario: Create GitHub connection for user
- **WHEN** user ID 1 authenticates via GitHub with external ID "12345" and username "octocat"
- **THEN** the system SHALL create a UserConnection with UserID=1, Provider="github", ExternalID="12345", ExternalName="octocat"

#### Scenario: Duplicate external identity
- **WHEN** a connection with Provider="github" and ExternalID="12345" already exists for user ID 1, and user ID 2 attempts to bind the same GitHub identity
- **THEN** the system SHALL return a unique constraint violation error

#### Scenario: One provider per user
- **WHEN** user ID 1 already has a connection with Provider="github" and attempts to bind another GitHub account
- **THEN** the system SHALL return a unique constraint violation error

#### Scenario: Cascade delete
- **WHEN** a user is deleted (soft delete)
- **THEN** all their UserConnection records SHALL remain (soft delete via BaseModel follows user lifecycle)

### Requirement: OAuthProvider interface
The system SHALL define an OAuthProvider interface with methods: `GetAuthURL(state string) string` (returns the OAuth authorization URL) and `ExchangeCode(ctx context.Context, code string) (*OAuthUserInfo, error)` (exchanges authorization code for user information). OAuthUserInfo SHALL contain ID (string), Name (string), Email (string), and AvatarURL (string).

#### Scenario: GitHub provider implementation
- **WHEN** the system creates a GitHub OAuthProvider with a valid ClientID, ClientSecret, and Scopes
- **THEN** `GetAuthURL` SHALL return a GitHub authorization URL and `ExchangeCode` SHALL call GitHub's token endpoint and user API to return OAuthUserInfo

#### Scenario: Google provider implementation
- **WHEN** the system creates a Google OAuthProvider with a valid ClientID, ClientSecret, and Scopes
- **THEN** `GetAuthURL` SHALL return a Google authorization URL and `ExchangeCode` SHALL call Google's token endpoint and userinfo API to return OAuthUserInfo

### Requirement: OAuth state management
The system SHALL maintain an in-memory map of OAuth state tokens with metadata (provider, createdAt). State tokens SHALL expire after 10 minutes. A background goroutine SHALL periodically clean up expired entries.

#### Scenario: Generate state token
- **WHEN** a user initiates OAuth login for provider "github"
- **THEN** the system SHALL generate a cryptographically random state token, store it with provider metadata and timestamp, and return it

#### Scenario: Validate state token
- **WHEN** an OAuth callback includes a valid, non-expired state token
- **THEN** the system SHALL return the associated provider metadata and delete the token from the map

#### Scenario: Expired state token
- **WHEN** an OAuth callback includes a state token that was created more than 10 minutes ago
- **THEN** the system SHALL return an error "state expired"

#### Scenario: Invalid state token
- **WHEN** an OAuth callback includes a state token that does not exist in the map
- **THEN** the system SHALL return an error "invalid state"

### Requirement: List enabled providers (public)
The system SHALL provide `GET /api/v1/auth/providers` (no authentication required) returning a list of enabled auth providers with ProviderKey, DisplayName, and SortOrder only (no secrets, no ClientID).

#### Scenario: List enabled providers
- **WHEN** GET /api/v1/auth/providers and two providers (GitHub enabled, Google disabled) exist
- **THEN** the system SHALL return only the GitHub provider with fields {providerKey, displayName, sortOrder}

#### Scenario: No providers enabled
- **WHEN** GET /api/v1/auth/providers and no providers are enabled
- **THEN** the system SHALL return an empty array

### Requirement: Initiate OAuth authorization
The system SHALL provide `GET /api/v1/auth/oauth/:provider` (no authentication required) that generates an OAuth authorization URL for the specified provider and returns it as JSON.

#### Scenario: Initiate GitHub OAuth
- **WHEN** GET /api/v1/auth/oauth/github and the GitHub provider is enabled with valid configuration
- **THEN** the system SHALL generate a state token, build the GitHub authorization URL with ClientID, scopes, redirect_uri (CallbackURL), and state, and return `{code: 0, data: {authURL: "https://github.com/login/oauth/authorize?...", state: "xxx"}}`

#### Scenario: Provider not found or disabled
- **WHEN** GET /api/v1/auth/oauth/wechat and the wechat provider does not exist or is disabled
- **THEN** the system SHALL return 400 with message "provider not available"

### Requirement: OAuth callback processing
The system SHALL provide `POST /api/v1/auth/oauth/callback` (no authentication required) accepting JSON body `{provider, code, state}` that completes the OAuth flow and returns a TokenPair.

#### Scenario: First-time OAuth login (new user)
- **WHEN** POST /api/v1/auth/oauth/callback with a valid code and state for a GitHub user with ExternalID "12345" who has no existing connection
- **THEN** the system SHALL exchange the code for user info, create a new User with auto-generated username "github_12345" and empty password, assign default role "user", create a UserConnection record, and return a TokenPair

#### Scenario: Returning OAuth login (existing connection)
- **WHEN** POST /api/v1/auth/oauth/callback with a valid code for a GitHub user who already has a UserConnection
- **THEN** the system SHALL find the existing user via the connection, update ExternalName/ExternalEmail/AvatarURL if changed, and return a TokenPair

#### Scenario: Email conflict on first OAuth login
- **WHEN** POST /api/v1/auth/oauth/callback for a new GitHub user whose email matches an existing local user
- **THEN** the system SHALL return 409 with message "email already registered, please login with password and bind this account in settings"

#### Scenario: OAuth user account is disabled
- **WHEN** POST /api/v1/auth/oauth/callback for a GitHub user whose linked local account has IsActive=false
- **THEN** the system SHALL return 401 with message "account disabled"

#### Scenario: Invalid state
- **WHEN** POST /api/v1/auth/oauth/callback with an invalid or expired state token
- **THEN** the system SHALL return 400 with message "invalid or expired state"

### Requirement: List user connections (authenticated)
The system SHALL provide `GET /api/v1/auth/connections` (requires authentication) returning the current user's bound external accounts.

#### Scenario: User has connections
- **WHEN** authenticated user GET /api/v1/auth/connections and has GitHub and Google connections
- **THEN** the system SHALL return `{code: 0, data: [{provider, externalName, externalEmail, avatarURL, createdAt}, ...]}`

#### Scenario: User has no connections
- **WHEN** authenticated user GET /api/v1/auth/connections and has no external accounts
- **THEN** the system SHALL return `{code: 0, data: []}`

### Requirement: Bind external account (authenticated)
The system SHALL provide `POST /api/v1/auth/connections/:provider` (requires authentication) that initiates an OAuth flow to bind a new external account to the current user. The endpoint SHALL return an authURL for the frontend to redirect to.

#### Scenario: Initiate binding
- **WHEN** authenticated user POST /api/v1/auth/connections/github
- **THEN** the system SHALL generate a state token (with bind mode metadata including userID), and return `{code: 0, data: {authURL: "...", state: "..."}}`

#### Scenario: Already bound
- **WHEN** authenticated user POST /api/v1/auth/connections/github and already has a GitHub connection
- **THEN** the system SHALL return 400 with message "already bound to this provider"

### Requirement: Bind OAuth callback processing
The system SHALL provide `POST /api/v1/auth/connections/callback` (requires authentication) accepting `{provider, code, state}` that completes the bind flow.

#### Scenario: Successful bind
- **WHEN** authenticated user POST /api/v1/auth/connections/callback with valid code and bind-mode state
- **THEN** the system SHALL create a UserConnection linking the current user to the external identity and return `{code: 0, message: "ok"}`

#### Scenario: External identity already bound to another user
- **WHEN** authenticated user attempts to bind a GitHub identity that is already bound to a different user
- **THEN** the system SHALL return 409 with message "this account is already bound to another user"

### Requirement: Unbind external account (authenticated)
The system SHALL provide `DELETE /api/v1/auth/connections/:provider` (requires authentication) to remove an external account binding.

#### Scenario: Successful unbind
- **WHEN** authenticated user DELETE /api/v1/auth/connections/github and has a GitHub connection
- **THEN** the system SHALL delete the UserConnection record and return `{code: 0, message: "ok"}`

#### Scenario: Cannot unbind last login method
- **WHEN** authenticated user DELETE /api/v1/auth/connections/github and has no password set and GitHub is their only connection
- **THEN** the system SHALL return 400 with message "cannot unbind last login method, please set a password first"

#### Scenario: Connection not found
- **WHEN** authenticated user DELETE /api/v1/auth/connections/google and has no Google connection
- **THEN** the system SHALL return 404 with message "connection not found"

### Requirement: Admin auth provider management
The system SHALL provide admin-only endpoints to manage auth provider configurations.

#### Scenario: List all providers (admin)
- **WHEN** admin GET /api/v1/admin/auth-providers
- **THEN** the system SHALL return all auth providers (including disabled ones) with ClientSecret masked

#### Scenario: Update provider configuration (admin)
- **WHEN** admin PUT /api/v1/admin/auth-providers/:key with `{displayName, clientId, clientSecret, scopes, callbackURL, sortOrder}`
- **THEN** the system SHALL update the specified fields (if clientSecret is "••••••" or empty, preserve the existing value)

#### Scenario: Toggle provider (admin)
- **WHEN** admin PATCH /api/v1/admin/auth-providers/:key/toggle
- **THEN** the system SHALL flip the Enabled flag and return the updated provider

#### Scenario: Provider not found (admin)
- **WHEN** admin PUT /api/v1/admin/auth-providers/nonexistent
- **THEN** the system SHALL return 404 with message "provider not found"

### Requirement: Auth provider seed data
The system SHALL seed default auth provider records for "github" and "google" with Enabled=false and empty credentials on first startup (via the seed mechanism).

#### Scenario: Seed creates default providers
- **WHEN** the application starts and no auth providers exist in the database
- **THEN** the system SHALL create records for "github" (DisplayName="GitHub", SortOrder=1) and "google" (DisplayName="Google", SortOrder=2) with Enabled=false

#### Scenario: Seed is idempotent
- **WHEN** the application starts and auth providers already exist
- **THEN** the system SHALL not duplicate existing records
