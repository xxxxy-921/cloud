package steps

import (
	"net/http"
	"strconv"
	"testing"

	"metis/internal/app/itsm/bdd/support"
)

func TestAssertPendingListContainsAcrossClaimLifecycle(t *testing.T) {
	ctx := support.NewContext()
	if err := ctx.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if err := ctx.EnsureDefaultActors(); err != nil {
		t.Fatalf("EnsureDefaultActors: %v", err)
	}
	if err := ctx.SeedAssignedSmartTicket("网络管理员"); err != nil {
		t.Fatalf("SeedAssignedSmartTicket: %v", err)
	}
	if _, err := ctx.Client.Do("网络管理员", http.MethodGet, "/api/v1/itsm/tickets/approvals/pending", nil); err != nil {
		t.Fatalf("query pending approvals: %v", err)
	}
	if err := assertPendingListContains(ctx, true); err != nil {
		t.Fatalf("assert pending contains: %v", err)
	}

	if _, err := ctx.Client.Do("网络管理员", http.MethodPost, "/api/v1/itsm/tickets/"+strconv.FormatUint(uint64(ctx.CurrentTicketID), 10)+"/claim", map[string]any{"activityId": ctx.CurrentActivityID}); err != nil {
		t.Fatalf("claim ticket: %v", err)
	}
	if _, err := ctx.Client.Do("网络管理员", http.MethodGet, "/api/v1/itsm/tickets/approvals/pending", nil); err != nil {
		t.Fatalf("query pending approvals after claim: %v", err)
	}
	if err := assertPendingListContains(ctx, false); err != nil {
		t.Fatalf("assert pending removed: %v", err)
	}
}
