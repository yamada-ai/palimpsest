package palimpsest

import (
	"context"
	"testing"
)

func TestRepairPlanTxProposals(t *testing.T) {
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "field:a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "expr:x", NodeType: NodeExpression})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "field:a", ToNode: "expr:x", Label: LabelUses})

	g := ReplayLatest(log)
	ctx := context.Background()
	e := Event{Type: EventAttrUpdated, NodeID: "field:a", Attrs: Attrs{"x": 1}}
	plan := ComputeRepairPlanTx(ctx, g, e)

	if len(plan.Actions) == 0 {
		t.Fatalf("expected actions")
	}
	if len(plan.Actions[0].Proposals) == 0 {
		t.Fatalf("expected proposals for top action")
	}
	if plan.Actions[0].Proposals[0].Applyable {
		t.Fatalf("expected proposals to be non-applyable placeholders")
	}
}
