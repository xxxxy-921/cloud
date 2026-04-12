## MODIFIED Requirements

### Requirement: LLM compilation produces knowledge graph nodes
The system SHALL compile source documents into knowledge graph concept nodes via LLM. Each node MUST be a complete, self-contained wiki article with non-empty content. Compilation SHALL additionally produce `keywords` (3-8 domain-specific keywords per node), `citationMap` (mapping `[S1]...[SN]` markers to source titles), and `description` on edges (explaining the semantic relationship).

#### Scenario: Compile produces nodes with keywords and citations
- **WHEN** the ai-knowledge-compile task processes sources
- **THEN** each new or updated node includes a `keywords` JSON array of 3-8 domain-specific keywords
- **THEN** each node's content includes inline `[S1]`, `[S2]` citation markers referencing source documents
- **THEN** each node includes a `citationMap` JSON object mapping marker keys to source titles

#### Scenario: Compile produces edges with descriptions
- **WHEN** the LLM outputs `references` or `contradicts` lists
- **THEN** each reference/contradiction includes a `description` string explaining WHY the two concepts are related or contradictory
- **THEN** the description is stored in the existing `KnowledgeEdge.Description` field

#### Scenario: Compile with legacy data
- **WHEN** nodes exist from a previous compilation without keywords/citationMap
- **THEN** those nodes continue to function normally with empty keywords and no citationMap
- **THEN** recompilation updates them with the new fields

### Requirement: Knowledge node data model
The system SHALL store knowledge nodes in FalkorDB with properties: id, title, summary, content, node_type, source_ids, compiled_at, keywords, citation_map. The `keywords` property SHALL be a JSON string array. The `citation_map` property SHALL be a JSON object mapping citation keys to source titles.

#### Scenario: Upsert node with keywords and citation map
- **WHEN** a node is created or updated via `UpsertNodeByTitle`
- **THEN** the `keywords` property is set as a JSON string on the FalkorDB node
- **THEN** the `citation_map` property is set as a JSON string on the FalkorDB node

#### Scenario: Read node with new properties
- **WHEN** a node is read from FalkorDB
- **THEN** the `keywords` and `citationMap` fields are parsed from the node properties
- **THEN** if the properties are missing (legacy nodes), they default to empty values
