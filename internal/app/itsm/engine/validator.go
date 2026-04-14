package engine

import (
	"encoding/json"
	"fmt"
)

// ValidationError represents a single validation issue.
type ValidationError struct {
	NodeID  string `json:"nodeId,omitempty"`
	EdgeID  string `json:"edgeId,omitempty"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string { return e.Message }

// ValidateWorkflow checks a workflow JSON for structural integrity.
// Returns a list of validation errors. An empty list means the workflow is valid.
func ValidateWorkflow(workflowJSON json.RawMessage) []ValidationError {
	var errs []ValidationError

	def, err := ParseWorkflowDef(workflowJSON)
	if err != nil {
		return []ValidationError{{Message: fmt.Sprintf("JSON 解析失败: %v", err)}}
	}

	nodeMap := make(map[string]*WFNode, len(def.Nodes))
	for i := range def.Nodes {
		nodeMap[def.Nodes[i].ID] = &def.Nodes[i]
	}

	// Build edge maps
	outEdges := make(map[string][]*WFEdge)
	inEdges := make(map[string][]*WFEdge)
	for i := range def.Edges {
		e := &def.Edges[i]
		outEdges[e.Source] = append(outEdges[e.Source], e)
		inEdges[e.Target] = append(inEdges[e.Target], e)
	}

	// 1. Validate node types
	var startNodes []*WFNode
	var endNodes []*WFNode
	for i := range def.Nodes {
		n := &def.Nodes[i]
		if !ValidNodeTypes[n.Type] {
			errs = append(errs, ValidationError{
				NodeID:  n.ID,
				Message: fmt.Sprintf("节点 %s 的类型 %q 不合法", n.ID, n.Type),
			})
		}
		if n.Type == NodeStart {
			startNodes = append(startNodes, n)
		}
		if n.Type == NodeEnd {
			endNodes = append(endNodes, n)
		}
	}

	// 2. Exactly one start node
	if len(startNodes) == 0 {
		errs = append(errs, ValidationError{Message: "工作流必须包含一个开始节点"})
	} else if len(startNodes) > 1 {
		errs = append(errs, ValidationError{Message: "工作流只能包含一个开始节点"})
	} else {
		// Start node must have exactly one outgoing edge
		start := startNodes[0]
		if len(outEdges[start.ID]) != 1 {
			errs = append(errs, ValidationError{
				NodeID:  start.ID,
				Message: "开始节点必须有且仅有一条出边",
			})
		}
		// Start node should have no incoming edges
		if len(inEdges[start.ID]) > 0 {
			errs = append(errs, ValidationError{
				NodeID:  start.ID,
				Message: "开始节点不应有入边",
			})
		}
	}

	// 3. At least one end node
	if len(endNodes) == 0 {
		errs = append(errs, ValidationError{Message: "工作流必须包含至少一个结束节点"})
	} else {
		// End nodes must have no outgoing edges
		for _, n := range endNodes {
			if len(outEdges[n.ID]) > 0 {
				errs = append(errs, ValidationError{
					NodeID:  n.ID,
					Message: fmt.Sprintf("结束节点 %s 不应有出边", n.ID),
				})
			}
		}
	}

	// 4. Edge references valid nodes
	for i := range def.Edges {
		e := &def.Edges[i]
		if _, ok := nodeMap[e.Source]; !ok {
			errs = append(errs, ValidationError{
				EdgeID:  e.ID,
				Message: fmt.Sprintf("边 %s 引用了不存在的源节点 %s", e.ID, e.Source),
			})
		}
		if _, ok := nodeMap[e.Target]; !ok {
			errs = append(errs, ValidationError{
				EdgeID:  e.ID,
				Message: fmt.Sprintf("边 %s 引用了不存在的目标节点 %s", e.ID, e.Target),
			})
		}
	}

	// 5. No isolated nodes (every non-start node must have at least one incoming edge)
	for i := range def.Nodes {
		n := &def.Nodes[i]
		if n.Type == NodeStart {
			continue
		}
		if len(inEdges[n.ID]) == 0 {
			errs = append(errs, ValidationError{
				NodeID:  n.ID,
				Message: fmt.Sprintf("节点 %s 没有入边，无法到达", n.ID),
			})
		}
	}

	// 6. Gateway constraints
	for i := range def.Nodes {
		n := &def.Nodes[i]
		if n.Type != NodeGateway {
			continue
		}
		edges := outEdges[n.ID]
		if len(edges) < 2 {
			errs = append(errs, ValidationError{
				NodeID:  n.ID,
				Message: fmt.Sprintf("网关节点 %s 至少需要两条出边", n.ID),
			})
			continue
		}
		for _, e := range edges {
			if !e.Data.Default && e.Data.Condition == nil {
				errs = append(errs, ValidationError{
					NodeID:  n.ID,
					EdgeID:  e.ID,
					Message: fmt.Sprintf("网关节点 %s 的出边 %s 缺少条件配置", n.ID, e.ID),
				})
			}
		}
	}

	return errs
}
