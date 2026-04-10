## MODIFIED Requirements

### Requirement: Vite dev proxy
During development, the Vite dev server SHALL proxy `/api` requests to the Go server.

#### Scenario: Dev mode API proxy
- **WHEN** the Vite dev server is running on port 3000 and Go server on port 8080
- **THEN** requests to `localhost:3000/api/*` SHALL be proxied to `localhost:8080/api/*`

#### Scenario: Tailwind CSS plugin configured
- **WHEN** the Vite build runs
- **THEN** the @tailwindcss/vite plugin SHALL be included in the Vite plugin chain
