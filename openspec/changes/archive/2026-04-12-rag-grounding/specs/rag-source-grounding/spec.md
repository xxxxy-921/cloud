## ADDED Requirements

### Requirement: Search response includes source texts for RAG grounding
The search API SHALL include original source text alongside concept nodes when returning results. For each returned node, the system SHALL resolve the node's `sourceIds` to SQL source records and include truncated source content in the response.

#### Scenario: Hybrid search returns nodes with source texts
- **WHEN** agent calls GET /api/v1/ai/knowledge/search?q=...&kb_id=...&mode=hybrid
- **THEN** response includes a `sourceTexts` map keyed by source ID, containing `{id, title, content, format}` for each source referenced by returned nodes
- **THEN** each source's content is truncated to at most 2000 characters
- **THEN** at most 3 source texts are included per node (by sourceIds order)

#### Scenario: Node has no source IDs
- **WHEN** a returned node has empty or missing `sourceIds`
- **THEN** no source texts are included for that node
- **THEN** the response still includes the node normally

#### Scenario: Source record not found
- **WHEN** a node references a sourceId that no longer exists in SQL
- **THEN** the system SHALL skip that source without error
- **THEN** other valid source texts are still included

### Requirement: Admin search endpoint includes source texts
The admin-facing search endpoint SHALL include the same source text grounding as the agent-facing endpoint.

#### Scenario: Admin search returns source texts
- **WHEN** admin calls GET /api/v1/ai/knowledge-bases/:id/search?q=...
- **THEN** response includes `sourceTexts` map with the same behavior as the agent search endpoint

### Requirement: Citation map on node response
The node response SHALL include a `citationMap` field that maps inline citation markers (e.g., `S1`, `S2`) to source titles, enabling the consumer to resolve `[S1]` markers in the content.

#### Scenario: Node with citations
- **WHEN** a node has a non-empty `citationMap` property
- **THEN** the node response includes `citationMap` as a JSON object `{"S1": "Source Title", "S2": "Other Source"}`

#### Scenario: Node without citations (legacy)
- **WHEN** a node has no `citationMap` property (compiled before this change)
- **THEN** the `citationMap` field is omitted or empty in the response
