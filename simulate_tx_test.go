package palimpsest

import (
	"context"
	"reflect"
	"sort"
	"testing"
)

type txSnapshot struct {
	Nodes map[NodeID]Node
}

func snapshotTxGraph(g *Graph) txSnapshot {
	ids := g.AllNodeIDs()
	nodes := make(map[NodeID]Node, len(ids))
	for _, id := range ids {
		node := g.GetNode(id)
		if node == nil {
			continue
		}
		sortEdgesTx(node.Outgoing)
		sortEdgesTx(node.Incoming)
		nodes[id] = *node
	}
	return txSnapshot{Nodes: nodes}
}

func TestSimulateTxApplyRollback(t *testing.T) {
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	g := ReplayLatest(log)

	before := snapshotTxGraph(g)
	ctx := context.Background()
	events := []Event{
		{Type: EventAttrUpdated, NodeID: "a", Attrs: Attrs{"x": VNumber(1)}},
		{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses},
	}

	res := SimulateTx(ctx, g, events)
	if !res.Applied {
		t.Fatalf("expected tx to be applied")
	}
	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error)
	}
	if res.AfterRevision != res.BeforeRevision+len(events) {
		t.Fatalf("expected AfterRevision to advance by event count")
	}

	after := snapshotTxGraph(g)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("expected graph to be unchanged after rollback")
	}
}

func TestSimulateTxPreValidateRejects(t *testing.T) {
	g := NewGraph()
	ctx := context.Background()
	events := []Event{
		{Type: EventNodeRemoved, NodeID: "missing"},
		{Type: EventAttrUpdated, NodeID: "missing", Attrs: Attrs{"x": VNumber(1)}},
	}

	res := SimulateTx(ctx, g, events)
	if res.Applied {
		t.Fatalf("expected tx to be rejected")
	}
	if res.AfterRevision != res.BeforeRevision {
		t.Fatalf("expected AfterRevision to stay on rejection")
	}
	if res.PostImpact != nil || res.PostValidate != nil {
		t.Fatalf("expected no post results on rejection")
	}
}

func TestSimulateTxAllowsIntraTxDependencies(t *testing.T) {
	g := NewGraph()
	ctx := context.Background()
	events := []Event{
		{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField},
		{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField},
		{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses},
	}

	res := SimulateTx(ctx, g, events)
	if !res.Applied {
		t.Fatalf("expected tx to be applied with intra-tx dependencies")
	}
	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error)
	}
}

func TestSimulateTxImpactPrePost(t *testing.T) {
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})
	g := ReplayLatest(log)

	ctx := context.Background()
	events := []Event{{Type: EventAttrUpdated, NodeID: "a", Attrs: Attrs{"x": VNumber(1)}}}
	res := SimulateTx(ctx, g, events)

	if res.PreImpact == nil || res.PostImpact == nil {
		t.Fatalf("expected pre/post impact results")
	}
	if !res.PreImpact.Impacted["a"] {
		t.Fatalf("expected pre-impact to include seed node")
	}
	if !res.PostImpact.Impacted["a"] {
		t.Fatalf("expected post-impact to include seed node")
	}
}

func sortEdgesTx(edges []Edge) {
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
