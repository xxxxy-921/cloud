package ticket

import (
	"testing"

	"github.com/samber/do/v2"

	. "metis/internal/app/itsm/domain"
	"metis/internal/database"
)

func TestTimelineServiceRecordPersistsTimelineEntry(t *testing.T) {
	db := newTestDB(t)
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.Provide(injector, NewTimelineRepo)
	do.Provide(injector, NewTimelineService)

	service := do.MustInvoke[*TimelineService](injector)
	details := JSONField(`{"source":"system"}`)
	if err := service.Record(42, 7, "ticket_created", "工单已创建", details); err != nil {
		t.Fatalf("record timeline: %v", err)
	}

	rows, err := service.ListByTicket(42)
	if err != nil {
		t.Fatalf("list by ticket: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 timeline row, got %+v", rows)
	}
	if rows[0].OperatorID != 7 || rows[0].EventType != "ticket_created" || rows[0].Message != "工单已创建" || string(rows[0].Details) != string(details) {
		t.Fatalf("unexpected persisted timeline row: %+v", rows[0])
	}
}
