package support

import (
	"fmt"
	"net/http"
	"testing"
)

func TestContextPendingApprovalFlow(t *testing.T) {
	ctx := NewContext()
	if err := ctx.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if err := ctx.EnsureDefaultActors(); err != nil {
		t.Fatalf("EnsureDefaultActors: %v", err)
	}
	if err := ctx.SeedAssignedSmartTicket("网络管理员"); err != nil {
		t.Fatalf("SeedAssignedSmartTicket: %v", err)
	}

	resp, err := ctx.Client.Do("网络管理员", http.MethodGet, "/api/v1/itsm/tickets/approvals/pending", nil)
	if err != nil {
		t.Fatalf("query pending approvals: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pending approvals status=%d body=%s", resp.StatusCode, resp.RawBody)
	}
	data, err := ctx.LastDataObject()
	if err != nil {
		t.Fatalf("LastDataObject: %v", err)
	}
	items, err := DecodeField[[]map[string]any](data, "items")
	if err != nil {
		t.Fatalf("DecodeField items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one pending approval item, got %d", len(items))
	}

	resp, err = ctx.Client.Do("网络管理员", http.MethodPost, "/api/v1/itsm/tickets/"+itoa(ctx.CurrentTicketID)+"/claim", map[string]any{"activityId": ctx.CurrentActivityID})
	if err != nil {
		t.Fatalf("claim pending ticket: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("claim status=%d body=%s", resp.StatusCode, resp.RawBody)
	}

	ticketData, err := ctx.LoadTicketViaAPI("管理员")
	if err != nil {
		t.Fatalf("LoadTicketViaAPI: %v", err)
	}
	if status, err := DecodeField[string](ticketData, "status"); err != nil || status == "" {
		t.Fatalf("DecodeField status=%q err=%v", status, err)
	}
}

func TestOrgResolverAndUserProviderExposeSeededOrgData(t *testing.T) {
	ctx := NewContext()
	if err := ctx.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if err := ctx.EnsureDefaultActors(); err != nil {
		t.Fatalf("EnsureDefaultActors: %v", err)
	}

	actor := ctx.Actors["网络管理员"]
	if actor == nil {
		t.Fatal("expected 网络管理员 actor")
	}

	resolver := &OrgResolver{db: ctx.DB}
	if ids, err := resolver.GetUserPositionIDs(actor.User.ID); err != nil || len(ids) != 1 {
		t.Fatalf("GetUserPositionIDs ids=%v err=%v", ids, err)
	}
	if ids, err := resolver.GetUserDepartmentIDs(actor.User.ID); err != nil || len(ids) != 1 {
		t.Fatalf("GetUserDepartmentIDs ids=%v err=%v", ids, err)
	}
	if ids, err := resolver.FindUsersByPositionCode("network_admin"); err != nil || len(ids) == 0 {
		t.Fatalf("FindUsersByPositionCode ids=%v err=%v", ids, err)
	}
	if ids, err := resolver.FindUsersByDepartmentCode("it"); err != nil || len(ids) == 0 {
		t.Fatalf("FindUsersByDepartmentCode ids=%v err=%v", ids, err)
	}
	if ids, err := resolver.FindUsersByPositionAndDepartment("network_admin", "it"); err != nil || len(ids) == 0 {
		t.Fatalf("FindUsersByPositionAndDepartment ids=%v err=%v", ids, err)
	}
	if managerID, err := resolver.FindManagerByUserID(actor.User.ID); err != nil || managerID != 0 {
		t.Fatalf("FindManagerByUserID managerID=%d err=%v", managerID, err)
	}

	provider := &UserProvider{db: ctx.DB}
	users, err := provider.ListActiveUsers()
	if err != nil {
		t.Fatalf("ListActiveUsers: %v", err)
	}
	if len(users) < 4 {
		t.Fatalf("expected seeded active users, got %+v", users)
	}
	found := false
	for _, user := range users {
		if user.Name == actor.Username {
			found = user.Position == "network_admin" && user.Department == "it"
			break
		}
	}
	if !found {
		t.Fatalf("expected network admin candidate with org info, got %+v", users)
	}
}

func itoa(v uint) string {
	return fmt.Sprintf("%d", v)
}
