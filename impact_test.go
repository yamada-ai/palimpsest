package palimpsest

import (
	"context"
	"testing"
)

// TestBasicReplayAndImpact tests the core flow:
// 1. Build event log
// 2. Replay to get graph
// 3. Compute impact from event
// 4. Verify evidence paths
func TestBasicReplayAndImpact(t *testing.T) {
	// 基本フロー: EventLog → Replay → Impact → Evidence の整合性を確認する
	// Build a simple dependency graph:
	// Entity "Order" -> Field "total" -> Expression "tax_calc" -> Field "tax"
	//                                                          -> Field "subtotal"
	log := NewEventLog()

	// Add nodes
	log.Append(Event{Type: EventNodeAdded, NodeID: "order", NodeType: NodeEntity, Attrs: Attrs{"name": "Order"}})
	log.Append(Event{Type: EventNodeAdded, NodeID: "total", NodeType: NodeField, Attrs: Attrs{"name": "total", "type": "number"}})
	log.Append(Event{Type: EventNodeAdded, NodeID: "tax_calc", NodeType: NodeExpression, Attrs: Attrs{"formula": "subtotal * 0.1"}})
	log.Append(Event{Type: EventNodeAdded, NodeID: "tax", NodeType: NodeField, Attrs: Attrs{"name": "tax", "type": "number"}})
	log.Append(Event{Type: EventNodeAdded, NodeID: "subtotal", NodeType: NodeField, Attrs: Attrs{"name": "subtotal", "type": "number"}})

	// Add edges (provider → consumer)
	log.Append(Event{Type: EventEdgeAdded, FromNode: "order", ToNode: "total", Label: LabelUses})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "total", ToNode: "tax_calc", Label: LabelUses})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "tax_calc", ToNode: "tax", Label: LabelDerives})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "tax_calc", ToNode: "subtotal", Label: LabelUses})

	// Replay to build graph
	g := ReplayLatest(log)

	if g.NodeCount() != 5 {
		t.Errorf("expected 5 nodes, got %d", g.NodeCount())
	}

	// Simulate changing the "total" field
	changeEvent := Event{Type: EventAttrUpdated, NodeID: "total", Attrs: Attrs{"type": "currency"}}
	seeds := changeEvent.ImpactSeeds()

	if len(seeds) != 1 || seeds[0] != "total" {
		t.Errorf("expected seeds [total], got %v", seeds)
	}

	// Compute impact
	ctx := context.Background()
	result := ComputeImpact(ctx, g, seeds)

	// Verify impacted nodes: total -> tax_calc -> tax, subtotal
	expectedImpacted := map[NodeID]bool{
		"total":    true,
		"tax_calc": true,
		"tax":      true,
		"subtotal": true,
	}

	if len(result.Impacted) != len(expectedImpacted) {
		t.Errorf("expected %d impacted nodes, got %d", len(expectedImpacted), len(result.Impacted))
	}

	for node := range expectedImpacted {
		if !result.Impacted[node] {
			t.Errorf("expected node %s to be impacted", node)
		}
	}

	// Verify "order" is NOT impacted (it's upstream)
	if result.Impacted["order"] {
		t.Errorf("node 'order' should not be impacted (it's a provider, not consumer)")
	}

	// Verify evidence path for "tax"
	taxEvidence, ok := result.EvidencePath("tax")
	if !ok {
		t.Fatalf("expected evidence path for tax")
	}
	expectedPath := []NodeID{"total", "tax_calc", "tax"}
	if len(taxEvidence.Path) != len(expectedPath) {
		t.Errorf("expected path length %d, got %d", len(expectedPath), len(taxEvidence.Path))
	} else {
		for i, node := range expectedPath {
			if taxEvidence.Path[i] != node {
				t.Errorf("expected path[%d] = %s, got %s", i, node, taxEvidence.Path[i])
			}
		}
	}

	// Test explain
	explanation := result.Explain("tax")
	t.Logf("Explanation for 'tax': %s", explanation)
	if explanation == "not impacted" {
		t.Error("expected explanation for impacted node")
	}
}

func TestEdgeSeedsForControlLabel(t *testing.T) {
	// controls の場合は両端がImpact Seedsに含まれる
	// When label is "controls", both endpoints should be in ImpactSeeds
	e := Event{
		Type:     EventEdgeAdded,
		FromNode: "role",
		ToNode:   "form",
		Label:    LabelControls,
	}

	seeds := e.ImpactSeeds()
	if len(seeds) != 2 {
		t.Errorf("expected 2 seeds for controls edge, got %d", len(seeds))
	}

	seedSet := make(map[NodeID]bool)
	for _, s := range seeds {
		seedSet[s] = true
	}
	if !seedSet["role"] || !seedSet["form"] {
		t.Errorf("expected seeds [role, form], got %v", seeds)
	}
}

func TestEdgeSeedsForUsesLabel(t *testing.T) {
	// uses の場合は consumer(ToNode) だけがImpact Seedsに含まれる
	// When label is "uses", only ToNode should be in ImpactSeeds
	e := Event{
		Type:     EventEdgeAdded,
		FromNode: "field_a",
		ToNode:   "field_b",
		Label:    LabelUses,
	}

	seeds := e.ImpactSeeds()
	if len(seeds) != 1 {
		t.Errorf("expected 1 seed for uses edge, got %d", len(seeds))
	}
	if seeds[0] != "field_b" {
		t.Errorf("expected seed [field_b], got %v", seeds)
	}

	// ValidationSeeds should return both
	vseeds := e.ValidationSeeds()
	if len(vseeds) != 2 {
		t.Errorf("expected 2 validation seeds, got %d", len(vseeds))
	}
}

func TestCancellation(t *testing.T) {
	// ctxキャンセルでImpact計算が中断される
	// Build a larger graph to test cancellation
	log := NewEventLog()

	// Create a chain: n0 -> n1 -> n2 -> ... -> n99
	for i := 0; i < 100; i++ {
		log.Append(Event{
			Type:     EventNodeAdded,
			NodeID:   NodeID(string(rune('a' + i%26)) + string(rune('0'+i/26))),
			NodeType: NodeField,
		})
	}
	// Note: simplified - would need proper ID generation for real test

	g := ReplayLatest(log)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := ComputeImpact(ctx, g, []NodeID{"a0"})
	if !result.Cancelled {
		t.Error("expected computation to be cancelled")
	}
}

func TestValidationDanglingEdge(t *testing.T) {
	// Dangling edge 検出の基本挙動を確認する
	// Test 1: Normal graph should be valid
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "a", ToNode: "b", Label: LabelUses})

	g := ReplayLatest(log)

	ctx := context.Background()
	result := Validate(ctx, g)
	if !result.Valid {
		t.Errorf("expected valid graph, got errors: %v", result.Errors)
	}

	// Test 2: removeNode cleans up edges (no dangling)
	// This verifies our design is correct
	g.removeNode("b")
	result = Validate(ctx, g)
	if !result.Valid {
		t.Errorf("expected valid graph after clean removeNode, got errors: %v", result.Errors)
	}

	// Test 3: Force a dangling edge by directly manipulating internal state
	// This simulates corruption or bug in replay
	g2 := NewGraph()
	g2.addNode("x", NodeField, nil)
	g2.addNode("y", NodeField, nil)
	g2.addEdge("x", "y", LabelUses)
	// Directly delete from map without cleanup (simulating corruption)
	g2.mu.Lock()
	delete(g2.nodes, "y")
	g2.mu.Unlock()

	result = Validate(ctx, g2)
	if result.Valid {
		t.Error("expected invalid graph with dangling edge")
	}
	if len(result.Errors) == 0 {
		t.Error("expected at least one validation error")
	}
	t.Logf("Detected errors: %+v", result.Errors)
}

func TestTransactionMarker(t *testing.T) {
	// TxMarker はグラフ状態に影響しない（PoCではno-op）
	log := NewEventLog()

	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{
		Type:   EventTransactionMarker,
		TxID:   "tx-001",
		TxMeta: map[string]string{"user": "alice", "reason": "initial setup"},
	})

	g := ReplayLatest(log)

	// Transaction marker should not affect graph state
	if g.NodeCount() != 2 {
		t.Errorf("expected 2 nodes, got %d", g.NodeCount())
	}

	// Transaction marker should have no seeds
	txEvent := Event{Type: EventTransactionMarker, TxID: "tx-002"}
	if len(txEvent.ImpactSeeds()) != 0 {
		t.Error("transaction marker should have no impact seeds")
	}
	if len(txEvent.ValidationSeeds()) != 0 {
		t.Error("transaction marker should have no validation seeds")
	}
}

func TestIncrementalReplay(t *testing.T) {
	// IncrementalReplay で差分適用できることを確認する
	log := NewEventLog()

	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "c", NodeType: NodeField})

	// Replay up to revision 1 (a and b)
	g := Replay(log, 1)
	if g.NodeCount() != 2 {
		t.Errorf("expected 2 nodes at rev 1, got %d", g.NodeCount())
	}
	if g.Revision() != 1 {
		t.Errorf("expected revision 1, got %d", g.Revision())
	}

	// Incrementally replay to include c
	IncrementalReplay(g, log, 2)
	if g.NodeCount() != 3 {
		t.Errorf("expected 3 nodes at rev 2, got %d", g.NodeCount())
	}
	if g.Revision() != 2 {
		t.Errorf("expected revision 2, got %d", g.Revision())
	}
}
