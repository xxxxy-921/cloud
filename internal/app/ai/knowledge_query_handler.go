package ai

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"metis/internal/handler"
	"metis/internal/llm"
	"metis/internal/pkg/crypto"
)

// KnowledgeQueryHandler serves the Agent-facing knowledge API.
// Routes are authenticated via NodeTokenMiddleware (Sidecar token).
type KnowledgeQueryHandler struct {
	graphRepo    *KnowledgeGraphRepo
	kbRepo       *KnowledgeBaseRepo
	sourceRepo   *KnowledgeSourceRepo
	modelRepo    *ModelRepo
	embeddingSvc *KnowledgeEmbeddingService
	encKey       crypto.EncryptionKey
}

func NewKnowledgeQueryHandler(i do.Injector) (*KnowledgeQueryHandler, error) {
	return &KnowledgeQueryHandler{
		graphRepo:    do.MustInvoke[*KnowledgeGraphRepo](i),
		kbRepo:       do.MustInvoke[*KnowledgeBaseRepo](i),
		sourceRepo:   do.MustInvoke[*KnowledgeSourceRepo](i),
		modelRepo:    do.MustInvoke[*ModelRepo](i),
		embeddingSvc: do.MustInvoke[*KnowledgeEmbeddingService](i),
		encKey:       do.MustInvoke[crypto.EncryptionKey](i),
	}, nil
}

// SourceTextEntry is a truncated source text for RAG grounding.
type SourceTextEntry struct {
	ID      uint   `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Format  string `json:"format"`
}

// Search performs hybrid knowledge search (vector + fulltext + graph expansion).
// GET /api/v1/ai/knowledge/search?q=&kb_id=&limit=&mode=hybrid|vector|fulltext
func (h *KnowledgeQueryHandler) Search(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		handler.Fail(c, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	kbID, _ := strconv.Atoi(c.Query("kb_id"))
	if kbID == 0 {
		handler.Fail(c, http.StatusBadRequest, "query parameter 'kb_id' is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	mode := c.DefaultQuery("mode", "hybrid")

	switch mode {
	case "vector":
		h.searchVector(c, uint(kbID), q, limit)
	case "fulltext":
		h.searchFullText(c, uint(kbID), q, limit)
	default:
		h.searchHybrid(c, uint(kbID), q, limit)
	}
}

// searchHybrid runs vector + fulltext concurrently, merges via RRF, then expands.
func (h *KnowledgeQueryHandler) searchHybrid(c *gin.Context, kbID uint, query string, limit int) {
	queryVec := h.embedQuery(c.Request.Context(), kbID, query)

	result, err := h.graphRepo.HybridSearch(kbID, queryVec, query, limit, 1)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondSearchResult(c, kbID, result.Nodes, result.Edges, result.Scores)
}

// searchVector runs vector-only search with graph expansion.
func (h *KnowledgeQueryHandler) searchVector(c *gin.Context, kbID uint, query string, limit int) {
	nodes, err := h.vectorSearch(c.Request.Context(), kbID, query, limit)
	if err != nil {
		slog.Debug("vector search failed, falling back to full-text", "kb_id", kbID, "error", err)
		h.searchFullText(c, kbID, query, limit)
		return
	}
	h.respondSearchResult(c, kbID, nodes, nil, nil)
}

// searchFullText runs fulltext-only search.
func (h *KnowledgeQueryHandler) searchFullText(c *gin.Context, kbID uint, query string, limit int) {
	nodes, scores, err := h.graphRepo.SearchFullText(kbID, query, limit)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	scoreMap := make(map[string]float64, len(nodes))
	for i, n := range nodes {
		if i < len(scores) {
			scoreMap[n.ID] = scores[i]
		}
	}
	h.respondSearchResult(c, kbID, nodes, nil, scoreMap)
}

// embedQuery tries to embed the query text; returns nil if embedding is not configured or fails.
func (h *KnowledgeQueryHandler) embedQuery(ctx context.Context, kbID uint, query string) []float32 {
	kb, err := h.kbRepo.FindByID(kbID)
	if err != nil || kb.EmbeddingProviderID == nil || kb.EmbeddingModelID == "" {
		return nil
	}
	client, err := h.embeddingSvc.resolveEmbeddingClient(kb)
	if err != nil {
		return nil
	}
	resp, err := client.Embedding(ctx, llm.EmbeddingRequest{
		Model: kb.EmbeddingModelID,
		Input: []string{query},
	})
	if err != nil || len(resp.Embeddings) == 0 {
		return nil
	}
	return resp.Embeddings[0]
}

// respondSearchResult builds the unified search response with nodes, edges, scores, and source texts.
func (h *KnowledgeQueryHandler) respondSearchResult(
	c *gin.Context, kbID uint,
	nodes []KnowledgeNode, edges []KnowledgeEdge, scores map[string]float64,
) {
	// Collect unique source IDs from all nodes (cap at 3 per node)
	sourceIDSet := make(map[uint]struct{})
	for _, n := range nodes {
		var ids []uint
		if n.SourceIDs != "" {
			if err := json.Unmarshal([]byte(n.SourceIDs), &ids); err != nil {
				slog.Warn("failed to parse node source_ids", "node_id", n.ID, "source_ids", n.SourceIDs, "error", err)
			}
		}
		for i, id := range ids {
			if i >= 3 {
				break
			}
			sourceIDSet[id] = struct{}{}
		}
	}

	// Batch-fetch source records and build sourceTexts map
	sourceTexts := make([]SourceTextEntry, 0)
	if len(sourceIDSet) > 0 {
		sourceIDs := make([]uint, 0, len(sourceIDSet))
		for id := range sourceIDSet {
			sourceIDs = append(sourceIDs, id)
		}
		sources, err := h.sourceRepo.FindByIDs(sourceIDs)
		if err != nil {
			slog.Warn("failed to fetch sources for RAG grounding", "source_ids", sourceIDs, "error", err)
		} else {
			if len(sources) < len(sourceIDs) {
				slog.Warn("some sources not found for RAG grounding", "requested", len(sourceIDs), "found", len(sources))
			}
			for _, src := range sources {
				content := src.Content
				if len(content) > 5000 {
					content = content[:5000]
				}
				sourceTexts = append(sourceTexts, SourceTextEntry{
					ID:      src.ID,
					Title:   src.Title,
					Content: content,
					Format:  src.Format,
				})
			}
		}
	}

	nodeResps := make([]KnowledgeNodeResponse, len(nodes))
	for i, n := range nodes {
		r := n.ToResponse()
		edgeCount, _ := h.graphRepo.CountEdgesForNode(kbID, n.ID)
		r.EdgeCount = edgeCount
		r.Content = ""
		if scores != nil {
			if s, ok := scores[n.ID]; ok {
				r.Score = s
			}
		}
		nodeResps[i] = r
	}

	edgeResps := make([]KnowledgeEdgeResponse, len(edges))
	for i, e := range edges {
		edgeResps[i] = e.ToResponse()
	}

	handler.OK(c, gin.H{"nodes": nodeResps, "edges": edgeResps, "sourceTexts": sourceTexts})
}

// vectorSearch embeds the query text and performs vector search + graph expansion.
func (h *KnowledgeQueryHandler) vectorSearch(ctx context.Context, kbID uint, query string, limit int) ([]KnowledgeNode, error) {
	kb, err := h.kbRepo.FindByID(kbID)
	if err != nil {
		return nil, err
	}

	// Check if embedding is configured
	if kb.EmbeddingProviderID == nil || kb.EmbeddingModelID == "" {
		return nil, errEmbeddingNotConfigured
	}

	// Resolve embedding client
	client, err := h.embeddingSvc.resolveEmbeddingClient(kb)
	if err != nil {
		return nil, err
	}

	// Embed the query
	resp, err := client.Embedding(ctx, llm.EmbeddingRequest{
		Model: kb.EmbeddingModelID,
		Input: []string{query},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Embeddings) == 0 {
		return nil, errEmbeddingEmpty
	}

	// Vector search with 1-hop graph expansion
	nodes, _, _, err := h.graphRepo.VectorSearchWithExpand(kbID, resp.Embeddings[0], limit, 1)
	return nodes, err
}

// SearchByKb is the admin-facing search endpoint (JWT auth).
// GET /api/v1/ai/knowledge-bases/:id/search?q=&limit=&mode=hybrid|vector|fulltext
func (h *KnowledgeQueryHandler) SearchByKb(c *gin.Context) {
	kbID, _ := strconv.Atoi(c.Param("id"))
	if kbID == 0 {
		handler.Fail(c, http.StatusBadRequest, "invalid knowledge base id")
		return
	}
	q := c.Query("q")
	if q == "" {
		handler.Fail(c, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	mode := c.DefaultQuery("mode", "hybrid")

	switch mode {
	case "vector":
		h.searchVector(c, uint(kbID), q, limit)
	case "fulltext":
		h.searchFullText(c, uint(kbID), q, limit)
	default:
		h.searchHybrid(c, uint(kbID), q, limit)
	}
}

// GetNode returns full node details including content.
// GET /api/v1/ai/knowledge/nodes/:id?kb_id=
func (h *KnowledgeQueryHandler) GetNode(c *gin.Context) {
	nodeID := c.Param("id")
	kbID, _ := strconv.Atoi(c.Query("kb_id"))
	if kbID == 0 {
		handler.Fail(c, http.StatusBadRequest, "query parameter 'kb_id' is required")
		return
	}

	node, err := h.graphRepo.FindNodeByID(uint(kbID), nodeID)
	if err != nil {
		handler.Fail(c, http.StatusNotFound, "node not found")
		return
	}

	resp := node.ToResponse()
	edgeCount, _ := h.graphRepo.CountEdgesForNode(uint(kbID), node.ID)
	resp.EdgeCount = edgeCount

	handler.OK(c, resp)
}

// GetNodeGraph returns N-hop subgraph around a node.
// GET /api/v1/ai/knowledge/nodes/:id/graph?kb_id=&depth=
func (h *KnowledgeQueryHandler) GetNodeGraph(c *gin.Context) {
	nodeID := c.Param("id")
	kbID, _ := strconv.Atoi(c.Query("kb_id"))
	if kbID == 0 {
		handler.Fail(c, http.StatusBadRequest, "query parameter 'kb_id' is required")
		return
	}
	depth, _ := strconv.Atoi(c.DefaultQuery("depth", "1"))

	nodes, edges, err := h.graphRepo.GetSubgraph(uint(kbID), nodeID, depth)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	nodeResps := make([]KnowledgeNodeResponse, len(nodes))
	for i, n := range nodes {
		nodeResps[i] = n.ToResponse()
	}

	edgeResps := make([]KnowledgeEdgeResponse, len(edges))
	for i, e := range edges {
		edgeResps[i] = e.ToResponse()
	}

	handler.OK(c, gin.H{"nodes": nodeResps, "edges": edgeResps})
}
