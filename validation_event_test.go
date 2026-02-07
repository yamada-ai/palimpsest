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
