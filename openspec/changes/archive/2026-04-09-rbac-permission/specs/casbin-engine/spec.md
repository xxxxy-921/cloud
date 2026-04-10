## ADDED Requirements

### Requirement: Casbin model definition
The system SHALL use an RBAC model with resource-based permission checking. The model SHALL define request as (subject, object, action), policy as (subject, object, action), a role hierarchy grouping, and a matcher that checks role membership plus exact object and action match.

#### Scenario: Model loaded on startup
- **WHEN** the application starts
- **THEN** the Casbin enforcer SHALL load the RBAC model from an embedded string (not external file) and be ready for permission checks

### Requirement: GORM adapter for Casbin
The system SHALL use gorm-adapter/v3 to persist Casbin policies in a `casbin_rule` table, reusing the existing GORM database connection.

#### Scenario: Casbin table auto-created
- **WHEN** the Casbin enforcer initializes with the GORM adapter
- **THEN** the `casbin_rule` table SHALL be created automatically if it does not exist

#### Scenario: Policy persistence
- **WHEN** a policy is added via Casbin API
- **THEN** the policy SHALL be persisted in the `casbin_rule` table and survive application restarts

### Requirement: Casbin enforcer IOC registration
The system SHALL register the Casbin enforcer as a provider in the samber/do IOC container, enabling injection into middleware and services.

#### Scenario: Enforcer injection
- **WHEN** CasbinAuth middleware or CasbinService resolves from the IOC container
- **THEN** they SHALL receive the same Casbin enforcer instance

### Requirement: CasbinAuth middleware
The system SHALL provide a Gin middleware that intercepts authenticated requests, extracts the user's role code from JWT context, and checks permission via Casbin enforcer with (roleCode, requestPath, requestMethod).

#### Scenario: Authorized request
- **WHEN** a user with role "admin" accesses GET /api/v1/users and the Casbin policy allows ("admin", "/api/v1/users", "GET")
- **THEN** the request SHALL proceed to the handler

#### Scenario: Unauthorized request
- **WHEN** a user with role "user" accesses GET /api/v1/users and no Casbin policy allows it
- **THEN** the middleware SHALL return 403 with message "forbidden: insufficient permission"

#### Scenario: Whitelist routes bypass
- **WHEN** a request targets a whitelisted route (login, refresh, public site-info, /auth/me, /auth/password, /auth/logout)
- **THEN** the CasbinAuth middleware SHALL skip permission checking and pass the request through

### Requirement: Casbin service
The system SHALL provide a CasbinService that wraps the Casbin enforcer to offer higher-level operations: get policies for a role, set policies for a role (full replacement), and check permission.

#### Scenario: Get role policies
- **WHEN** CasbinService.GetPoliciesForRole("admin") is called
- **THEN** it SHALL return all policy rules where the subject is "admin"

#### Scenario: Set role policies (full replacement)
- **WHEN** CasbinService.SetPoliciesForRole("editor", newPolicies) is called
- **THEN** it SHALL remove all existing policies for "editor" and add the new policies atomically
