# Capability: ai-provider

## Purpose
Manages AI provider configurations including CRUD operations, connectivity testing, protocol auto-derivation, and encrypted API key storage.

## Requirements

### Requirement: Provider CRUD
The system SHALL support creating, reading, updating, and deleting AI providers. Each provider SHALL have: name, type (`openai` / `anthropic` / `ollama`), protocol (`openai` / `anthropic`), base_url, api_key (encrypted), status (`active` / `inactive` / `error`), and health_checked_at timestamp.

#### Scenario: Create a provider
- **WHEN** admin submits a valid provider form with name, type, base_url, and api_key
- **THEN** the system creates the provider with api_key encrypted via AES-256-GCM, status set to `inactive`, and protocol auto-derived from type

#### Scenario: List providers
- **WHEN** admin requests the provider list
- **THEN** the system returns paginated providers with api_key masked (show only last 4 chars), including model count per provider

#### Scenario: Update a provider
- **WHEN** admin updates a provider's base_url or api_key
- **THEN** the system saves the changes, re-encrypting api_key if changed, and resets status to `inactive`

#### Scenario: Delete a provider
- **WHEN** admin deletes a provider that has no active models
- **THEN** the system soft-deletes the provider

#### Scenario: Delete a provider with active models
- **WHEN** admin deletes a provider that has active models
- **THEN** the system rejects the deletion with error message

### Requirement: Provider protocol auto-derivation
The system SHALL automatically set the `protocol` field based on the provider `type` when creating or updating a provider.

#### Scenario: OpenAI type provider
- **WHEN** a provider is created with type `openai` or `ollama`
- **THEN** the protocol is set to `openai`

#### Scenario: Anthropic type provider
- **WHEN** a provider is created with type `anthropic`
- **THEN** the protocol is set to `anthropic`

### Requirement: Provider connectivity test
The system SHALL provide an API to test a provider's connectivity and API key validity.

#### Scenario: Test OpenAI-compatible provider
- **WHEN** admin triggers a connectivity test for a provider with protocol `openai`
- **THEN** the system calls `GET {base_url}/v1/models` with the decrypted api_key and returns success/failure with error detail

#### Scenario: Test Anthropic provider
- **WHEN** admin triggers a connectivity test for a provider with protocol `anthropic`
- **THEN** the system sends a minimal `POST {base_url}/v1/messages` request (max_tokens=1) and returns success/failure

#### Scenario: Successful connectivity test
- **WHEN** a connectivity test succeeds
- **THEN** the provider status is updated to `active` and health_checked_at is set to current time

#### Scenario: Failed connectivity test
- **WHEN** a connectivity test fails
- **THEN** the provider status is updated to `error` and the error message is returned to the caller

### Requirement: API Key encrypted storage
The system SHALL encrypt provider API keys at rest using AES-256-GCM with a key derived from `secret_key` in metis.yaml. The shared encryption package SHALL be at `internal/pkg/crypto/`.

#### Scenario: Store API key
- **WHEN** a provider is created or updated with an api_key
- **THEN** the api_key is encrypted with `sha256(config.SecretKey)` as AES key before storage

#### Scenario: Read API key for internal use
- **WHEN** the LLM client needs the api_key to call a provider's API
- **THEN** the system decrypts the api_key from the database and passes it to the client

#### Scenario: API response never exposes raw API key
- **WHEN** provider data is returned in any API response
- **THEN** the api_key field shows only a masked form (e.g., `sk-****xxxx`)
