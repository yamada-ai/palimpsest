package palimpsest

import (
	"reflect"
	"sort"
	"testing"
)

type graphSnapshot struct {
	Nodes map[NodeID]Node
}

func snapshotGraph(g *Graph) graphSnapshot {
	ids := g.AllNodeIDs()
	nodes := make(map[NodeID]Node, len(ids))
	for _, id := range ids {
		node := g.GetNode(id)
		if node == nil {
			continue
		}
		sortEdges(node.Outgoing)
		sortEdges(node.Incoming)
		nodes[id] = *node
	}
	return graphSnapshot{Nodes: nodes}
}

func sortEdges(edges []Edge) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Label < edges[j].Label
	})
}

func TestApplyRollbackNoopNodeAdded(t *testing.T) {
	// NodeAdded の apply + rollback で元の状態に戻ることを確認
	g := NewGraph()
	before := snapshotGraph(g)

	delta, err := ApplyEvent(g, Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if err := RollbackDelta(g, delta); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	after := snapshotGraph(g)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("expected graph to be unchanged after rollback")
	}
}

func TestApplyRollbackNoopNodeRemoved(t *testing.T) {
	// NodeRemoved の apply + rollback で元の状態に戻ることを確認
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField, Attrs: Attrs{"x": 1}})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})
	g := ReplayLatest(log)

	before := snapshotGraph(g)
	delta, err := ApplyEvent(g, Event{Type: EventNodeRemoved, NodeID: "a"})
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if err := RollbackDelta(g, delta); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	after := snapshotGraph(g)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("expected graph to be unchanged after rollback")
	}
}

func TestApplyRollbackNoopAttrUpdated(t *testing.T) {
	// AttrUpdated の apply + rollback で元の状態に戻ることを確認
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField, Attrs: Attrs{"x": 1, "y": 2}})
	g := ReplayLatest(log)

	before := snapshotGraph(g)
	delta, err := ApplyEvent(g, Event{Type: EventAttrUpdated, NodeID: "a", Attrs: Attrs{"x": 3, "y": nil}})
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if err := RollbackDelta(g, delta); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	after := snapshotGraph(g)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("expected graph to be unchanged after rollback")
	}
}

func TestApplyRollbackNoopEdgeRemoved(t *testing.T) {
	// EdgeRemoved の apply + rollback で元の状態に戻ることを確認
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})
	g := ReplayLatest(log)

	before := snapshotGraph(g)
	delta, err := ApplyEvent(g, Event{Type: EventEdgeRemoved, FromNode: "a", ToNode: "b", Label: LabelUses})
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if err := RollbackDelta(g, delta); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	after := snapshotGraph(g)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("expected graph to be unchanged after rollback")
	}
}

func TestApplyEventRejectsDuplicateEdge(t *testing.T) {
	// 重複エッジはApplyEventで拒否される
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})
	g := ReplayLatest(log)

	_, err := ApplyEvent(g, Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})
	if err == nil {
		t.Fatalf("expected duplicate edge to be rejected")
	}
}
