## Context

The knowledge module compiles source documents into a FalkorDB graph of concept nodes. The current RAG pipeline retrieves concept nodes via hybrid search (vector + fulltext) and returns LLM-compiled wiki articles to the answering agent. Original source text is never included in the retrieval response.

Key current state:
- `KnowledgeNode` has `SourceIDs` (JSON array of SQL IDs), linking concepts back to source records
- `KnowledgeEdge.Description` field exists in the model but is always stored as empty string `""`
- The `mapSystemPrompt` and `compileSystemPrompt` request `references` and `contradicts` lists but no edge descriptions
- `respondSearchResult` in `knowledge_query_handler.go` strips node content (`r.Content = ""`) for list responses; full content is fetched separately via `GetNode`
- The Node Token API at `/api/v1/ai/knowledge/search` and `/api/v1/ai/knowledge/nodes/:id` serves the agent runtime

## Goals / Non-Goals

**Goals:**
- Reduce hallucination risk by grounding RAG context in original source text
- Activate edge `description` for semantic edge explanations
- Add `keywords` field to nodes for better search recall
- Add inline source citations (`[S1]`, `[S2]`) in compiled content for provenance tracking
- Keep changes backward-compatible: existing graphs continue to work without recompilation

**Non-Goals:**
- Introducing new edge relation types (keeping `related` + `contradicts`)
- Adding Source nodes to FalkorDB (source linkage via `sourceIds` JSON property is sufficient)
- Changing the graph visualization frontend beyond minor keyword/citation display
- Modifying the agent runtime's system prompt construction (that's a separate concern)

## Decisions

### D1: Source text delivery — on-demand via search response, not pre-embedded

When the search API returns concept nodes for RAG, include a `sourceTexts` map in the response. The query handler resolves `sourceIds` → SQL source records → extracts content snippets.

**Why not embed sources in the graph?** FalkorDB node properties are loaded entirely on read. Storing full source text (potentially MBs) as node properties would bloat every graph query. SQL lookup via `sourceIds` is cheap and only happens at RAG retrieval time.

**Why not a separate API call?** Adding source text to the existing search response keeps the agent integration simple — one request returns everything needed for grounded answering.

**Truncation**: Source texts can be very long. Include up to 2000 characters per source, prioritizing the first section. The agent can request full source content via a follow-up `GetNode`-style endpoint if needed.

### D2: Edge description — LLM-generated during compilation, stored in FalkorDB

Update `compileSystemPrompt` and `mapSystemPrompt` to request a `description` field on each reference/contradiction. Store in the existing `KnowledgeEdge.Description` field (already in FalkorDB schema, just unused).

**Alternatives considered:**
- Dynamic relation type names (e.g., "inspired_by", "supersedes") — rejected because normalization is impossible, and programmatic edge traversal needs stable types
- Separate description node — over-engineered for a simple string property

### D3: Keywords — JSON array property on FalkorDB nodes

Add `Keywords` field to `KnowledgeNode` model (stored as JSON string in FalkorDB). LLM generates 3-8 keywords per node during compilation. Keywords feed into fulltext search index for better recall.

**Storage format**: JSON string `["keyword1", "keyword2"]` as a FalkorDB node property, matching the `sourceIds` pattern already used.

### D4: Inline citations — `[S1]...[SN]` markers in compiled content

LLM is instructed to add `[S1]`, `[S2]` markers in the compiled wiki article, referencing the source documents by index. The source mapping is encoded in a `citationMap` JSON property on the node: `{"S1": "Source Title 1", "S2": "Source Title 2"}`.

**Why title-based, not ID-based?** During compilation, the LLM works with source titles (not SQL IDs). The `citationMap` uses the same title→index mapping visible to the LLM. At retrieval time, titles are resolved to source IDs via `sourceIds`.

### D5: Backward compatibility — old nodes work without recompilation

New fields (`keywords`, `citationMap`) default to empty. Edge `description` defaults to empty string. The search response gracefully handles missing fields. Recompilation is needed only for new features to take effect.

## Risks / Trade-offs

- **Increased prompt complexity** → LLM may produce lower-quality articles if overloaded with instructions. Mitigation: keep citation/keyword instructions concise, test with real sources.
- **Source text size in search response** → Large knowledge bases with many sources could produce heavy responses. Mitigation: 2000-char truncation per source, max 3 sources per node in search response.
- **Citation accuracy** → LLM may place `[S1]` markers incorrectly. Mitigation: citations are informational, not enforced. Incorrect markers don't break functionality.
- **Keyword quality** → LLM-generated keywords may be too generic. Mitigation: prompt instructs "specific, domain-relevant keywords", and keywords supplement (not replace) fulltext search.
