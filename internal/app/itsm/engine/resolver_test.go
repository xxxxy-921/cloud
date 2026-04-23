package engine

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestParticipantResolverRequesterReturnsTicketRequester(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&ticketModel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	ticket := ticketModel{RequesterID: 42, Status: "in_progress"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	ids, err := NewParticipantResolver(nil).Resolve(db, ticket.ID, Participant{Type: "requester"})
	if err != nil {
		t.Fatalf("resolve requester: %v", err)
	}
	if len(ids) != 1 || ids[0] != 42 {
		t.Fatalf("expected requester id 42, got %+v", ids)
	}
}
