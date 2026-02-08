package palimpsest

import (
	"context"
	"reflect"
	"testing"
)

func TestSandboxSimulateEventIsolation(t *testing.T) {
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})

	snap := SnapshotFromLog(log, log.Len()-1)
	before := snapshotGraph(snap.BaseGraph())

	sb := NewSandbox(snap, log, log.Len()-1)
	ctx := context.Background()
	res := sb.SimulateEvent(ctx, Event{Type: EventAttrUpdated, NodeID: "a", Attrs: Attrs{"x": VNumber(1)}})
	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error)
	}

	after := snapshotGraph(snap.BaseGraph())
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("expected snapshot to remain unchanged after simulation")
	}
}

func TestSandboxSimulateTxIsolation(t *testing.T) {
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})

	snap := SnapshotFromLog(log, log.Len()-1)
	before := snapshotGraph(snap.BaseGraph())

	sb := NewSandbox(snap, log, log.Len()-1)
	ctx := context.Background()
	events := []Event{
		{Type: EventAttrUpdated, NodeID: "a", Attrs: Attrs{"x": VNumber(1)}},
		{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses},
	}
	res := sb.SimulateTx(ctx, events)
	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error)
	}

	after := snapshotGraph(snap.BaseGraph())
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("expected snapshot to remain unchanged after tx simulation")
	}
}

func TestSandboxSnapshotAheadFallback(t *testing.T) {
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	g := ReplayLatest(log)
	snap := SnapshotFromGraph(g)

	// Make log shorter than snapshot revision by using an empty log.
	shortLog := NewEventLog()
	sb := NewSandbox(snap, shortLog, 0)
	if sb.BuildGraph() == nil {
		t.Fatalf("expected fallback replay to return a graph")
	}
}

func TestSandboxNilLog(t *testing.T) {
	sb := NewSandbox(nil, nil, 0)
	if sb.BuildGraph() != nil {
		t.Fatalf("expected nil graph when log is nil")
	}

	ctx := context.Background()
	if res := sb.SimulateEvent(ctx, Event{Type: EventAttrUpdated, NodeID: "x"}); res.Error == nil {
		t.Fatalf("expected error when simulating with nil graph")
	}
	if res := sb.SimulateTx(ctx, []Event{{Type: EventAttrUpdated, NodeID: "x"}}); res.Error == nil {
		t.Fatalf("expected error when simulating tx with nil graph")
	}
}
