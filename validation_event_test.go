package palimpsest

import (
	"context"
	"testing"
)

func TestValidateEventNodeAddedDuplicate(t *testing.T) {
	// 既存ノードと同じIDで追加しようとすると拒否される
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	g := ReplayLatest(log)

	ctx := context.Background()
	result := ValidateEvent(ctx, g, Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	if result.Valid {
		t.Fatalf("expected duplicate node to be invalid")
	}
}

func TestValidateEventNodeRemovedInUse(t *testing.T) {
	// 依存エッジが残っているノードは削除できない
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})
	g := ReplayLatest(log)

	ctx := context.Background()
	result := ValidateEvent(ctx, g, Event{Type: EventNodeRemoved, NodeID: "b"})
	if result.Valid {
		t.Fatalf("expected node removal with edges to be invalid")
	}
}

func TestValidateEventAttrUpdatedMissingNode(t *testing.T) {
	// 存在しないノードへの属性更新は拒否される
	g := NewGraph()
	ctx := context.Background()
	result := ValidateEvent(ctx, g, Event{Type: EventAttrUpdated, NodeID: "missing", Attrs: Attrs{"x": 1}})
	if result.Valid {
		t.Fatalf("expected attr update on missing node to be invalid")
	}
}

func TestValidateEventEdgeAddedMissingEndpoint(t *testing.T) {
	// 片側が存在しないエッジ追加は拒否される
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	g := ReplayLatest(log)

	ctx := context.Background()
	result := ValidateEvent(ctx, g, Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})
	if result.Valid {
		t.Fatalf("expected edge add with missing endpoint to be invalid")
	}
}

func TestValidateEventEntityToEntityRequiresRelation(t *testing.T) {
	// Entity間の直接エッジはRelationノード必須（N:Mを想定）
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "entity:product", NodeType: NodeEntity})
	log.Append(Event{Type: EventNodeAdded, NodeID: "entity:tag", NodeType: NodeEntity})

	g := ReplayLatest(log)
	ctx := context.Background()
	e := Event{Type: EventEdgeAdded, FromNode: "entity:product", ToNode: "entity:tag", Label: LabelUses}
	result := ValidateEvent(ctx, g, e)

	if result.Valid {
		t.Fatalf("expected entity-to-entity edge to be invalid")
	}
	found := false
	for _, err := range result.Errors {
		if err.Type == "relation_required" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected relation_required error, got %v", result.Errors)
	}
}

func TestValidateEventEdgeRemovedMissingEdge(t *testing.T) {
	// 存在しないエッジの削除は拒否される
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	g := ReplayLatest(log)

	ctx := context.Background()
	result := ValidateEvent(ctx, g, Event{Type: EventEdgeRemoved, FromNode: "a", ToNode: "b", Label: LabelUses})
	if result.Valid {
		t.Fatalf("expected edge removal of missing edge to be invalid")
	}
}

type testValidator struct{}

func (v testValidator) ValidateEvent(ctx context.Context, g *Graph, e Event) []ValidationError {
	return []ValidationError{
		{Type: "custom_rule", NodeID: e.NodeID, Message: "custom validation failed"},
	}
}

func TestValidateEventWithCustomValidators(t *testing.T) {
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	g := ReplayLatest(log)

	ctx := context.Background()
	res := ValidateEventWith(ctx, g, Event{Type: EventAttrUpdated, NodeID: "a", Attrs: Attrs{"x": 1}}, []Validator{testValidator{}})
	if res.Valid {
		t.Fatalf("expected validation to fail with custom validator")
	}
	if len(res.Errors) == 0 || res.Errors[0].Type != "custom_rule" {
		t.Fatalf("expected custom validation error")
	}
}
