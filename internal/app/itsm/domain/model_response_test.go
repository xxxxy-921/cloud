package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestJSONFieldAndResponseMappings(t *testing.T) {
	var field JSONField
	if err := field.Scan(`{"mode":"smart"}`); err != nil {
		t.Fatalf("Scan string: %v", err)
	}
	if string(field) != `{"mode":"smart"}` {
		t.Fatalf("field = %s", field)
	}
	data, err := field.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) != `{"mode":"smart"}` {
		t.Fatalf("MarshalJSON = %s", data)
	}

	now := time.Unix(1710000000, 0)
	catalog := &ServiceCatalog{Name: "VPN", Code: "vpn", Description: "desc", Icon: "key", SortOrder: 1, IsActive: true}
	catalog.ID = 1
	catalog.CreatedAt = now
	catalog.UpdatedAt = now
	if catalog.ToResponse().Code != "vpn" {
		t.Fatalf("unexpected catalog response: %+v", catalog.ToResponse())
	}

	checkedAt := now.Add(time.Minute)
	service := &ServiceDefinition{
		Name:                "VPN access",
		Code:                "vpn-access",
		CatalogID:           1,
		EngineType:          "smart",
		CollaborationSpec:   "审批后开通",
		PublishHealthStatus: "warn",
		PublishHealthItems:  JSONField(`[{"key":"agent","label":"Agent","status":"warn","message":"missing"}]`),
		PublishHealthCheckedAt: &checkedAt,
		IsActive:            true,
	}
	service.ID = 2
	resp := service.ToResponse()
	if resp.PublishHealthCheck == nil || resp.PublishHealthCheck.Status != "warn" {
		t.Fatalf("unexpected service publish health: %+v", resp.PublishHealthCheck)
	}
	listResp := service.ToListItemResponse()
	if listResp.PublishHealthCheck == nil || listResp.PublishHealthCheck.ServiceID != service.ID {
		t.Fatalf("unexpected list publish health: %+v", listResp.PublishHealthCheck)
	}

	priority := &Priority{Name: "P1", Code: "P1", Value: 1, Color: "#ef4444", IsActive: true}
	if priority.ToResponse().Code != "P1" {
		t.Fatalf("unexpected priority response: %+v", priority.ToResponse())
	}

	sla := &SLATemplate{Name: "Standard", Code: "std", ResponseMinutes: 5, ResolutionMinutes: 30, IsActive: true}
	if sla.ToResponse().ResponseMinutes != 5 {
		t.Fatalf("unexpected sla response: %+v", sla.ToResponse())
	}

	rule := &EscalationRule{SLAID: 1, TriggerType: "response_timeout", Level: 1, WaitMinutes: 5, ActionType: "notify", TargetConfig: JSONField(`{"channelId":1}`), IsActive: true}
	if rule.ToResponse().ActionType != "notify" {
		t.Fatalf("unexpected escalation rule response: %+v", rule.ToResponse())
	}

	doc := &ServiceKnowledgeDocument{ServiceID: 2, FileName: "vpn.md", FileSize: 128, FileType: "text/markdown", ParseStatus: "completed"}
	doc.CreatedAt = now
	if doc.ToResponse().CreatedAt == "" {
		t.Fatalf("unexpected knowledge doc response: %+v", doc.ToResponse())
	}

	ticket := &Ticket{
		Code:           "ITSM-1",
		Title:          "VPN 开通",
		ServiceID:      1,
		EngineType:     "classic",
		Status:         TicketStatusCompleted,
		Outcome:        TicketOutcomeApproved,
		PriorityID:     1,
		RequesterID:    7,
		Source:         TicketSourceCatalog,
		FormData:       JSONField(`{"account":"alice"}`),
		WorkflowJSON:   JSONField(`{"nodes":[]}`),
		SLAStatus:      SLAStatusOnTrack,
		AIFailureCount: 2,
	}
	if !ticket.IsTerminal() {
		t.Fatal("expected completed ticket to be terminal")
	}
	ticketResp := ticket.ToResponse()
	if ticketResp.StatusLabel == "" || ticketResp.Code != "ITSM-1" {
		t.Fatalf("unexpected ticket response: %+v", ticketResp)
	}

	var decoded map[string]string
	if err := json.Unmarshal(ticketResp.FormData, &decoded); err != nil {
		t.Fatalf("unmarshal form data: %v", err)
	}
	if decoded["account"] != "alice" {
		t.Fatalf("decoded form data = %+v", decoded)
	}
}
