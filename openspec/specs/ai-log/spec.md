# Capability: ai-log

## Purpose
Defines the AI call log model for tracking LLM API calls, including token usage, cost, latency, and status.

## Requirements

### Requirement: AI call log model
The system SHALL define an AILog model for tracking LLM API calls. Fields: model_id, provider_id, user_id, app_source, input_tokens, output_tokens, total_cost, latency_ms, status (`success` / `error` / `timeout`), error_message, created_at.

#### Scenario: AILog table created on migration
- **WHEN** the AI app is loaded and database migration runs
- **THEN** the `ai_logs` table is created with all defined columns

#### Scenario: AILog model available for future use
- **WHEN** an upper-layer module (agent-runtime, knowledge) makes an LLM call
- **THEN** it can create an AILog record via the repository (logging middleware not implemented in this change)
