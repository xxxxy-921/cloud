package itsm

// vpn_support_test.go — VPN workflow fixture and service publish helpers for BDD tests.

import (
	"encoding/json"
	"fmt"
)

// vpnWorkflowIDs holds dynamic IDs needed for building the VPN workflow fixture.
type vpnWorkflowIDs struct {
	NetworkAdminPosID  uint
	SecurityAdminPosID uint
}

// vpnSampleFormData provides typical VPN request form values for BDD tests.
// The "request_kind" field drives the exclusive gateway routing.
var vpnSampleFormData = map[string]any{
	"request_kind": "network_support",
	"vpn_type":     "l2tp",
	"reason":       "需要远程访问内网开发环境",
}

// buildVPNWorkflowJSON returns a valid ReactFlow workflow JSON for the VPN activation service.
//
// Topology:
//
//	start → form_submit → exclusive_gw →[form.request_kind=="network_support"]→ approve_network → end_network
//	                                    →[default]→ approve_security → end_security
func buildVPNWorkflowJSON(ids vpnWorkflowIDs) json.RawMessage {
	workflow := fmt.Sprintf(`{
  "nodes": [
    {"id":"node_start","type":"start","data":{"label":"开始"},"position":{"x":250,"y":0}},
    {"id":"node_form","type":"form","data":{"label":"填写VPN申请表"},"position":{"x":250,"y":100}},
    {"id":"node_gw","type":"exclusive","data":{"label":"路由审批"},"position":{"x":250,"y":200}},
    {"id":"node_approve_network","type":"approve","data":{"label":"网络管理员审批","approve_mode":"single","participants":[{"type":"position","value":"%d"}]},"position":{"x":100,"y":300}},
    {"id":"node_approve_security","type":"approve","data":{"label":"安全管理员审批","approve_mode":"single","participants":[{"type":"position","value":"%d"}]},"position":{"x":400,"y":300}},
    {"id":"node_end_network","type":"end","data":{"label":"完成(网络)"},"position":{"x":100,"y":400}},
    {"id":"node_end_security","type":"end","data":{"label":"完成(安全)"},"position":{"x":400,"y":400}}
  ],
  "edges": [
    {"id":"edge_start_form","source":"node_start","target":"node_form","data":{}},
    {"id":"edge_form_gw","source":"node_form","target":"node_gw","data":{}},
    {"id":"edge_gw_network","source":"node_gw","target":"node_approve_network","data":{"condition":{"field":"form.request_kind","operator":"equals","value":"network_support","edge_id":"edge_gw_network"}}},
    {"id":"edge_gw_security","source":"node_gw","target":"node_approve_security","data":{"default":true}},
    {"id":"edge_approve_network_end","source":"node_approve_network","target":"node_end_network","data":{}},
    {"id":"edge_approve_security_end","source":"node_approve_security","target":"node_end_security","data":{}}
  ]
}`, ids.NetworkAdminPosID, ids.SecurityAdminPosID)

	return json.RawMessage(workflow)
}

// publishVPNService creates the full service configuration for VPN BDD tests:
// ServiceCatalog + Priority + ServiceDefinition with workflow JSON.
func publishVPNService(bc *bddContext, ids vpnWorkflowIDs) error {
	// 1. ServiceCatalog
	catalog := &ServiceCatalog{
		Name:     "VPN服务",
		Code:     "vpn",
		IsActive: true,
	}
	if err := bc.db.Create(catalog).Error; err != nil {
		return fmt.Errorf("create service catalog: %w", err)
	}

	// 2. Priority
	priority := &Priority{
		Name:     "普通",
		Code:     "normal",
		Value:    3,
		Color:    "#52c41a",
		IsActive: true,
	}
	if err := bc.db.Create(priority).Error; err != nil {
		return fmt.Errorf("create priority: %w", err)
	}
	bc.priority = priority

	// 3. ServiceDefinition
	svc := &ServiceDefinition{
		Name:         "VPN开通申请",
		Code:         "vpn-activation",
		CatalogID:    catalog.ID,
		EngineType:   "classic",
		WorkflowJSON: JSONField(buildVPNWorkflowJSON(ids)),
		IsActive:     true,
	}
	if err := bc.db.Create(svc).Error; err != nil {
		return fmt.Errorf("create service definition: %w", err)
	}
	bc.service = svc

	return nil
}
