package engine

import "testing"

func TestEngineNodeClassificationAndTicketStatusMapping(t *testing.T) {
	if !IsAutoNode(NodeAction) || !IsAutoNode(NodeNotify) || !IsAutoNode(NodeExclusive) {
		t.Fatal("expected action/notify/exclusive to be auto nodes")
	}
	if IsAutoNode(NodeProcess) || IsAutoNode(NodeWait) {
		t.Fatal("human nodes must not be auto nodes")
	}

	if got := ticketStatusForDecisionActivity(NodeAction); got != TicketStatusExecutingAction {
		t.Fatalf("action ticket status = %s, want %s", got, TicketStatusExecutingAction)
	}
	if got := ticketStatusForDecisionActivity(NodeProcess); got != TicketStatusWaitingHuman {
		t.Fatalf("process ticket status = %s, want %s", got, TicketStatusWaitingHuman)
	}
	if got := ticketStatusForDecisionActivity(NodeParallel); got != TicketStatusDecisioning {
		t.Fatalf("parallel ticket status = %s, want %s", got, TicketStatusDecisioning)
	}
}
