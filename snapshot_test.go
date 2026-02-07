package palimpsest

import (
	"reflect"
	"testing"
)

func TestReplayFromSnapshotMatchesFullReplay(t *testing.T) {
	// snapshot + tail replay が full replay と一致することを確認
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})
	log.Append(Event{Type: EventAttrUpdated, NodeID: "a", Attrs: Attrs{"x": 1}})
	log.Append(Event{Type: EventAttrUpdated, NodeID: "b", Attrs: Attrs{"y": 2}})

	snap := SnapshotFromLog(log, 2)
	full := Replay(log, 4)
	fromSnap := ReplayFromSnapshot(snap, log, 4)

	if !reflect.DeepEqual(snapshotGraph(full), snapshotGraph(fromSnap)) {
		t.Fatalf("expected snapshot replay to match full replay")
	}
}

func TestReplayFromSnapshotBeforeRevisionFallsBack(t *testing.T) {
	// snapshotより前のrev指定は full replay にフォールバックする
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})

	snap := SnapshotFromLog(log, 2)
	full := Replay(log, 1)
	fromSnap := ReplayFromSnapshot(snap, log, 1)

	if !reflect.DeepEqual(snapshotGraph(full), snapshotGraph(fromSnap)) {
		t.Fatalf("expected fallback replay to match full replay")
	}
}

func TestSnapshotBaseGraphIsolation(t *testing.T) {
	// BaseGraphで得たGraphを変更してもsnapshotが壊れないことを確認
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})

	snap := SnapshotFromLog(log, 2)
	before := Replay(log, 2)

	bg := snap.BaseGraph()
	bg.addNode("x", NodeField, nil)

	after := ReplayFromSnapshot(snap, log, 2)
	if !reflect.DeepEqual(snapshotGraph(before), snapshotGraph(after)) {
		t.Fatalf("expected snapshot to remain immutable after BaseGraph mutation")
	}
}

func TestReplayFromSnapshotExactRevision(t *testing.T) {
	// toRevision == snapshot.Revision の境界で一致することを確認
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})

	snap := SnapshotFromLog(log, 2)
	full := Replay(log, 2)
	fromSnap := ReplayFromSnapshot(snap, log, 2)

	if !reflect.DeepEqual(snapshotGraph(full), snapshotGraph(fromSnap)) {
		t.Fatalf("expected replay at snapshot revision to match full replay")
	}
}

func TestSnapshotFromGraphIsolation(t *testing.T) {
	// SnapshotFromGraph が元Graphから独立していることを確認
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	g := ReplayLatest(log)

	before := snapshotGraph(g)
	snap := SnapshotFromGraph(g)

	g.addNode("c", NodeField, nil)

	if !reflect.DeepEqual(before, snapshotGraph(snap.BaseGraph())) {
		t.Fatalf("expected snapshot to be isolated from source graph mutations")
	}
}
