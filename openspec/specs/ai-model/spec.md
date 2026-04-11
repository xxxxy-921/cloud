# Capability: ai-model

## Purpose
Manages AI model definitions including CRUD operations, capability tagging, default model selection, and model syncing from providers.

## Requirements

### Requirement: Model CRUD
The system SHALL support creating, reading, updating, and deleting AI models. Each model SHALL have: model_id (the actual model identifier used in API calls), display_name, provider_id (FK), type (`llm` / `embed` / `rerank` / `tts` / `stt` / `image`), capabilities (JSON array, LLM only), context_window, max_output_tokens, input_price, output_price, is_default, status (`active` / `deprecated`).

#### Scenario: Create a model
- **WHEN** admin submits a valid model form with model_id, display_name, provider_id, and type
- **THEN** the system creates the model linked to the provider

#### Scenario: List models
- **WHEN** admin requests the model list
- **THEN** the system returns paginated models with provider name, filterable by type and provider_id

#### Scenario: List models grouped by type
- **WHEN** frontend requests models for display
- **THEN** models are returned grouped by type (llm / embed / rerank / tts / stt / image) with provider info

#### Scenario: Update a model
- **WHEN** admin updates a model's display_name, pricing, or capabilities
- **THEN** the system saves the changes

#### Scenario: Delete a model
- **WHEN** admin deletes a model
- **THEN** the system soft-deletes the model

### Requirement: Model capabilities
The system SHALL support capability tags for LLM-type models. Valid capabilities are: `vision`, `tool_use`, `reasoning`, `coding`, `long_context`. Capabilities are stored as a JSON array.

#### Scenario: Set capabilities on LLM model
- **WHEN** admin creates or updates an LLM model with capabilities `["vision", "tool_use"]`
- **THEN** the system stores the capabilities array

#### Scenario: Non-LLM model has no capabilities
- **WHEN** admin creates an embed/rerank/tts/stt/image model
- **THEN** capabilities is stored as an empty array `[]`

### Requirement: Default model per type
The system SHALL support one default model per model type. Setting a new default SHALL automatically clear the previous default of the same type.

#### Scenario: Set default model
- **WHEN** admin marks a model as default
- **THEN** the system clears is_default on all other models of the same type and sets is_default on the selected model

#### Scenario: Query default model
- **WHEN** another module queries for the default LLM model
- **THEN** the system returns the model with is_default=true and type=llm

### Requirement: Model sync from provider
The system SHALL support syncing models from a provider's API. Sync mode depends on the provider type.

#### Scenario: Sync from OpenAI-compatible provider
- **WHEN** admin triggers model sync for an openai/ollama provider
- **THEN** the system calls the provider's list models API, compares with existing models, and adds new ones (never deletes existing)

#### Scenario: Sync from Anthropic provider
- **WHEN** admin triggers model sync for an anthropic provider
- **THEN** the system populates from a built-in known model list (claude-sonnet-4-20250514, claude-opus-4-20250514, etc.) without calling an external API

#### Scenario: Manual model creation always available
- **WHEN** admin manually creates a model for any provider
- **THEN** the system accepts it regardless of whether the model appears in the provider's API
