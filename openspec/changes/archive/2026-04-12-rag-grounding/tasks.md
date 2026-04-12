## 1. Model & Schema Changes

- [x] 1.1 Add `Keywords` (string) and `CitationMap` (string) fields to `KnowledgeNode` struct in `knowledge_model.go`
- [x] 1.2 Add `Keywords` (json.RawMessage) and `CitationMap` (json.RawMessage) fields to `KnowledgeNodeResponse`, update `ToResponse()`
- [x] 1.3 Update `mapNodeOutput` struct: add `Keywords []string` and `Description` field to reference entries
- [x] 1.4 Update `compileNodeOutput` struct: add `Keywords []string`, `CitationMap map[string]string`; change `References`/`Contradicts` from `[]string` to `[]compileRelationOutput` with `Name` + `Description`

## 2. LLM Prompt Changes

- [x] 2.1 Update `mapSystemPrompt`: instruct LLM to output `keywords` (3-8 per node), `description` on each reference, and `[S1]...[SN]` citation markers in content
- [x] 2.2 Update `compileSystemPrompt`: same additions — `keywords`, edge `description`, citation markers `[S1]...[SN]`, and `citation_map` object in output
- [x] 2.3 Update `scanSystemPrompt` (long-doc): add `keywords` to scan output for better concept deduplication

## 3. Compile Pipeline Adaptations

- [x] 3.1 In `writeCompileOutput`: pass `Keywords` and `CitationMap` to `UpsertNodeByTitle`; pass `Description` to edge creation
- [x] 3.2 In `mapSource` fast path: build source index header (`[S1] = "Source Title"`) in the user prompt so LLM can cite sources
- [x] 3.3 In `buildReducePrompt`: include source index mapping for citation markers; propagate keywords from map results

## 4. FalkorDB Repository Changes

- [x] 4.1 Update `UpsertNodeByTitle` Cypher: set `keywords` and `citation_map` properties on node
- [x] 4.2 Update `UpsertNode` (by ID): set `keywords` and `citation_map` properties
- [x] 4.3 Update `recordToNode`: parse `keywords` and `citation_map` from FalkorDB node properties into `KnowledgeNode` fields
- [x] 4.4 Update edge creation methods: pass `description` parameter to Cypher SET clause (currently always empty)

## 5. Search Response — Source Text Grounding

- [x] 5.1 Add `SourceTextEntry` struct: `{ID uint, Title string, Content string, Format string}`
- [x] 5.2 In `respondSearchResult`: collect unique sourceIds from returned nodes, batch-fetch source records from SQL, truncate content to 2000 chars, include `sourceTexts` map in response
- [x] 5.3 Cap source texts at 3 per node to limit response size

## 6. Frontend Adaptations

- [x] 6.1 Display `keywords` as tags on node detail panel in knowledge graph view
- [x] 6.2 Render `[S1]...[SN]` citation markers as styled badges or links in node content view
- [x] 6.3 Update i18n locale files (en.json, zh-CN.json) with new labels for keywords and citations

## 7. Verification

- [x] 7.1 Verify `go build -tags dev ./cmd/server/` compiles without errors
- [x] 7.2 Verify `cd web && bun run lint` passes
- [x] 7.3 Test: recompile a knowledge base — nodes should have keywords, citation markers, and edges should have descriptions
