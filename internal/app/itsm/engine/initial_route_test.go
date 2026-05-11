package engine

import (
	"encoding/json"
	"testing"
)

func TestInitialReachableParticipantNodesFollowsExclusiveGatewayByFormData(t *testing.T) {
	workflow := json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"gate","type":"exclusive","data":{"label":"初始路由"}},
			{"id":"net","type":"process","data":{"label":"网络处理","participants":[{"type":"user","value":"7"}]}},
			{"id":"fallback","type":"process","data":{"label":"默认处理","participants":[{"type":"user","value":"8"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"gate","data":{}},
			{"id":"e2","source":"gate","target":"net","data":{"condition":{"field":"form.request_kind","operator":"equals","value":"network"}}},
			{"id":"e3","source":"gate","target":"fallback","data":{"default":true}},
			{"id":"e4","source":"net","target":"end","data":{}},
			{"id":"e5","source":"fallback","target":"end","data":{}}
		]
	}`)

	nodes, err := InitialReachableParticipantNodes(workflow, map[string]any{"request_kind": "network"})
	if err != nil {
		t.Fatalf("InitialReachableParticipantNodes: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "net" || nodes[0].Label != "网络处理" {
		t.Fatalf("unexpected reachable nodes: %+v", nodes)
	}

	nodes, err = InitialReachableParticipantNodes(workflow, map[string]any{"request_kind": "other"})
	if err != nil {
		t.Fatalf("InitialReachableParticipantNodes fallback: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "fallback" {
		t.Fatalf("unexpected fallback node selection: %+v", nodes)
	}
}

func TestInitialReachableParticipantNodesTraversesRequesterFormAndInclusiveBranches(t *testing.T) {
	workflow := json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"form","type":"form","data":{"label":"申请表","participants":[{"type":"requester"}]}},
			{"id":"fork","type":"inclusive","data":{"label":"分支"}},
			{"id":"net","type":"process","data":{"label":"网络处理","participants":[{"type":"user","value":"7"}]}},
			{"id":"security","type":"process","data":{"label":"安全处理","participants":[{"type":"user","value":"8"}]}},
			{"id":"fallback","type":"process","data":{"label":"默认处理","participants":[{"type":"user","value":"9"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"form","data":{}},
			{"id":"e2","source":"form","target":"fork","data":{}},
			{"id":"e3","source":"fork","target":"net","data":{"condition":{"field":"form.need_network","operator":"equals","value":"yes"}}},
			{"id":"e4","source":"fork","target":"security","data":{"condition":{"field":"form.need_security","operator":"equals","value":"yes"}}},
			{"id":"e5","source":"fork","target":"fallback","data":{"default":true}},
			{"id":"e6","source":"net","target":"end","data":{}},
			{"id":"e7","source":"security","target":"end","data":{}},
			{"id":"e8","source":"fallback","target":"end","data":{}}
		]
	}`)

	nodes, err := InitialReachableParticipantNodes(workflow, map[string]any{
		"need_network":  "yes",
		"need_security": "yes",
	})
	if err != nil {
		t.Fatalf("InitialReachableParticipantNodes: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("reachable node count = %d, want 3, nodes=%+v", len(nodes), nodes)
	}
	if nodes[0].ID != "form" || nodes[1].ID != "net" || nodes[2].ID != "security" {
		t.Fatalf("unexpected reachable node order: %+v", nodes)
	}
}

func TestInitialReachableParticipantNodesReturnsNoMatchErrorWithoutDefault(t *testing.T) {
	workflow := json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"gate","type":"exclusive","data":{"label":"初始路由"}},
			{"id":"net","type":"process","data":{"label":"网络处理","participants":[{"type":"user","value":"7"}]}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"gate","data":{}},
			{"id":"e2","source":"gate","target":"net","data":{"condition":{"field":"form.request_kind","operator":"equals","value":"network"}}}
		]
	}`)

	if _, err := InitialReachableParticipantNodes(workflow, map[string]any{"request_kind": "database"}); err == nil {
		t.Fatal("expected no-match error, got nil")
	}
}

func TestInitialReachableParticipantNodesStopsAtNonRequesterFormAndUsesGatewayDataConditions(t *testing.T) {
	workflow := json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"form","type":"form","data":{"label":"人工补充","participants":[{"type":"position","value":"ops_admin"}]}},
			{"id":"gate","type":"exclusive","data":{"label":"初始路由","conditions":[{"field":"form.request_kind","operator":"equals","value":"vpn","edge_id":"edge-vpn"}]}},
			{"id":"vpn","type":"process","data":{"label":"VPN 处理","participants":[{"type":"user","value":"7"}]}},
			{"id":"default","type":"process","data":{"label":"默认处理","participants":[{"type":"user","value":"8"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges":[
			{"id":"edge-start-form","source":"start","target":"form","data":{}},
			{"id":"edge-form-gate","source":"form","target":"gate","data":{}},
			{"id":"edge-vpn","source":"gate","target":"vpn","data":{}},
			{"id":"edge-default","source":"gate","target":"default","data":{"default":true}},
			{"id":"edge-vpn-end","source":"vpn","target":"end","data":{}},
			{"id":"edge-default-end","source":"default","target":"end","data":{}}
		]
	}`)

	nodes, err := InitialReachableParticipantNodes(workflow, map[string]any{"request_kind": "vpn"})
	if err != nil {
		t.Fatalf("InitialReachableParticipantNodes: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "form" {
		t.Fatalf("non-requester form should stop traversal at current form, got %+v", nodes)
	}

	requesterOnlyWorkflow := json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"form","type":"form","data":{"label":"申请表","participants":[{"type":"requester"}]}},
			{"id":"gate","type":"exclusive","data":{"label":"初始路由","conditions":[{"field":"form.request_kind","operator":"equals","value":"vpn","edge_id":"edge-vpn"}]}},
			{"id":"vpn","type":"process","data":{"label":"VPN 处理","participants":[{"type":"user","value":"7"}]}},
			{"id":"default","type":"process","data":{"label":"默认处理","participants":[{"type":"user","value":"8"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges":[
			{"id":"edge-start-form","source":"start","target":"form","data":{}},
			{"id":"edge-form-gate","source":"form","target":"gate","data":{}},
			{"id":"edge-vpn","source":"gate","target":"vpn","data":{}},
			{"id":"edge-default","source":"gate","target":"default","data":{"default":true}},
			{"id":"edge-vpn-end","source":"vpn","target":"end","data":{}},
			{"id":"edge-default-end","source":"default","target":"end","data":{}}
		]
	}`)

	nodes, err = InitialReachableParticipantNodes(requesterOnlyWorkflow, map[string]any{"request_kind": "vpn"})
	if err != nil {
		t.Fatalf("InitialReachableParticipantNodes requester path: %v", err)
	}
	if len(nodes) != 2 || nodes[0].ID != "form" || nodes[1].ID != "vpn" {
		t.Fatalf("gateway data conditions should route requester flow to vpn branch, got %+v", nodes)
	}
}

func TestInitialReachableParticipantNodesFallsBackToInclusiveDefaultAndHelperSemantics(t *testing.T) {
	workflow := json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"fork","type":"inclusive","data":{"label":"分支"}},
			{"id":"fallback","type":"process","data":{"label":"默认处理","participants":[{"type":"user","value":"9"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"fork","data":{}},
			{"id":"e2","source":"fork","target":"fallback","data":{"default":true}},
			{"id":"e3","source":"fallback","target":"end","data":{}}
		]
	}`)

	nodes, err := InitialReachableParticipantNodes(workflow, map[string]any{"need_network": "no"})
	if err != nil {
		t.Fatalf("InitialReachableParticipantNodes inclusive default: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "fallback" {
		t.Fatalf("inclusive default should be selected when no branch matches, got %+v", nodes)
	}

	if !formNodeSatisfiedByInitialRequest(nil) {
		t.Fatal("empty participants should be treated as requester-satisfied")
	}
	if !formNodeSatisfiedByInitialRequest([]Participant{{Type: "requester"}}) {
		t.Fatal("requester-only participants should be satisfied by initial request")
	}
	if formNodeSatisfiedByInitialRequest([]Participant{{Type: "requester"}, {Type: "user", Value: "7"}}) {
		t.Fatal("mixed requester and non-requester participants should not auto-progress")
	}
}

func TestInitialReachableParticipantNodesCoversParallelDefaultAndInvalidWorkflowPaths(t *testing.T) {
	t.Run("parallel and generic nodes keep traversing until participant nodes", func(t *testing.T) {
		workflow := json.RawMessage(`{
			"nodes":[
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"script","type":"script","data":{"label":"脚本准备"}},
				{"id":"fork","type":"parallel","data":{"label":"并行分流"}},
				{"id":"ops","type":"process","data":{"label":"运维处理","participants":[{"type":"user","value":"11"}]}},
				{"id":"sec","type":"process","data":{"label":"安全处理","participants":[{"type":"user","value":"12"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges":[
				{"id":"e1","source":"start","target":"script","data":{}},
				{"id":"e2","source":"script","target":"fork","data":{}},
				{"id":"e3","source":"fork","target":"ops","data":{}},
				{"id":"e4","source":"fork","target":"sec","data":{}},
				{"id":"e5","source":"ops","target":"end","data":{}},
				{"id":"e6","source":"sec","target":"end","data":{}}
			]
		}`)

		nodes, err := InitialReachableParticipantNodes(workflow, nil)
		if err != nil {
			t.Fatalf("InitialReachableParticipantNodes parallel path: %v", err)
		}
		if len(nodes) != 2 || nodes[0].ID != "ops" || nodes[1].ID != "sec" {
			t.Fatalf("parallel traversal should reach both participant nodes, got %+v", nodes)
		}
	})

	t.Run("invalid workflow inputs surface concrete errors", func(t *testing.T) {
		if _, err := InitialReachableParticipantNodes(json.RawMessage(`{`), nil); err == nil {
			t.Fatal("invalid workflow json should fail")
		}

		missingNodeWorkflow := json.RawMessage(`{
			"nodes":[
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"gate","type":"parallel","data":{"label":"并行分流"}}
			],
			"edges":[
				{"id":"e1","source":"start","target":"gate","data":{}},
				{"id":"e2","source":"gate","target":"missing","data":{}}
			]
		}`)
		if _, err := InitialReachableParticipantNodes(missingNodeWorkflow, nil); err == nil {
			t.Fatal("missing workflow target node should fail")
		}
	})
}
