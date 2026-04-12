## Why

Current RAG pipeline only sends LLM-compiled concept articles as context to the answering LLM. Original source text never participates in retrieval — compilation errors become uncorrectable "facts," creating a hallucination vector. Additionally, the knowledge graph edges carry no semantic explanation (description is always empty), and nodes lack keywords for better search matching. These gaps reduce RAG accuracy and make it harder for the LLM to determine which retrieved context is trustworthy.

## What Changes

- **RAG retrieval includes original sources**: When concept nodes are retrieved for RAG, the system also fetches and includes the original source text that produced each concept, so the answering LLM can cross-reference compiled knowledge against ground-truth documents.
- **Edge description activation**: LLM compilation prompts now request a `description` field explaining WHY two concepts are related/contradictory. This semantic richness helps RAG determine which neighbor concepts are relevant to a query.
- **Node keywords field**: Nodes gain a `keywords []string` field populated during compilation, improving full-text search recall and enabling keyword-based filtering.
- **Inline source citations**: Compiled concept content includes Wikipedia-style `[S1]`, `[S2]` markers mapping to source documents, so the RAG LLM can distinguish sourced facts from inferences.

## Capabilities

### New Capabilities
- `rag-source-grounding`: RAG retrieval pipeline sends original source text alongside compiled concept content, with citation mapping for provenance tracking.

### Modified Capabilities
- `ai-knowledge`: Compilation output gains edge descriptions, node keywords, and inline source citations. Node model adds `keywords` JSON field.

## Impact

- **Backend**: `knowledge_compile_service.go` (map/compile prompts, output structs), `knowledge_model.go` (Keywords field), `knowledge_graph_repository.go` (keywords in upsert/query), `knowledge_query_handler.go` (source text retrieval in search response)
- **LLM prompts**: Map and compile system prompts updated to request `description` on edges, `keywords` on nodes, and `[S1]...[SN]` citation markers in content
- **Frontend**: Minor — display keywords as tags on node detail, show citation markers in content
- **Graph schema**: FalkorDB node property `keywords` added; edge property `description` populated (field already exists)
