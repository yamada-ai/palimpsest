package palimpsest

import "context"

// ValidationError represents a constraint violation.
// MVPでは主に参照整合性（dangling）を扱う。
type ValidationError struct {
	Type    string
	NodeID  NodeID
	Message string
	// For edge-related errors
	FromNode NodeID
	ToNode   NodeID
	Label    EdgeLabel
}

// ValidationResult contains the result of validation.
// Validationは適用前のゲートとして使う想定。
type ValidationResult struct {
	Valid    bool
	Errors   []ValidationError
	Revision int
	// Whether the computation was cancelled
	Cancelled bool
}

// Validate checks invariants on the graph.
// 現在は参照整合性（dangling edge なし）のみを確認する。
func Validate(ctx context.Context, g *Graph) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Revision: g.Revision(),
	}

	nodeIDs := g.AllNodeIDs()
	for _, id := range nodeIDs {
		// Check cancellation periodically
		select {
		case <-ctx.Done():
			result.Cancelled = true
			return result
		default:
		}

		node := g.GetNode(id)
		if node == nil {
			continue
		}

		// Check outgoing edges for dangling references
		for _, edge := range node.Outgoing {
			if !g.HasNode(edge.To) {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Type:     "dangling_edge",
					NodeID:   id,
					FromNode: edge.From,
					ToNode:   edge.To,
					Label:    edge.Label,
					Message:  "edge references non-existent target node",
				})
			}
		}

		// Check incoming edges for dangling references
		for _, edge := range node.Incoming {
			if !g.HasNode(edge.From) {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Type:     "dangling_edge",
					NodeID:   id,
					FromNode: edge.From,
					ToNode:   edge.To,
					Label:    edge.Label,
					Message:  "edge references non-existent source node",
				})
			}
		}
	}

	return result
}

// ValidateSeeds checks only the nodes in seeds and their edges.
// イベント局所の検証に使い、全走査を避ける。
func ValidateSeeds(ctx context.Context, g *Graph, seeds []NodeID) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Revision: g.Revision(),
	}

	checked := make(map[NodeID]bool)
	for _, id := range seeds {
		if checked[id] {
			continue
		}
		checked[id] = true

		// Check cancellation
		select {
		case <-ctx.Done():
			result.Cancelled = true
			return result
		default:
		}

		node := g.GetNode(id)
		if node == nil {
			// Seed references a removed node - this is okay for NodeRemoved events
			continue
		}

		// Check outgoing edges
		for _, edge := range node.Outgoing {
			if !g.HasNode(edge.To) {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Type:     "dangling_edge",
					NodeID:   id,
					FromNode: edge.From,
					ToNode:   edge.To,
					Label:    edge.Label,
					Message:  "edge references non-existent target node",
				})
			}
		}

		// Check incoming edges
		for _, edge := range node.Incoming {
			if !g.HasNode(edge.From) {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Type:     "dangling_edge",
					NodeID:   id,
					FromNode: edge.From,
					ToNode:   edge.To,
					Label:    edge.Label,
					Message:  "edge references non-existent source node",
				})
			}
		}
	}

	return result
}

// ValidateEvent validates a single event before applying it.
// 1) イベント固有の前提チェック
// 2) 重要イベントのみ局所の不変条件（ValidateSeeds）も併用
func ValidateEvent(ctx context.Context, g *Graph, e Event) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Revision: g.Revision(),
	}

	// Check cancellation early
	select {
	case <-ctx.Done():
		result.Cancelled = true
		return result
	default:
	}

	switch e.Type {
	case EventNodeAdded:
		if g.HasNode(e.NodeID) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    "node_exists",
				NodeID:  e.NodeID,
				Message: "node already exists",
			})
		}
	case EventNodeRemoved:
		if !g.HasNode(e.NodeID) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    "missing_node",
				NodeID:  e.NodeID,
				Message: "node does not exist",
			})
			return result
		}
		node := g.GetNode(e.NodeID)
		if node != nil && (len(node.Incoming) > 0 || len(node.Outgoing) > 0) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    "node_in_use",
				NodeID:  e.NodeID,
				Message: "node has incoming or outgoing edges",
			})
		}
	case EventAttrUpdated:
		if !g.HasNode(e.NodeID) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    "missing_node",
				NodeID:  e.NodeID,
				Message: "node does not exist",
			})
		}
	case EventEdgeAdded:
		if !g.HasNode(e.FromNode) || !g.HasNode(e.ToNode) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:     "missing_endpoint",
				FromNode: e.FromNode,
				ToNode:   e.ToNode,
				Label:    e.Label,
				Message:  "edge endpoints must exist",
			})
		}
	case EventEdgeRemoved:
		if !g.HasNode(e.FromNode) || !g.HasNode(e.ToNode) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:     "missing_endpoint",
				FromNode: e.FromNode,
				ToNode:   e.ToNode,
				Label:    e.Label,
				Message:  "edge endpoints must exist",
			})
			return result
		}
		// Ensure the edge exists before removing
		node := g.GetNode(e.FromNode)
		if node == nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:     "missing_edge",
				FromNode: e.FromNode,
				ToNode:   e.ToNode,
				Label:    e.Label,
				Message:  "edge not found",
			})
			return result
		}
		found := false
		for _, edge := range node.Outgoing {
			if edge.To == e.ToNode && edge.Label == e.Label {
				found = true
				break
			}
		}
		if !found {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:     "missing_edge",
				FromNode: e.FromNode,
				ToNode:   e.ToNode,
				Label:    e.Label,
				Message:  "edge not found",
			})
		}
	case EventTransactionMarker:
		// No-op
	}

	// Merge local invariant checks for selected events
	switch e.Type {
	case EventEdgeAdded, EventEdgeRemoved, EventNodeRemoved:
		seedResult := ValidateSeeds(ctx, g, e.ValidationSeeds())
		if seedResult.Cancelled {
			result.Cancelled = true
			return result
		}
		if !seedResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, seedResult.Errors...)
		}
	}

	return result
}
