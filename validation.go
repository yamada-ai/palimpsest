package palimpsest

import "context"

// ValidationError represents a constraint violation
type ValidationError struct {
	Type    string
	NodeID  NodeID
	Message string
	// For edge-related errors
	FromNode NodeID
	ToNode   NodeID
	Label    EdgeLabel
}

// ValidationResult contains the result of validation
type ValidationResult struct {
	Valid    bool
	Errors   []ValidationError
	Revision int
	// Whether the computation was cancelled
	Cancelled bool
}

// Validate checks invariants on the graph
// Currently implements: referential integrity (no dangling edges)
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

// ValidateSeeds checks only the nodes in seeds and their edges
// More efficient for incremental validation after events
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

// ValidateEvent validates seeds from a single event
func ValidateEvent(ctx context.Context, g *Graph, e Event) *ValidationResult {
	return ValidateSeeds(ctx, g, e.ValidationSeeds())
}
