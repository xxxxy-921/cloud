## ADDED Requirements

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

### Requirement: Vite dev proxy
During development, the Vite dev server SHALL proxy `/api` requests to the Go server.

#### Scenario: Dev mode API proxy
- **WHEN** the Vite dev server is running on port 3000 and Go server on port 8080
- **THEN** requests to `localhost:3000/api/*` SHALL be proxied to `localhost:8080/api/*`

#### Scenario: Tailwind CSS plugin configured
- **WHEN** the Vite build runs
- **THEN** the @tailwindcss/vite plugin SHALL be included in the Vite plugin chain

### Requirement: Single binary deployment
The built binary SHALL contain all frontend assets and run without any external file dependencies.

#### Scenario: Run from any directory
- **WHEN** the compiled binary is copied to an arbitrary directory and executed
- **THEN** the server SHALL start and serve both API and frontend without errors (database file excluded)
