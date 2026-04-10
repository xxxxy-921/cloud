## MODIFIED Requirements

### Requirement: Embed frontend dist into binary
The system SHALL use `//go:embed` to embed the `web/dist` directory into the Go binary. The build process SHALL include a registry generation step before frontend compilation when `APPS` parameter is specified.

#### Scenario: Production mode static file serving
- **WHEN** the binary runs and a request is made to a path that matches a file in the embedded `web/dist`
- **THEN** the system SHALL serve the embedded static file with appropriate content type

#### Scenario: SPA fallback
- **WHEN** a request path does not match any API route or embedded static file
- **THEN** the system SHALL serve the embedded `index.html` to support client-side routing

#### Scenario: API routes take precedence
- **WHEN** a request is made to `/api/v1/*`
- **THEN** the API handler SHALL process it, not the static file server

#### Scenario: Registry generation before build
- **WHEN** `make build APPS=system,ai` is executed
- **THEN** the build process SHALL run `gen-registry.sh` before `bun run build`, then restore registry.ts after build completes
