package model

import (
	"testing"
	"time"
)

func TestAuditLogToResponse_MapsAllFields(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	userID := uint(42)
	detail := "detailed info"
	log := AuditLog{
		ID:         1,
		CreatedAt:  now,
		Category:   AuditCategoryOperation,
		UserID:     &userID,
		Username:   "alice",
		Action:     "create_user",
		Resource:   "user",
		ResourceID: "42",
		Summary:    "created user alice",
		Level:      AuditLevelInfo,
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Detail:     &detail,
	}

	resp := log.ToResponse()

	if resp.ID != log.ID {
		t.Fatalf("expected ID %d, got %d", log.ID, resp.ID)
	}
	if !resp.CreatedAt.Equal(log.CreatedAt) {
		t.Fatalf("expected CreatedAt %v, got %v", log.CreatedAt, resp.CreatedAt)
	}
	if resp.Category != log.Category {
		t.Fatalf("expected Category %s, got %s", log.Category, resp.Category)
	}
	if resp.UserID == nil || *resp.UserID != *log.UserID {
		t.Fatalf("expected UserID %v, got %v", log.UserID, resp.UserID)
	}
	if resp.Username != log.Username {
		t.Fatalf("expected Username %s, got %s", log.Username, resp.Username)
	}
	if resp.Action != log.Action {
		t.Fatalf("expected Action %s, got %s", log.Action, resp.Action)
	}
	if resp.Resource != log.Resource {
		t.Fatalf("expected Resource %s, got %s", log.Resource, resp.Resource)
	}
	if resp.ResourceID != log.ResourceID {
		t.Fatalf("expected ResourceID %s, got %s", log.ResourceID, resp.ResourceID)
	}
	if resp.Summary != log.Summary {
		t.Fatalf("expected Summary %s, got %s", log.Summary, resp.Summary)
	}
	if resp.Level != log.Level {
		t.Fatalf("expected Level %s, got %s", log.Level, resp.Level)
	}
	if resp.IPAddress != log.IPAddress {
		t.Fatalf("expected IPAddress %s, got %s", log.IPAddress, resp.IPAddress)
	}
	if resp.UserAgent != log.UserAgent {
		t.Fatalf("expected UserAgent %s, got %s", log.UserAgent, resp.UserAgent)
	}
	if resp.Detail == nil || *resp.Detail != *log.Detail {
		t.Fatalf("expected Detail %v, got %v", log.Detail, resp.Detail)
	}
}
