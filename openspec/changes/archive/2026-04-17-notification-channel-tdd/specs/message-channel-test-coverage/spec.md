## ADDED Requirements

### Requirement: Repository layer tests
The message channel repository SHALL have unit tests covering all data access methods using an in-memory SQLite database.

#### Scenario: Create channel
- **WHEN** `Create` is called with a valid `MessageChannel`
- **THEN** the record SHALL be persisted with an auto-generated ID and `enabled=true`

#### Scenario: Find existing channel by ID
- **WHEN** `FindByID` is called with a valid ID
- **THEN** the stored `MessageChannel` SHALL be returned

#### Scenario: Find missing channel by ID
- **WHEN** `FindByID` is called with a non-existent ID
- **THEN** `gorm.ErrRecordNotFound` SHALL be returned

#### Scenario: List channels with pagination
- **WHEN** `List` is called with `page=1` and `pageSize=10` and 25 records exist
- **THEN** 10 items SHALL be returned and `total` SHALL equal 25

#### Scenario: List channels with keyword filter
- **WHEN** `List` is called with `keyword="smtp"`
- **THEN** only channels whose `name` contains "smtp" SHALL be returned

#### Scenario: Update channel
- **WHEN** `Update` is called with modified fields
- **THEN** the stored record SHALL reflect the changes

#### Scenario: Delete existing channel
- **WHEN** `Delete` is called with a valid ID
- **THEN** the record SHALL be removed and subsequent `FindByID` SHALL return `gorm.ErrRecordNotFound`

#### Scenario: Delete non-existent channel
- **WHEN** `Delete` is called with a non-existent ID
- **THEN** `gorm.ErrRecordNotFound` SHALL be returned

#### Scenario: Toggle enabled state
- **WHEN** `ToggleEnabled` is called on an enabled channel
- **THEN** the channel SHALL become disabled, and a second toggle SHALL re-enable it

#### Scenario: Mask sensitive config fields
- **WHEN** `MaskConfig` is called with a JSON string containing a `password` field
- **THEN** the returned JSON SHALL have `password` replaced by `"******"` and all other fields preserved

#### Scenario: Mask config with malformed JSON
- **WHEN** `MaskConfig` is called with invalid JSON
- **THEN** the original string SHALL be returned unchanged

### Requirement: Service layer tests
The message channel service SHALL have unit tests covering business logic, error handling, and integration with the repository.

#### Scenario: Create channel with valid type
- **WHEN** `Create` is called with a supported channel type
- **THEN** a `MessageChannel` SHALL be persisted and returned with `enabled=true`

#### Scenario: Create channel with invalid type
- **WHEN** `Create` is called with an unsupported channel type
- **THEN** an error SHALL be returned and no record SHALL be created

#### Scenario: Get channel masks password
- **WHEN** `Get` returns a channel whose config contains a password
- **THEN** the response config SHALL have the password masked as `"******"`

#### Scenario: Get non-existent channel
- **WHEN** `Get` is called with a non-existent ID
- **THEN** `ErrChannelNotFound` SHALL be returned

#### Scenario: Update channel preserves masked password
- **WHEN** `Update` receives a config where `password` equals `"******"`
- **THEN** the original password from the stored record SHALL be retained and other fields updated

#### Scenario: Update channel with invalid JSON
- **WHEN** `Update` receives a config that is not valid JSON
- **THEN** an error SHALL be returned and the stored record SHALL remain unchanged

#### Scenario: Delete existing channel
- **WHEN** `Delete` is called with a valid ID
- **THEN** the channel SHALL be removed

#### Scenario: Delete non-existent channel
- **WHEN** `Delete` is called with a non-existent ID
- **THEN** `ErrChannelNotFound` SHALL be returned

#### Scenario: Test channel connection success
- **WHEN** `TestChannel` is called and the driver reports success
- **THEN** no error SHALL be returned

#### Scenario: Test channel connection failure
- **WHEN** `TestChannel` is called and the driver reports failure
- **THEN** the driver error SHALL be returned

#### Scenario: Send test message success
- **WHEN** `SendTest` is called with recipient and content
- **THEN** the driver `Send` method SHALL be invoked with the constructed payload

#### Scenario: Send test message to missing channel
- **WHEN** `SendTest` is called with a non-existent channel ID
- **THEN** `ErrChannelNotFound` SHALL be returned

### Requirement: EmailDriver tests
The `EmailDriver` SHALL be refactored to accept an injectable SMTP client abstraction and have unit tests covering all send and test paths.

#### Scenario: Send plain text email
- **WHEN** `Send` is called with a payload containing only `To`, `Subject`, and `Body`
- **THEN** the SMTP client SHALL receive a message with `Content-Type: text/plain` and the body text

#### Scenario: Send HTML email
- **WHEN** `Send` is called with a payload containing `HTML` in addition to `Body`
- **THEN** the SMTP client SHALL receive a multipart MIME message containing both plain text and HTML parts

#### Scenario: Send via TLS
- **WHEN** `Send` is called with `secure=true`
- **THEN** a TLS connection SHALL be established and authentication SHALL succeed

#### Scenario: Test connection via STARTTLS
- **WHEN** `Test` is called with `secure=false` and the server advertises STARTTLS
- **THEN** STARTTLS SHALL be negotiated before authentication

#### Scenario: Test connection plain SMTP
- **WHEN** `Test` is called with `secure=false` and the server does not advertise STARTTLS
- **THEN** authentication SHALL proceed over the plain connection

#### Scenario: Test connection authentication failure
- **WHEN** `Test` is called and the SMTP server rejects credentials
- **THEN** an authentication error SHALL be returned

### Requirement: Handler layer tests
The channel handler HTTP endpoints SHALL have integration-style tests verifying request binding, status codes, and JSON responses.

#### Scenario: List channels
- **WHEN** `GET /api/v1/channels` is requested with valid query parameters
- **THEN** the response SHALL have status 200 and contain `items`, `total`, `page`, and `pageSize`

#### Scenario: Get single channel
- **WHEN** `GET /api/v1/channels/:id` requests an existing channel
- **THEN** the response SHALL have status 200 and the channel JSON

#### Scenario: Get missing channel
- **WHEN** `GET /api/v1/channels/:id` requests a non-existent channel
- **THEN** the response SHALL have status 404

#### Scenario: Create channel success
- **WHEN** `POST /api/v1/channels` is called with valid JSON
- **THEN** the response SHALL have status 200, return the new ID, and set audit fields

#### Scenario: Create channel with invalid type
- **WHEN** `POST /api/v1/channels` is called with an unsupported type
- **THEN** the response SHALL have status 400

#### Scenario: Update channel success
- **WHEN** `PUT /api/v1/channels/:id` is called with valid JSON
- **THEN** the response SHALL have status 200 and return the updated channel

#### Scenario: Update missing channel
- **WHEN** `PUT /api/v1/channels/:id` is called for a non-existent channel
- **THEN** the response SHALL have status 404

#### Scenario: Delete channel success
- **WHEN** `DELETE /api/v1/channels/:id` is called for an existing channel
- **THEN** the response SHALL have status 200

#### Scenario: Delete missing channel
- **WHEN** `DELETE /api/v1/channels/:id` is called for a non-existent channel
- **THEN** the response SHALL have status 404

#### Scenario: Toggle channel success
- **WHEN** `PUT /api/v1/channels/:id/toggle` is called
- **THEN** the response SHALL have status 200 and reflect the toggled state

#### Scenario: Test channel success
- **WHEN** `POST /api/v1/channels/:id/test` is called and the driver succeeds
- **THEN** the response SHALL have status 200 and `{"success": true}`

#### Scenario: Test channel failure
- **WHEN** `POST /api/v1/channels/:id/test` is called and the driver fails
- **THEN** the response SHALL have status 200 and `{"success": false, "error": "..."}`

#### Scenario: Send test message success
- **WHEN** `POST /api/v1/channels/:id/send-test` is called with valid body
- **THEN** the response SHALL have status 200 and `{"success": true}`
