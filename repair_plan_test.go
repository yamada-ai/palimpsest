package palimpsest

import (
	"context"
	"testing"
)

func TestRepairPlanOrdering(t *testing.T) {
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "field:a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "expr:x", NodeType: NodeExpression})
	log.Append(Event{Type: EventNodeAdded, NodeID: "field:b", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "form:f", NodeType: NodeForm})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "field:a", ToNode: "expr:x", Label: LabelUses})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "expr:x", ToNode: "field:b", Label: LabelDerives})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "field:b", ToNode: "form:f", Label: LabelUses})

	g := ReplayLatest(log)
	ctx := context.Background()
	e := Event{Type: EventAttrUpdated, NodeID: "field:a", Attrs: Attrs{"x": 1}}
	plan := ComputeRepairPlan(ctx, g, e)

	if len(plan.Suggestions) == 0 {
		t.Fatalf("expected suggestions")
	}
	// First suggestion should be expression (critical)
	if plan.Suggestions[0].NodeType != NodeExpression {
		t.Fatalf("expected first suggestion to be expression")
	}
}

func TestRepairPlanExcludesSeeds(t *testing.T) {
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "field:a", NodeType: NodeField})
	g := ReplayLatest(log)

	ctx := context.Background()
	e := Event{Type: EventAttrUpdated, NodeID: "field:a", Attrs: Attrs{"x": 1}}
	plan := ComputeRepairPlan(ctx, g, e)
	if len(plan.Suggestions) != 0 {
		t.Fatalf("expected no suggestions when only seed is impacted")
	}
}
