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

func TestRepairPlanTxCascadeDelete(t *testing.T) {
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "field:a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "field:b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "field:a", ToNode: "field:b", Label: LabelUses})

	g := ReplayLatest(log)
	ctx := context.Background()
	e := Event{Type: EventNodeRemoved, NodeID: "field:b"}
	plan := ComputeRepairPlanTx(ctx, g, e)

	if len(plan.Actions) != 1 {
		t.Fatalf("expected one cascade action")
	}
	if len(plan.Actions[0].Proposals) < 2 {
		t.Fatalf("expected edge removal + node removal proposals")
	}
	if !plan.Actions[0].Proposals[0].Applyable {
		t.Fatalf("expected cascade proposals to be applyable")
	}
}

func TestRepairPlanTxRelationImpact(t *testing.T) {
	// リレーション属性の変更が下流に影響し、修復提案が出ることを確認する
	log := buildRelationLog()
	g := ReplayLatest(log)

	ctx := context.Background()
	e := Event{Type: EventAttrUpdated, NodeID: "field:product_tag.quantity"}
	plan := ComputeRepairPlanTx(ctx, g, e)

	found := false
	for _, a := range plan.Actions {
		if a.NodeID == "list:tagged_products" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected list:tagged_products to be suggested, got %v", plan.Actions)
	}
}
