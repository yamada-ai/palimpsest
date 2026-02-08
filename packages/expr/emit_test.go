package expr

import (
	"testing"

	core "github.com/user/palimpsest"
)

func TestBuildDepEvents(t *testing.T) {
	s := &DepSummary{
		SelfID:      "expr:x",
		TargetField: "field:y",
		ExactDeps: []DepEntry{
			{NodeID: "field:a"},
		},
		SchemaDeps: []DepEntry{
			{NodeID: "entity:e"},
		},
	}
	events := BuildDepEvents(s)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Type != core.EventEdgeAdded || events[0].Label != core.LabelUses {
		t.Fatalf("expected EdgeAdded uses")
	}
	if events[2].FromNode != "expr:x" || events[2].ToNode != "field:y" {
		t.Fatalf("expected self -> target edge")
	}
}

func TestBuildDepEventsEmptyIDs(t *testing.T) {
	s := &DepSummary{SelfID: "", TargetField: "field:y"}
	if events := BuildDepEvents(s); events != nil {
		t.Fatalf("expected nil events for empty SelfID")
	}
}
