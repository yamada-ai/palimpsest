package palimpsest

import "context"

// EvidencePath represents a path from a seed to an impacted node
// π(s → x) = (s = v_0, v_1, ..., v_k = x)
type EvidencePath struct {
	Seed   NodeID
	Target NodeID
	Path   []NodeID // includes both seed and target
}

// ImpactResult contains the result of impact analysis
type ImpactResult struct {
	// Seeds that initiated the analysis
	Seeds []NodeID

	// Impacted nodes (including seeds)
	Impacted map[NodeID]bool

	// Evidence paths: for each impacted node, the shortest path from a seed
	Evidence map[NodeID]EvidencePath

	// Revision at which analysis was performed
	Revision int

	// Whether the computation was cancelled
	Cancelled bool
}

// ComputeImpact performs BFS from seeds to find all reachable nodes
// Impact(S) = Reach_G(S) = { v ∈ V | ∃s ∈ S, s ⤳ v }
//
// Uses context for cancellation support
func ComputeImpact(ctx context.Context, g *Graph, seeds []NodeID) *ImpactResult {
	result := &ImpactResult{
		Seeds:    seeds,
		Impacted: make(map[NodeID]bool),
		Evidence: make(map[NodeID]EvidencePath),
		Revision: g.Revision(),
	}

	if len(seeds) == 0 {
		return result
	}

	// BFS state
	visited := make(map[NodeID]bool)
	parent := make(map[NodeID]NodeID)  // for reconstructing paths
	seedOf := make(map[NodeID]NodeID)  // which seed reached this node
	queue := make([]NodeID, 0, len(seeds))

	// Initialize with seeds
	for _, seed := range seeds {
		if !g.HasNode(seed) {
			continue
		}
		if visited[seed] {
			continue
		}
		visited[seed] = true
		seedOf[seed] = seed
		queue = append(queue, seed)
		result.Impacted[seed] = true
		result.Evidence[seed] = EvidencePath{
			Seed:   seed,
			Target: seed,
			Path:   []NodeID{seed},
		}
	}

	// BFS traversal following provider → consumer edges
	for len(queue) > 0 {
		// Check for cancellation
		select {
		case <-ctx.Done():
			result.Cancelled = true
			return result
		default:
		}

		current := queue[0]
		queue = queue[1:]

		// Get successors (nodes that depend on current)
		successors := g.Successors(current)
		for _, next := range successors {
			if visited[next] {
				continue
			}
			visited[next] = true
			parent[next] = current
			seedOf[next] = seedOf[current]
			queue = append(queue, next)

			result.Impacted[next] = true
			result.Evidence[next] = buildEvidencePath(seedOf[next], next, parent)
		}
	}

	return result
}

// buildEvidencePath reconstructs the path from seed to target using parent pointers
func buildEvidencePath(seed, target NodeID, parent map[NodeID]NodeID) EvidencePath {
	// Build path in reverse
	path := []NodeID{target}
	current := target
	for current != seed {
		current = parent[current]
		path = append(path, current)
	}

	// Reverse to get seed → target order
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return EvidencePath{
		Seed:   seed,
		Target: target,
		Path:   path,
	}
}

// ImpactFromEvent computes impact for a single event
func ImpactFromEvent(ctx context.Context, g *Graph, e Event) *ImpactResult {
	return ComputeImpact(ctx, g, e.ImpactSeeds())
}

// ImpactFromEvents computes combined impact for multiple events
func ImpactFromEvents(ctx context.Context, g *Graph, events []Event) *ImpactResult {
	seedSet := make(map[NodeID]bool)
	for _, e := range events {
		for _, seed := range e.ImpactSeeds() {
			seedSet[seed] = true
		}
	}

	seeds := make([]NodeID, 0, len(seedSet))
	for seed := range seedSet {
		seeds = append(seeds, seed)
	}

	return ComputeImpact(ctx, g, seeds)
}

// Explain returns a human-readable explanation of why a node is impacted
func (r *ImpactResult) Explain(nodeID NodeID) string {
	evidence, ok := r.Evidence[nodeID]
	if !ok {
		return "not impacted"
	}

	if evidence.Seed == evidence.Target {
		return "directly modified (seed)"
	}

	// Build explanation string
	explanation := "impacted via: "
	for i, node := range evidence.Path {
		if i > 0 {
			explanation += " → "
		}
		explanation += string(node)
	}
	return explanation
}
