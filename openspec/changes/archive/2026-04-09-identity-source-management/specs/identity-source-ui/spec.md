## ADDED Requirements

### Requirement: Identity source management page（App 前端）
The identity App frontend SHALL provide a management page in `web/src/apps/identity/pages/index.tsx`, registered via `registerApp()` in `web/src/apps/identity/module.ts`.

#### Scenario: Admin views identity source list
- **WHEN** admin navigates to the identity source management page
- **THEN** SHALL display a table of all identity sources with name, type badge, domains, status, and action buttons

#### Scenario: Empty state
- **WHEN** no identity sources exist
- **THEN** SHALL display empty state with "create identity source" button

### Requirement: Create identity source form
The App frontend SHALL provide a Sheet form for creating identity sources with type-specific configuration fields.

#### Scenario: Select OIDC type
- **WHEN** admin selects "OIDC"
- **THEN** SHALL show: Issuer URL, Client ID, Client Secret, Scopes, read-only Callback URL

#### Scenario: Select LDAP type
- **WHEN** admin selects "LDAP"
- **THEN** SHALL show: Server URL, Bind DN, Bind Password, Search Base, User Filter, TLS options, Attribute Mapping

#### Scenario: Submit create form
- **WHEN** admin fills and submits
- **THEN** SHALL call POST /api/v1/identity-sources and refresh list

### Requirement: Edit identity source form
The App frontend SHALL provide a Sheet form for editing, with sensitive fields showing "••••••".

#### Scenario: Edit preserves secrets
- **WHEN** admin submits without changing Client Secret (still "••••••")
- **THEN** backend SHALL preserve existing encrypted value

### Requirement: Toggle, test, delete actions
The App frontend SHALL provide toggle, test connection, and delete actions.

#### Scenario: Toggle enabled
- **WHEN** admin clicks toggle
- **THEN** SHALL call PATCH /api/v1/identity-sources/:id/toggle

#### Scenario: Test connection success
- **WHEN** admin clicks "test connection" and it succeeds
- **THEN** SHALL show success toast

#### Scenario: Delete with confirmation
- **WHEN** admin clicks delete and confirms
- **THEN** SHALL call DELETE /api/v1/identity-sources/:id

### Requirement: App route registration
The identity App frontend SHALL register its routes via `registerApp()` in `module.ts`.

#### Scenario: Route registration
- **WHEN** `web/src/apps/identity/module.ts` is imported
- **THEN** `registerApp()` SHALL register the identity source management page route and SSO callback route
