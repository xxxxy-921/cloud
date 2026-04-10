## ADDED Requirements

### Requirement: IdentitySource model（App 内自包含）
The identity App SHALL define an `IdentitySource` model in `internal/app/identity/model.go` with BaseModel (ID, timestamps, soft delete), Name (string, display name), Type (string, "oidc" or "ldap"), Enabled (bool, default false), Domains (string, comma-separated email domains), ForceSso (bool, default false), DefaultRoleID (uint, FK to roles), ConflictStrategy (string, "link" or "fail", default "fail"), Config (text, JSON serialized protocol-specific configuration), SortOrder (int, default 0). The model SHALL be registered via the App's `Models()` method for AutoMigrate.

#### Scenario: App registers model for migration
- **WHEN** the identity App's `Models()` is called during startup
- **THEN** it SHALL return `[]any{&IdentitySource{}}` and main.go SHALL AutoMigrate the table

#### Scenario: Create OIDC identity source
- **WHEN** an admin creates an identity source with Name "Okta SSO", Type "oidc", Domains "acme.com"
- **THEN** the system SHALL store the record with Enabled=false, ForceSso=false, ConflictStrategy="fail"

#### Scenario: Domain uniqueness across sources
- **WHEN** an identity source with Domain "acme.com" already exists and another source attempts to bind the same domain
- **THEN** the system SHALL return a validation error "domain already bound to another identity source"

#### Scenario: Type validation
- **WHEN** an admin attempts to create an identity source with Type "saml"
- **THEN** the system SHALL return a validation error "unsupported identity source type"

### Requirement: OIDC configuration structure
The OIDC Config JSON SHALL contain IssuerURL (string), ClientID (string), ClientSecret (string, AES-256-GCM encrypted), Scopes ([]string, default ["openid","profile","email"]), UsePKCE (bool, default true), and CallbackURL (string). The system SHALL validate IssuerURL by fetching `/.well-known/openid-configuration`.

#### Scenario: Valid OIDC config
- **WHEN** an admin saves OIDC config with IssuerURL "https://accounts.google.com"
- **THEN** the system SHALL fetch the OpenID discovery document and store the configuration

#### Scenario: Invalid OIDC issuer
- **WHEN** an admin saves OIDC config with an IssuerURL that returns no valid discovery document
- **THEN** the system SHALL return a validation error "failed to discover OIDC provider"

### Requirement: LDAP configuration structure
The LDAP Config JSON SHALL contain ServerURL (string), BindDN (string), BindPassword (string, AES-256-GCM encrypted), SearchBase (string), UserFilter (string, e.g. "(uid={{username}})"), UseTLS (bool), SkipVerify (bool, default false), and AttributeMapping (JSON object: username, email, display_name, avatar).

#### Scenario: Valid LDAP config
- **WHEN** an admin saves LDAP config with ServerURL "ldaps://ldap.corp.com:636"
- **THEN** the system SHALL store the configuration with BindPassword encrypted

#### Scenario: Default attribute mapping
- **WHEN** an admin creates an LDAP source without specifying AttributeMapping
- **THEN** the system SHALL use defaults: username="uid", email="mail", display_name="cn"

### Requirement: Sensitive field encryption（App 内自包含）
The identity App SHALL implement AES-256-GCM encryption in `internal/app/identity/crypto.go`. Encryption key SHALL be read from `ENCRYPTION_KEY` env var. If not set, SHALL auto-generate and store in SystemConfig table as `security.encryption_key`.

#### Scenario: Encrypt on save
- **WHEN** an admin saves an identity source with ClientSecret "my-secret"
- **THEN** the App SHALL AES-256-GCM encrypt the value before persisting

#### Scenario: Mask in API response
- **WHEN** GET /api/v1/identity-sources returns identity sources
- **THEN** sensitive fields SHALL show as "••••••"

### Requirement: Identity source CRUD API（App Routes）
The identity App SHALL register admin-only REST endpoints via its `Routes()` method under `/identity-sources`.

#### Scenario: List identity sources
- **WHEN** admin GET /api/v1/identity-sources
- **THEN** the system SHALL return all identity sources with sensitive fields masked

#### Scenario: Create identity source
- **WHEN** admin POST /api/v1/identity-sources with `{name, type, domains, config, defaultRoleId}`
- **THEN** the system SHALL validate, encrypt sensitive fields, and create the record

#### Scenario: Update identity source
- **WHEN** admin PUT /api/v1/identity-sources/:id with updated fields
- **THEN** the system SHALL update. If sensitive fields are "••••••", preserve existing values.

#### Scenario: Delete identity source
- **WHEN** admin DELETE /api/v1/identity-sources/:id
- **THEN** the system SHALL soft-delete the identity source

#### Scenario: Toggle identity source
- **WHEN** admin PATCH /api/v1/identity-sources/:id/toggle
- **THEN** the system SHALL flip Enabled and return the updated record

### Requirement: Test connection endpoint
The App SHALL provide `POST /api/v1/identity-sources/:id/test` (admin-only) for connectivity testing.

#### Scenario: Test OIDC connection
- **WHEN** admin tests an OIDC source with valid IssuerURL
- **THEN** SHALL return `{success: true, message: "OIDC discovery successful"}`

#### Scenario: Test LDAP connection
- **WHEN** admin tests an LDAP source with valid credentials
- **THEN** SHALL attempt LDAP bind and return `{success: true, message: "LDAP bind successful"}`

#### Scenario: Test failure
- **WHEN** connection test fails
- **THEN** SHALL return `{success: false, message: "<error details>"}`

### Requirement: Casbin policies via App Seed
The identity App SHALL register Casbin policies for admin access to `/api/v1/identity-sources/*` in its `Seed()` method.

#### Scenario: Admin can manage identity sources
- **WHEN** an admin accesses GET /api/v1/identity-sources
- **THEN** Casbin SHALL allow the request

#### Scenario: Non-admin denied
- **WHEN** a user with role "user" accesses GET /api/v1/identity-sources
- **THEN** Casbin SHALL deny with 403

### Requirement: Menu entry via App Seed
The identity App SHALL seed an "身份源管理" menu entry under "系统管理" in its `Seed()` method, with permission `system:identity-source:list`.

#### Scenario: App seeds menu
- **WHEN** the identity App's `Seed()` runs on startup
- **THEN** an "身份源管理" menu SHALL be created under the "系统管理" directory if not exists
