package palimpsest

import "fmt"

// Delta represents the change set produced by applying a single event.
// ロールバックに必要十分な情報を保持する差分。
type Delta struct {
	Event     Event
	BeforeRev int

	// Node changes
	AddedNodes   []NodeID
	RemovedNodes []NodeSnapshot
	UpdatedAttrs []AttrChange

	// Edge changes
	AddedEdges   []Edge
	RemovedEdges []Edge
}

// NodeSnapshot captures a complete node snapshot for rollback.
// NodeRemoved の復元に必要な情報（attrs/edges含む）を保持する。
type NodeSnapshot struct {
	Node Node
}

// AttrChange captures a single attribute update.
// Deleted はキー削除（After == nil）のとき true。
type AttrChange struct {
	NodeID  NodeID
	Key     string
	Before  any
	After   any
	Deleted bool
}

// ApplyEvent applies a single event to the graph and returns a rollback delta.
// 変更はΔとして記録され、Rollbackで元に戻せることが前提。
func ApplyEvent(g *Graph, e Event) (Delta, error) {
	delta := Delta{
		Event:     e,
		BeforeRev: g.Revision(),
	}

	switch e.Type {
	case EventNodeAdded:
		// 既存ノードへの追加は不正
		if g.HasNode(e.NodeID) {
			return delta, fmt.Errorf("node already exists: %s", e.NodeID)
		}
		g.addNode(e.NodeID, e.NodeType, e.Attrs)
		delta.AddedNodes = append(delta.AddedNodes, e.NodeID)
	case EventNodeRemoved:
		// 削除前のスナップショットを保持してロールバック可能にする
		node := g.GetNode(e.NodeID)
		if node == nil {
			return delta, fmt.Errorf("node does not exist: %s", e.NodeID)
		}
		delta.RemovedNodes = append(delta.RemovedNodes, NodeSnapshot{Node: *node})
		delta.RemovedEdges = append(delta.RemovedEdges, collectIncidentEdges(node)...)
		g.removeNode(e.NodeID)
	case EventAttrUpdated:
		// 変更前後をDeltaに記録して復元できるようにする
		node := g.GetNode(e.NodeID)
		if node == nil {
			return delta, fmt.Errorf("node does not exist: %s", e.NodeID)
		}
		for k, v := range e.Attrs {
			before, _ := node.Attrs[k]
			change := AttrChange{
				NodeID:  e.NodeID,
				Key:     k,
				Before:  before,
				After:   v,
				Deleted: v == nil,
			}
			delta.UpdatedAttrs = append(delta.UpdatedAttrs, change)
		}
		g.updateAttrs(e.NodeID, e.Attrs)
	case EventEdgeAdded:
		// 重複エッジは拒否（Rollbackで既存エッジを消さないため）
		if !g.HasNode(e.FromNode) || !g.HasNode(e.ToNode) {
			return delta, fmt.Errorf("edge endpoints must exist: %s -> %s", e.FromNode, e.ToNode)
		}
		if hasEdge(g, e.FromNode, e.ToNode, e.Label) {
			return delta, fmt.Errorf("edge already exists: %s -> %s (%s)", e.FromNode, e.ToNode, e.Label)
		}
		edge := Edge{From: e.FromNode, To: e.ToNode, Label: e.Label}
		g.addEdge(edge.From, edge.To, edge.Label)
		delta.AddedEdges = append(delta.AddedEdges, edge)
	case EventEdgeRemoved:
		// 削除対象のエッジをDeltaに保存して復元できるようにする
		if !g.HasNode(e.FromNode) || !g.HasNode(e.ToNode) {
			return delta, fmt.Errorf("edge endpoints must exist: %s -> %s", e.FromNode, e.ToNode)
		}
		node := g.GetNode(e.FromNode)
		if node == nil {
			return delta, fmt.Errorf("node does not exist: %s", e.FromNode)
		}
		removed := matchingOutgoingEdges(node.Outgoing, e.ToNode, e.Label)
		if len(removed) == 0 {
			return delta, fmt.Errorf("edge not found: %s -> %s (%s)", e.FromNode, e.ToNode, e.Label)
		}
		g.removeEdge(e.FromNode, e.ToNode, e.Label)
		delta.RemovedEdges = append(delta.RemovedEdges, removed...)
	case EventTransactionMarker:
		// No-op
	}

	return delta, nil
}

// RollbackDelta restores the graph state using a delta.
// 失敗した場合はGraphが不正状態になる可能性があるため、呼び出し側で破棄する。
func RollbackDelta(g *Graph, d Delta) error {
	for _, snap := range d.RemovedNodes {
		if g.HasNode(snap.Node.ID) {
			return fmt.Errorf("node already exists during rollback: %s", snap.Node.ID)
		}
		attrs := cloneAttrs(snap.Node.Attrs)
		g.addNode(snap.Node.ID, snap.Node.Type, attrs)
	}

	for _, change := range d.UpdatedAttrs {
		attrs := Attrs{change.Key: change.Before}
		g.updateAttrs(change.NodeID, attrs)
	}

	for _, edge := range d.AddedEdges {
		g.removeEdge(edge.From, edge.To, edge.Label)
	}

	for _, edge := range d.RemovedEdges {
		if !g.HasNode(edge.From) || !g.HasNode(edge.To) {
			return fmt.Errorf("edge endpoints missing during rollback: %s -> %s", edge.From, edge.To)
		}
		g.addEdge(edge.From, edge.To, edge.Label)
	}

	for _, id := range d.AddedNodes {
		if !g.HasNode(id) {
			return fmt.Errorf("node missing during rollback: %s", id)
		}
		g.removeNode(id)
	}

	return nil
}

func matchingOutgoingEdges(edges []Edge, to NodeID, label EdgeLabel) []Edge {
	matches := make([]Edge, 0)
	for _, edge := range edges {
		if edge.To == to && edge.Label == label {
			matches = append(matches, edge)
		}
	}
	return matches
}

func hasEdge(g *Graph, from, to NodeID, label EdgeLabel) bool {
	node := g.GetNode(from)
	if node == nil {
		return false
	}
	for _, edge := range node.Outgoing {
		if edge.To == to && edge.Label == label {
			return true
		}
	}
	return false
}

func collectIncidentEdges(node *Node) []Edge {
	if node == nil {
		return nil
	}
	seen := make(map[string]bool)
	result := make([]Edge, 0, len(node.Outgoing)+len(node.Incoming))
	add := func(edge Edge) {
		key := fmt.Sprintf("%s|%s|%s", edge.From, edge.To, edge.Label)
		if seen[key] {
			return
		}
		seen[key] = true
		result = append(result, edge)
	}
	for _, edge := range node.Outgoing {
		add(edge)
	}
	for _, edge := range node.Incoming {
		add(edge)
	}
	return result
}

func cloneAttrs(src Attrs) Attrs {
	if src == nil {
		return nil
	}
	out := make(Attrs, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
