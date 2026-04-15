## Context

The license module (`internal/app/license/`) implements a full license lifecycle management system including Ed25519 key pairs, AES-GCM encrypted license files, product/plan constraint schemas, and license state machines (pending → active → suspended/expired/revoked). Despite its complexity and security sensitivity, the module currently has zero automated tests. The only test file in the entire backend is `internal/app/ai/data_stream_test.go`, which uses the standard `testing` package.

## Goals / Non-Goals

**Goals:**
- Achieve comprehensive test coverage for the license module using strict red-green-refactor TDD.
- Establish a fast, deterministic, local test suite that requires no Docker or external services.
- Make hidden dependencies (time, randomness) explicit so they can be controlled in tests.
- Document the testing pattern so future license features are trivial to TDD.

**Non-Goals:**
- Changing license business rules or API contracts.
- Introducing BDD frameworks (Ginkgo/Gomega) or mocking libraries (gomock/testify mock).
- CI pipeline integration (out of scope for this change).
- Frontend testing.

## Decisions

### 1. Standard `testing` + table-driven tests (no BDD frameworks)
**Rationale:** The project already uses the standard `testing` package (`internal/app/ai/data_stream_test.go`). Adding Ginkgo/Gomega would fragment the codebase and increase cognitive load. Table-driven tests are idiomatic in Go and sufficient for all license test needs.

### 2. SQLite in-memory integration tests for service layer
**Rationale:** The license service layer is dominated by GORM transactions, multi-table joins, and state-dependent queries. Mocking the repository layer would produce tests that pass while the real SQL is broken. Because Metis uses SQLite by default, running `gorm.Open(sqlite.Open("file::memory:?cache=shared"))` gives us 100% dialect fidelity with sub-millisecond setup time.

### 3. Direct struct construction for service tests; mini-DI only for smoke tests
**Rationale:** `samber/do` is great for runtime wiring but adds noise in unit tests. We will construct `ProductService{productRepo: &ProductRepo{db: db}, ...}` directly in most tests. One optional smoke test can verify that `do.Provide` registration is not broken.

### 4. Parameter injection for `time.Now()`
**Rationale:** Functions like `deriveLifecycleStatus` call `time.Now()` implicitly, making time-bound tests flaky and non-deterministic. We will refactor such functions to accept a `now time.Time` parameter. Callers (services) will pass `time.Now()`. Tests will pass frozen timestamps.

### 5. Fuzz tests for crypto and validation helpers
**Rationale:** `Canonicalize`, `EncryptLicenseFile`/`DecryptLicenseFile`, and `validateConstraintSchema` have huge input spaces but clear invariants (determinism, round-trip, no-panic). Go 1.18+ built-in fuzzing is the perfect fit and requires no new dependencies.

### 6. Handler tests only for error-mapping branches
**Rationale:** Handlers are thin glue code (bind JSON → call service → map error to HTTP status). We will test only the sentinel-error-to-status-code branches using `gin.CreateTestContext` and a hand-rolled service stub. Full handler coverage is low ROI.

## Risks / Trade-offs

- **[Risk]** Refactoring `time.Now()` into parameters touches multiple call sites and could introduce regressions.
  **Mitigation:** Make the change mechanical (rename internal helper, add `now` parameter, pass `time.Now()` at top-level). No behavioral change.

- **[Risk]** In-memory SQLite behaves slightly differently from PostgreSQL in edge cases (e.g. foreign key enforcement).
  **Mitigation:** We are testing license business logic, not PostgreSQL-specific syntax. GORM abstracts the dialect well enough for our queries.

- **[Trade-off]** Service integration tests are slower than pure unit tests.
  **Acceptance:** Still fast enough for local TDD (&lt; 1s for the full suite). The realism gain outweighs the speed cost.
