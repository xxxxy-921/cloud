# Capability: license-test-coverage

## Purpose
License 模块测试覆盖能力 — 确保 crypto、service、handler 各层通过单元测试、集成测试和 fuzz 测试实现可验证的正确性。

## Requirements

### Requirement: Crypto and validation functions SHALL have deterministic unit and fuzz tests
All pure functions in `crypto.go` and validation helpers SHALL be verifiable through table-driven tests and fuzz tests without external dependencies.

#### Scenario: Canonicalize produces deterministic JSON
- **WHEN** any JSON-serializable value is canonicalized twice
- **THEN** both outputs SHALL be byte-for-byte identical

#### Scenario: License file encryption round-trip
- **WHEN** a plaintext payload is encrypted with a registration code and product name
- **THEN** decrypting the result with the same registration code SHALL yield the original plaintext

#### Scenario: Constraint schema validation rejects invalid structures
- **WHEN** a constraint schema with duplicate module keys or unsupported feature types is validated
- **THEN** the system SHALL return `ErrInvalidConstraintSchema`

#### Scenario: Fuzz testing finds no panics in validation
- **WHEN** random JSON bytes are unmarshaled into `ConstraintSchema` and passed to `validateConstraintSchema`
- **THEN** the function SHALL never panic

---

### Requirement: Service layer SHALL have SQLite in-memory integration tests
Business logic in `ProductService`, `PlanService`, `LicenseService`, and `LicenseeService` SHALL be tested against an in-memory SQLite database with real GORM transactions.

#### Scenario: Product creation generates initial key pair
- **WHEN** a product is created with a unique code
- **THEN** the database SHALL contain exactly one `ProductKey` record marked as current for that product

#### Scenario: License issuance enforces product publication status
- **GIVEN** a product with status `unpublished`
- **WHEN** a license is issued for that product
- **THEN** the system SHALL return `ErrProductNotPublished` and no license record SHALL be created

#### Scenario: License upgrade reuses registration code
- **GIVEN** an active license bound to a registration code
- **WHEN** the license is upgraded using the same registration code
- **THEN** the original license SHALL be revoked, a new license SHALL be created, and the registration code SHALL be bound to the new license

#### Scenario: Key rotation invalidates old key and issues new one
- **GIVEN** a product with a current key at version N
- **WHEN** the key is rotated
- **THEN** the old key SHALL no longer be current, a new key at version N+1 SHALL be current, and existing licenses SHALL remain valid but report as reissueable

#### Scenario: Status transition guards prevent illegal moves
- **GIVEN** a product in status `published`
- **WHEN** an attempt is made to transition it to `published`
- **THEN** the system SHALL return `ErrInvalidStatusTransition`

---

### Requirement: Time-dependent logic SHALL be testable with frozen timestamps
Functions that depend on the current wall-clock time SHALL accept the reference time as an explicit parameter so tests can be deterministic.

#### Scenario: deriveLifecycleStatus predicts pending state
- **WHEN** `deriveLifecycleStatus` is called with `validFrom` in the future and a frozen `now` timestamp
- **THEN** it SHALL return `pending`

#### Scenario: deriveLifecycleStatus detects expired state
- **WHEN** `deriveLifecycleStatus` is called with `validUntil` in the past relative to a frozen `now` timestamp
- **THEN** it SHALL return `expired`

---

### Requirement: Handlers SHALL map sentinel errors to correct HTTP status codes
HTTP handlers in the license module SHALL be verified for correct error-to-status-code mapping without requiring a full running server.

#### Scenario: Product not found returns 404
- **WHEN** a GET request targets a non-existent product ID
- **THEN** the handler SHALL respond with HTTP status 404

#### Scenario: Invalid constraint schema returns 400
- **WHEN** a product schema update receives malformed constraint JSON
- **THEN** the handler SHALL respond with HTTP status 400

#### Scenario: Bulk reissue of too many licenses returns 400
- **WHEN** a bulk reissue request contains more than 100 license IDs
- **THEN** the handler SHALL respond with HTTP status 400
