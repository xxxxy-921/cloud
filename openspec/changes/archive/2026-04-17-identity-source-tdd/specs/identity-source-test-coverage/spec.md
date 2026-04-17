## ADDED Requirements

### Requirement: Crypto utility tests
The AES-256-GCM encryption utilities in `internal/pkg/token` SHALL have unit tests verifying round-trip encryption, key auto-generation, and error handling.

#### Scenario: Encrypt and decrypt round-trip
- **WHEN** `Encrypt` is called with a plaintext string followed by `Decrypt` on the resulting ciphertext
- **THEN** the decrypted plaintext SHALL equal the original input

#### Scenario: Auto-generate encryption key
- **WHEN** `Encrypt` is called on a fresh in-memory database with no pre-existing `security.encryption_key`
- **THEN** a `SystemConfig` record with key `security.encryption_key` SHALL be created and the encryption SHALL succeed

#### Scenario: Decrypt invalid hex
- **WHEN** `Decrypt` is called with a string that is not valid hexadecimal
- **THEN** an error SHALL be returned

#### Scenario: Decrypt tampered ciphertext
- **WHEN** `Decrypt` is called with a hex string whose ciphertext has been altered
- **THEN** a GCM authentication error SHALL be returned

### Requirement: Model layer tests
The `IdentitySource` model and its related structs SHALL have unit tests for response serialization and defaults.

#### Scenario: OIDC config secrets are masked in response
- **WHEN** `ToResponse` is called on an IdentitySource of type "oidc" whose Config contains a ClientSecret
- **THEN** the response Config SHALL have the ClientSecret replaced by `••••••`

#### Scenario: LDAP config secrets are masked in response
- **WHEN** `ToResponse` is called on an IdentitySource of type "ldap" whose Config contains a BindPassword
- **THEN** the response Config SHALL have the BindPassword replaced by `••••••`

#### Scenario: Unknown type config is passed through
- **WHEN** `ToResponse` is called on an IdentitySource with an unsupported type
- **THEN** the Config SHALL be returned as-is without modification

#### Scenario: Default LDAP attribute mapping
- **WHEN** `DefaultLDAPAttributeMapping` is invoked
- **THEN** it SHALL return a map containing `username` mapped to `uid`, `email` to `mail`, and `display_name` to `cn`

### Requirement: Repository layer tests
The `IdentitySourceRepo` SHALL have unit tests for all data access and domain logic methods.

#### Scenario: List ordered by sort order
- **WHEN** `List` is called with multiple records having different `sort_order` values
- **THEN** the results SHALL be ordered by `sort_order ASC, id ASC`

#### Scenario: Find by domain exact match
- **WHEN** `FindByDomain` is called with a domain that exactly matches an enabled source's domain
- **THEN** the matching source SHALL be returned

#### Scenario: Find by domain case insensitive
- **WHEN** `FindByDomain` is called with "EXAMPLE.COM" and the stored domain is "example.com"
- **THEN** the matching source SHALL be returned

#### Scenario: Find by domain with whitespace trimming
- **WHEN** `FindByDomain` is called with " example.com " and the stored domains include "example.com"
- **THEN** the matching source SHALL be returned

#### Scenario: Find by domain no match
- **WHEN** `FindByDomain` is called with a domain not bound to any enabled source
- **THEN** `gorm.ErrRecordNotFound` SHALL be returned

#### Scenario: Domain conflict detected
- **WHEN** `CheckDomainConflict` is called with a domain already bound to another source
- **THEN** `ErrDomainConflict` SHALL be returned

#### Scenario: Domain conflict self-update allowed
- **WHEN** `CheckDomainConflict` is called with the source's own existing domain and its own ID as excludeID
- **THEN** no error SHALL be returned

#### Scenario: Domain conflict empty domains pass
- **WHEN** `CheckDomainConflict` is called with an empty domains string
- **THEN** no error SHALL be returned

#### Scenario: Create, update, delete, and toggle
- **WHEN** `Create`, `Update`, `Delete`, and `Toggle` are exercised
- **THEN** each operation SHALL persist or remove the record as expected and `Toggle` SHALL invert the `enabled` state

### Requirement: Service layer tests
The `IdentitySourceService` SHALL be refactored for testability and have unit tests covering business logic, encryption, and external authentication integration.

#### Scenario: Create OIDC source encrypts client secret
- **WHEN** `Create` is called with an OIDC config containing a ClientSecret
- **THEN** the stored Config SHALL contain an encrypted ClientSecret (hex ciphertext, not plaintext)

#### Scenario: Create LDAP source fills defaults and encrypts password
- **WHEN** `Create` is called with an LDAP config missing AttributeMapping and BindPassword
- **THEN** the stored Config SHALL contain the default attribute mapping and the BindPassword SHALL be encrypted if provided

#### Scenario: Create unsupported type rejected
- **WHEN** `Create` is called with Type "saml"
- **THEN** `ErrUnsupportedType` SHALL be returned

#### Scenario: Create domain conflict rejected
- **WHEN** `Create` is called with a domain already bound to another source
- **THEN** `ErrDomainConflict` SHALL be returned

#### Scenario: Update preserves masked OIDC secret
- **WHEN** `Update` receives a config where ClientSecret equals `••••••`
- **THEN** the original encrypted ClientSecret SHALL be retained and other fields updated

#### Scenario: Update preserves masked LDAP password
- **WHEN** `Update` receives a config where BindPassword equals `••••••`
- **THEN** the original encrypted BindPassword SHALL be retained and other fields updated

#### Scenario: Update not found
- **WHEN** `Update` is called with a non-existent ID
- **THEN** `ErrSourceNotFound` SHALL be returned

#### Scenario: Delete and toggle not found
- **WHEN** `Delete` or `Toggle` is called with a non-existent ID
- **THEN** `ErrSourceNotFound` SHALL be returned

#### Scenario: Test connection OIDC success
- **WHEN** `TestConnection` is called on an OIDC source and the injected discovery function succeeds
- **THEN** it SHALL return `(true, "OIDC discovery successful")`

#### Scenario: Test connection OIDC failure
- **WHEN** `TestConnection` is called on an OIDC source and the injected discovery function fails
- **THEN** it SHALL return `(false, "OIDC discovery failed: ...")`

#### Scenario: Test connection LDAP success
- **WHEN** `TestConnection` is called on an LDAP source and the injected test function succeeds
- **THEN** it SHALL return `(true, "LDAP bind successful")`

#### Scenario: Test connection not found
- **WHEN** `TestConnection` is called with a non-existent ID
- **THEN** it SHALL return `(false, "identity source not found")`

#### Scenario: Authenticate by password success
- **WHEN** `AuthenticateByPassword` is called with valid credentials and the injected LDAP auth function returns a result
- **THEN** an `ExternalAuthResult` SHALL be returned with Provider, ExternalID, Email, DisplayName, Username, DefaultRoleID, and ConflictStrategy populated

#### Scenario: Authenticate by password all sources fail
- **WHEN** all enabled LDAP sources' auth attempts fail
- **THEN** an error containing `ldap_auth_failed` SHALL be returned

#### Scenario: Check domain and forced SSO
- **WHEN** `CheckDomain` and `IsForcedSSO` are called with matching and non-matching emails
- **THEN** they SHALL return the expected source metadata or boolean values

#### Scenario: Extract domain from email
- **WHEN** `ExtractDomain` is called with various email strings
- **THEN** it SHALL return the lower-case domain part or an empty string for invalid input

### Requirement: Handler layer tests
The identity source handler HTTP endpoints SHALL have integration-style tests verifying request binding, status codes, and JSON responses.

#### Scenario: List identity sources
- **WHEN** `GET /api/v1/identity-sources` is requested
- **THEN** the response SHALL have status 200 and contain an array of sources

#### Scenario: Create identity source success
- **WHEN** `POST /api/v1/identity-sources` is called with valid JSON
- **THEN** the response SHALL have status 200, return the created source, and set audit fields

#### Scenario: Create identity source unsupported type
- **WHEN** `POST /api/v1/identity-sources` is called with Type "saml"
- **THEN** the response SHALL have status 400

#### Scenario: Create identity source domain conflict
- **WHEN** `POST /api/v1/identity-sources` is called with a conflicting domain
- **THEN** the response SHALL have status 409

#### Scenario: Update identity source success
- **WHEN** `PUT /api/v1/identity-sources/:id` is called with valid JSON
- **THEN** the response SHALL have status 200 and return the updated source

#### Scenario: Update identity source not found
- **WHEN** `PUT /api/v1/identity-sources/:id` is called for a non-existent source
- **THEN** the response SHALL have status 404

#### Scenario: Delete identity source success
- **WHEN** `DELETE /api/v1/identity-sources/:id` is called for an existing source
- **THEN** the response SHALL have status 200

#### Scenario: Delete identity source not found
- **WHEN** `DELETE /api/v1/identity-sources/:id` is called for a non-existent source
- **THEN** the response SHALL have status 404

#### Scenario: Toggle identity source
- **WHEN** `PUT /api/v1/identity-sources/:id/toggle` is called
- **THEN** the response SHALL have status 200 and reflect the toggled enabled state

#### Scenario: Test connection endpoint
- **WHEN** `POST /api/v1/identity-sources/:id/test` is called
- **THEN** the response SHALL have status 200 and a body containing `success` and `message` fields
