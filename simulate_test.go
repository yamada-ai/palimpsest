package palimpsest

import (
	"context"
	"testing"
)

func TestSimulateEventNodeAddedPrePost(t *testing.T) {
	// NodeAdded は PreImpact が空になりやすく、PostImpact に影響が出る
	g := NewGraph()
	ctx := context.Background()

	e := Event{Type: EventNodeAdded, NodeID: "n1", NodeType: NodeField}
	res := SimulateEvent(ctx, g, e)

	if res.PreImpact == nil || res.PreValidate == nil {
		t.Fatalf("expected pre results to be present")
	}
	if res.Applied != true {
		t.Fatalf("expected event to be applied")
	}
	if res.PostImpact == nil || res.PostValidate == nil {
		t.Fatalf("expected post results to be present")
	}
	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error)
	}
	if res.PreImpact.Impacted["n1"] {
		t.Fatalf("expected pre-impact to be empty for NodeAdded")
	}
	if !res.PostImpact.Impacted["n1"] {
		t.Fatalf("expected post-impact to include newly added node")
	}
}

func TestSimulateEventNodeRemovedPrePost(t *testing.T) {
	// NodeRemoved は pre に影響が出て、適用後は消える
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	g := ReplayLatest(log)

	ctx := context.Background()
	e := Event{Type: EventNodeRemoved, NodeID: "a"}
	res := SimulateEvent(ctx, g, e)

	if res.PreImpact == nil || res.PreValidate == nil {
		t.Fatalf("expected pre results to be present")
	}
	if res.Applied != true {
		t.Fatalf("expected event to be applied")
	}
	if res.PostImpact == nil || res.PostValidate == nil {
		t.Fatalf("expected post results to be present")
	}
	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error)
	}
	if !res.PreImpact.Impacted["a"] {
		t.Fatalf("expected pre-impact to include removed node")
	}
	if res.PostImpact.Impacted["a"] {
		t.Fatalf("expected post-impact to not include removed node")
	}
}

func TestSimulateEventEdgeRemovedPrePost(t *testing.T) {
	// EdgeRemoved は pre で理由が見え、post は影響が減る
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})
	g := ReplayLatest(log)

	ctx := context.Background()
	e := Event{Type: EventEdgeRemoved, FromNode: "a", ToNode: "b", Label: LabelUses}
	res := SimulateEvent(ctx, g, e)

	if res.PreImpact == nil || res.PreValidate == nil {
		t.Fatalf("expected pre results to be present")
	}
	if res.Applied != true {
		t.Fatalf("expected event to be applied")
	}
	if res.PostImpact == nil || res.PostValidate == nil {
		t.Fatalf("expected post results to be present")
	}
	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error)
	}
	if !res.PreImpact.Impacted["b"] {
		t.Fatalf("expected pre-impact to include consumer node")
	}
}

func TestSimulateEventInvalidPreValidate(t *testing.T) {
	// PreValidate で拒否されるイベントは適用されない
	g := NewGraph()
	ctx := context.Background()

	e := Event{Type: EventNodeRemoved, NodeID: "missing"}
	res := SimulateEvent(ctx, g, e)

	if res.Applied {
		t.Fatalf("expected event to be rejected")
	}
	if res.AfterRevision != res.BeforeRevision {
		t.Fatalf("expected AfterRevision to equal BeforeRevision on rejection")
	}
	if res.PostImpact != nil || res.PostValidate != nil {
		t.Fatalf("expected no post results on rejection")
	}
}

func TestSimulateEventRejectedPreValidate(t *testing.T) {
	// PreValidate で拒否された場合は適用されず、AfterRevision は進まない
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	g := ReplayLatest(log)

	ctx := context.Background()
	e := Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "missing", Label: LabelUses}
	res := SimulateEvent(ctx, g, e)

	if res.Applied {
		t.Fatalf("expected event to be rejected")
	}
	if res.PostImpact != nil || res.PostValidate != nil {
		t.Fatalf("expected post results to be nil for rejected event")
	}
	if res.AfterRevision != res.BeforeRevision {
		t.Fatalf("expected AfterRevision to stay at BeforeRevision when not applied")
	}
}
