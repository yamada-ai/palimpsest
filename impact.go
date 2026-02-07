package palimpsest

import "context"

// EvidencePath represents a path from a seed to an impacted node.
// π(s → x) = (s = v_0, v_1, ..., v_k = x)
type EvidencePath struct {
	Seed   NodeID
	Target NodeID
	Path   []NodeID // includes both seed and target
}

// ImpactResult contains the result of impact analysis.
// Impactは情報提供であり、ブロック判定はValidationの責務。
type ImpactResult struct {
	// Seeds that initiated the analysis
	Seeds []NodeID

	// Impacted nodes (including seeds)
	Impacted map[NodeID]bool

	// Revision at which analysis was performed
	Revision int

	// Whether the computation was cancelled
	Cancelled bool

	parent map[NodeID]NodeID
	seedOf map[NodeID]NodeID
}

// ImpactFilter controls which edges are traversed and which nodes are included.
// EdgeLabels filters traversal; NodeTypes filters inclusion in Impacted.
type ImpactFilter struct {
	EdgeLabels map[EdgeLabel]bool
	NodeTypes  map[NodeType]bool
}

// ComputeImpact performs BFS from seeds to find all reachable nodes.
// Impact(S) = Reach_G(S) = { v ∈ V | ∃s ∈ S, s ⤳ v }
// ctx でキャンセルできる。
func ComputeImpact(ctx context.Context, g *Graph, seeds []NodeID) *ImpactResult {
	return ComputeImpactFiltered(ctx, g, seeds, nil)
}

// ComputeImpactFiltered performs BFS from seeds with optional filters.
// EdgeLabels filters traversal; NodeTypes filters which nodes are included in Impacted.
func ComputeImpactFiltered(ctx context.Context, g *Graph, seeds []NodeID, filter *ImpactFilter) *ImpactResult {
	result := &ImpactResult{
		Seeds:    seeds,
		Impacted: make(map[NodeID]bool),
		Revision: g.Revision(),
		parent:   make(map[NodeID]NodeID),
		seedOf:   make(map[NodeID]NodeID),
	}

	if len(seeds) == 0 {
		return result
	}

	// BFS state（最短パスの親を保持）
	visited := make(map[NodeID]bool)
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
		queue = append(queue, seed)
		if includeNodeType(g, seed, filter) {
			result.Impacted[seed] = true
		}
		result.seedOf[seed] = seed
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
		for _, edge := range g.OutgoingEdges(current) {
			if !allowEdgeLabel(edge.Label, filter) {
				continue
			}
			next := edge.To
			if visited[next] {
				continue
			}
			visited[next] = true
			result.parent[next] = current
			result.seedOf[next] = result.seedOf[current]
			queue = append(queue, next)

			if includeNodeType(g, next, filter) {
				result.Impacted[next] = true
			}
		}
	}

	return result
}

// ImpactFromEvent computes impact for a single event.
// 変更イベントから seeds を引き、影響範囲を計算する。
func ImpactFromEvent(ctx context.Context, g *Graph, e Event) *ImpactResult {
	return ComputeImpact(ctx, g, e.ImpactSeeds())
}

// ImpactFromEvents computes combined impact for multiple events.
// 複数イベントの seeds を集合化して一度だけBFSする。
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

// ImpactFromEventFiltered computes impact for a single event with filters.
func ImpactFromEventFiltered(ctx context.Context, g *Graph, e Event, filter *ImpactFilter) *ImpactResult {
	return ComputeImpactFiltered(ctx, g, e.ImpactSeeds(), filter)
}

// ImpactFromEventsFiltered computes combined impact for multiple events with filters.
func ImpactFromEventsFiltered(ctx context.Context, g *Graph, events []Event, filter *ImpactFilter) *ImpactResult {
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

	return ComputeImpactFiltered(ctx, g, seeds, filter)
}

// EvidencePath returns the shortest evidence path for a node on demand.
// NOTE: NodeType filters only affect inclusion; path may include filtered-out nodes.
func (r *ImpactResult) EvidencePath(nodeID NodeID) (EvidencePath, bool) {
	if !r.Impacted[nodeID] {
		return EvidencePath{}, false
	}
	seed, ok := r.seedOf[nodeID]
	if !ok {
		return EvidencePath{}, false
	}
	if seed == nodeID {
		return EvidencePath{Seed: seed, Target: nodeID, Path: []NodeID{nodeID}}, true
	}
	path := []NodeID{nodeID}
	current := nodeID
	for current != seed {
		parent, ok := r.parent[current]
		if !ok {
			return EvidencePath{}, false
		}
		path = append(path, parent)
		current = parent
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return EvidencePath{Seed: seed, Target: nodeID, Path: path}, true
}

// Path returns only the node sequence for the evidence path.
func (r *ImpactResult) Path(nodeID NodeID) []NodeID {
	evidence, ok := r.EvidencePath(nodeID)
	if !ok {
		return nil
	}
	return evidence.Path
}

// Explain returns a human-readable explanation of why a node is impacted.
// 影響理由（証拠パス）を簡潔に返す。
func (r *ImpactResult) Explain(nodeID NodeID) string {
	evidence, ok := r.EvidencePath(nodeID)
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

func allowEdgeLabel(label EdgeLabel, filter *ImpactFilter) bool {
	if filter == nil || len(filter.EdgeLabels) == 0 {
		return true
	}
	return filter.EdgeLabels[label]
}

func includeNodeType(g *Graph, id NodeID, filter *ImpactFilter) bool {
	if filter == nil || len(filter.NodeTypes) == 0 {
		return true
	}
	nodeType, ok := g.NodeTypeOf(id)
	if !ok {
		return false
	}
	return filter.NodeTypes[nodeType]
}
