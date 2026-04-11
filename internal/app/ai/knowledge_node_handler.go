package ai

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"metis/internal/handler"
)

type KnowledgeNodeHandler struct {
	nodeRepo *KnowledgeNodeRepo
	edgeRepo *KnowledgeEdgeRepo
	logRepo  *KnowledgeLogRepo
}

func NewKnowledgeNodeHandler(i do.Injector) (*KnowledgeNodeHandler, error) {
	return &KnowledgeNodeHandler{
		nodeRepo: do.MustInvoke[*KnowledgeNodeRepo](i),
		edgeRepo: do.MustInvoke[*KnowledgeEdgeRepo](i),
		logRepo:  do.MustInvoke[*KnowledgeLogRepo](i),
	}, nil
}

func (h *KnowledgeNodeHandler) List(c *gin.Context) {
	kbID, _ := strconv.Atoi(c.Param("id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	items, total, err := h.nodeRepo.List(NodeListParams{
		KbID:     uint(kbID),
		Keyword:  c.Query("keyword"),
		NodeType: c.Query("nodeType"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]KnowledgeNodeResponse, len(items))
	for i, n := range items {
		resp[i] = n.ToResponse()
		edgeCount, _ := h.edgeRepo.CountByNodeID(n.ID)
		resp[i].EdgeCount = int(edgeCount)
	}
	handler.OK(c, gin.H{"items": resp, "total": total})
}

func (h *KnowledgeNodeHandler) Get(c *gin.Context) {
	nid, _ := strconv.Atoi(c.Param("nid"))
	node, err := h.nodeRepo.FindByID(uint(nid))
	if err != nil {
		handler.Fail(c, http.StatusNotFound, "node not found")
		return
	}

	resp := node.ToResponse()
	edgeCount, _ := h.edgeRepo.CountByNodeID(node.ID)
	resp.EdgeCount = int(edgeCount)

	handler.OK(c, resp)
}

func (h *KnowledgeNodeHandler) GetGraph(c *gin.Context) {
	nid, _ := strconv.Atoi(c.Param("nid"))
	depth, _ := strconv.Atoi(c.DefaultQuery("depth", "1"))

	nodes, edges, err := h.nodeRepo.GetGraphNodes(uint(nid), depth)
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

func (h *KnowledgeNodeHandler) GetFullGraph(c *gin.Context) {
	kbID, _ := strconv.Atoi(c.Param("id"))

	nodes, err := h.nodeRepo.FindByKbID(uint(kbID))
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Filter out index nodes — only concept nodes are visualized in the graph
	conceptNodes := make([]KnowledgeNode, 0, len(nodes))
	conceptIDs := make(map[uint]bool)
	for _, n := range nodes {
		if n.NodeType != "index" {
			conceptNodes = append(conceptNodes, n)
			conceptIDs[n.ID] = true
		}
	}

	edges, err := h.edgeRepo.FindByKbID(uint(kbID))
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	nodeResps := make([]KnowledgeNodeResponse, len(conceptNodes))
	for i, n := range conceptNodes {
		nodeResps[i] = n.ToResponse()
		edgeCount, _ := h.edgeRepo.CountByNodeID(n.ID)
		nodeResps[i].EdgeCount = int(edgeCount)
	}

	// Only include edges between concept nodes
	edgeResps := make([]KnowledgeEdgeResponse, 0, len(edges))
	for _, e := range edges {
		if conceptIDs[e.FromNodeID] && conceptIDs[e.ToNodeID] {
			edgeResps = append(edgeResps, e.ToResponse())
		}
	}

	handler.OK(c, gin.H{"nodes": nodeResps, "edges": edgeResps})
}

func (h *KnowledgeNodeHandler) ListLogs(c *gin.Context) {
	kbID, _ := strconv.Atoi(c.Param("id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	items, total, err := h.logRepo.List(uint(kbID), page, pageSize)
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	handler.OK(c, gin.H{"items": items, "total": total})
}
