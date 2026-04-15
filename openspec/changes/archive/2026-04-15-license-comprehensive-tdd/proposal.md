## Why

The license module currently has zero automated tests across 13 source files, despite containing security-critical logic (Ed25519 signing, AES-GCM encryption) and complex business rules (license lifecycle state machine, key rotation, bulk reissue). This creates unacceptable regression risk and makes refactoring impossible. We need comprehensive test coverage using strict TDD practices to harden the module before any further feature work.

## What Changes

- Introduce table-driven unit tests and fuzz tests for `crypto.go` and validation helpers.
- Build SQLite in-memory integration tests for `ProductService`, `PlanService`, `LicenseService`, and `LicenseeService`.
- Add `gin.Test` handler-level tests for critical API error mappings and happy paths.
- Refactor hidden time dependencies (e.g. `time.Now()` inside `deriveLifecycleStatus`) to make code testable without monkey-patching.
- Establish a lightweight test fixture pattern for license models so future tests are cheap to write.

## Capabilities

### New Capabilities
- `license-test-coverage`: Comprehensive TDD test suite for the license module, spanning pure-function unit tests, fuzz tests, and SQLite in-memory service integration tests.

### Modified Capabilities
<!-- No spec-level behavior changes; existing license capabilities remain functionally identical. -->

## Impact

- Affected packages: `internal/app/license/*`
- Test runtime: local only (SQLite in-memory), no CI or Docker dependencies
- No API contract changes; minor internal refactor (clock injection) to improve testability
