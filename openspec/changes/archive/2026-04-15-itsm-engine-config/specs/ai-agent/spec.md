## MODIFIED Requirements

### Requirement: Agent entity with two types
The system SHALL support an Agent entity with three types: `assistant`, `coding`, and `internal`. Each Agent SHALL have: name (unique), code (unique, optional — required for internal type), description, avatar, type, visibility (`private` | `team` | `public`), created_by (FK → User), and is_active status. Internal agents are created by application modules for programmatic use and SHALL NOT appear in user-facing agent listings.

#### Scenario: Create assistant agent
- **WHEN** admin creates an Agent with type `assistant`
- **THEN** system stores the agent with assistant-specific fields (strategy, model_id, system_prompt, temperature, max_tokens, max_turns)

#### Scenario: Create coding agent
- **WHEN** admin creates an Agent with type `coding`
- **THEN** system stores the agent with coding-specific fields (runtime, runtime_config, exec_mode, node_id, workspace)

#### Scenario: Create internal agent
- **WHEN** a module creates an Agent with type `internal` and a unique code
- **THEN** system stores the agent with LLM configuration fields (model_id, system_prompt, temperature) and the specified code. Fields specific to assistant (strategy, max_turns) or coding (runtime, exec_mode) are ignored.

#### Scenario: Internal agent requires code
- **WHEN** a module creates an Agent with type `internal` but without a code
- **THEN** system SHALL return a 400 error "internal agent requires a unique code"

#### Scenario: Agent name uniqueness
- **WHEN** admin creates an Agent with a name that already exists
- **THEN** system SHALL return a 409 conflict error

#### Scenario: Agent code uniqueness
- **WHEN** a module creates an Agent with a code that already exists
- **THEN** system SHALL return a 409 conflict error

### Requirement: Agent CRUD API
The system SHALL provide REST endpoints under `/api/v1/ai/agents` with JWT + Casbin auth:
- `POST /` — create agent
- `GET /` — list agents (with pagination, keyword search, type filter, visibility filter)
- `GET /:id` — get agent detail
- `PUT /:id` — update agent
- `DELETE /:id` — soft-delete agent (blocked if active sessions exist)

Internal agents SHALL be excluded from the default list response unless explicitly requested via `type=internal` filter.

#### Scenario: List agents with type filter
- **WHEN** user requests `GET /api/v1/ai/agents?type=assistant`
- **THEN** system SHALL return only assistant-type agents visible to the user

#### Scenario: Default agent listing excludes internal
- **WHEN** user requests `GET /api/v1/ai/agents` without type filter
- **THEN** system SHALL return only `assistant` and `coding` type agents, excluding `internal` type

#### Scenario: Explicit internal agent listing
- **WHEN** user requests `GET /api/v1/ai/agents?type=internal`
- **THEN** system SHALL return internal-type agents

#### Scenario: Delete agent with active sessions
- **WHEN** admin deletes an agent that has sessions with status `running`
- **THEN** system SHALL return a 409 error

### Requirement: Agent code-based lookup
The system SHALL support looking up an Agent by its code field, enabling modules to retrieve their internal agents programmatically.

#### Scenario: Lookup by code
- **WHEN** a module calls `GetByCode("itsm.generator")`
- **THEN** system SHALL return the Agent with that code, or an error if not found

#### Scenario: Code is optional for non-internal agents
- **WHEN** an assistant or coding agent is created without a code
- **THEN** system SHALL allow creation with code as empty string
