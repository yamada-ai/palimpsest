package palimpsest

import "sync"

// Edge represents a labeled directed edge in the graph.
// provider → consumer の向きで保持する。
type Edge struct {
	From  NodeID
	To    NodeID
	Label EdgeLabel
}

// Node represents a configuration element in the graph.
// Outgoing は「このノードに依存するノード群」。
type Node struct {
	ID       NodeID
	Type     NodeType
	Attrs    Attrs
	Outgoing []Edge // edges where this node is the provider (from)
	Incoming []Edge // edges where this node is the consumer (to)
}

// Graph represents the configuration state at a given revision.
// Impact計算中の並行読み取りを想定し、読み取りはRLockで守る。
type Graph struct {
	mu       sync.RWMutex
	nodes    map[NodeID]*Node
	revision int
}

// NewGraph creates an empty graph
func NewGraph() *Graph {
	return &Graph{
		nodes:    make(map[NodeID]*Node),
		revision: -1,
	}
}

// Revision returns the current revision (event log offset)
func (g *Graph) Revision() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.revision
}

// GetNode returns a defensive copy of the node by ID (nil if not found).
// 呼び出し側の無ロック変更を防ぐためコピーを返す。
func (g *Graph) GetNode(id NodeID) *Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	node := g.nodes[id]
	if node == nil {
		return nil
	}
	return cloneNode(node)
}

// HasNode checks if a node exists
func (g *Graph) HasNode(id NodeID) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.nodes[id]
	return ok
}

// NodeCount returns the number of nodes
func (g *Graph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// AllNodeIDs returns all node IDs (for iteration)
func (g *Graph) AllNodeIDs() []NodeID {
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := make([]NodeID, 0, len(g.nodes))
	for id := range g.nodes {
		ids = append(ids, id)
	}
	return ids
}

// Successors returns nodes that depend on the given node (outgoing edges).
// 変更の影響はこの順方向に伝播する。
func (g *Graph) Successors(id NodeID) []NodeID {
	g.mu.RLock()
	defer g.mu.RUnlock()
	node := g.nodes[id]
	if node == nil {
		return nil
	}
	result := make([]NodeID, len(node.Outgoing))
	for i, e := range node.Outgoing {
		result[i] = e.To
	}
	return result
}

// OutgoingEdges returns outgoing edges for a node.
// Returned slice is a copy and safe for read-only use.
func (g *Graph) OutgoingEdges(id NodeID) []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	node := g.nodes[id]
	if node == nil {
		return nil
	}
	out := make([]Edge, len(node.Outgoing))
	copy(out, node.Outgoing)
	return out
}

// IncomingEdges returns incoming edges for a node.
// Returned slice is a copy and safe for read-only use.
func (g *Graph) IncomingEdges(id NodeID) []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	node := g.nodes[id]
	if node == nil {
		return nil
	}
	in := make([]Edge, len(node.Incoming))
	copy(in, node.Incoming)
	return in
}

// NodeTypeOf returns the node type and whether it exists.
func (g *Graph) NodeTypeOf(id NodeID) (NodeType, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	node := g.nodes[id]
	if node == nil {
		return "", false
	}
	return node.Type, true
}

// Predecessors returns nodes that the given node depends on (incoming edges).
// 依存元の参照に使う。
func (g *Graph) Predecessors(id NodeID) []NodeID {
	g.mu.RLock()
	defer g.mu.RUnlock()
	node := g.nodes[id]
	if node == nil {
		return nil
	}
	result := make([]NodeID, len(node.Incoming))
	for i, e := range node.Incoming {
		result[i] = e.From
	}
	return result
}

// --- Mutation methods (used during Replay) ---

func (g *Graph) addNode(id NodeID, nodeType NodeType, attrs Attrs) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if attrs == nil {
		attrs = make(Attrs)
	}
	g.nodes[id] = &Node{
		ID:       id,
		Type:     nodeType,
		Attrs:    attrs,
		Outgoing: make([]Edge, 0),
		Incoming: make([]Edge, 0),
	}
}

func (g *Graph) removeNode(id NodeID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	node := g.nodes[id]
	if node == nil {
		return
	}
	// Remove all edges referencing this node
	for _, e := range node.Outgoing {
		if target := g.nodes[e.To]; target != nil {
			target.Incoming = removeEdgeFrom(target.Incoming, id)
		}
	}
	for _, e := range node.Incoming {
		if source := g.nodes[e.From]; source != nil {
			source.Outgoing = removeEdgeTo(source.Outgoing, id)
		}
	}
	delete(g.nodes, id)
}

func (g *Graph) updateAttrs(id NodeID, attrs Attrs) {
	g.mu.Lock()
	defer g.mu.Unlock()
	node := g.nodes[id]
	if node == nil {
		return
	}
	for k, v := range attrs {
		if v == nil {
			delete(node.Attrs, k)
		} else {
			node.Attrs[k] = v
		}
	}
}

func (g *Graph) addEdge(from, to NodeID, label EdgeLabel) {
	g.mu.Lock()
	defer g.mu.Unlock()
	fromNode := g.nodes[from]
	toNode := g.nodes[to]
	if fromNode == nil || toNode == nil {
		return // silently ignore dangling edges during replay
	}
	edge := Edge{From: from, To: to, Label: label}
	fromNode.Outgoing = append(fromNode.Outgoing, edge)
	toNode.Incoming = append(toNode.Incoming, edge)
}

func (g *Graph) removeEdge(from, to NodeID, label EdgeLabel) {
	g.mu.Lock()
	defer g.mu.Unlock()
	fromNode := g.nodes[from]
	toNode := g.nodes[to]
	if fromNode != nil {
		fromNode.Outgoing = removeEdgeByTarget(fromNode.Outgoing, to, label)
	}
	if toNode != nil {
		toNode.Incoming = removeEdgeBySource(toNode.Incoming, from, label)
	}
}

func (g *Graph) setRevision(rev int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.revision = rev
}

// Clone returns a deep copy of the graph suitable for speculative updates.
func (g *Graph) Clone() *Graph {
	g.mu.RLock()
	defer g.mu.RUnlock()
	nodes := make(map[NodeID]*Node, len(g.nodes))
	for id, node := range g.nodes {
		nodes[id] = cloneNode(node)
	}
	return &Graph{
		nodes:    nodes,
		revision: g.revision,
	}
}

func cloneNode(src *Node) *Node {
	if src == nil {
		return nil
	}
	attrs := make(Attrs, len(src.Attrs))
	for k, v := range src.Attrs {
		attrs[k] = v
	}
	outgoing := make([]Edge, len(src.Outgoing))
	copy(outgoing, src.Outgoing)
	incoming := make([]Edge, len(src.Incoming))
	copy(incoming, src.Incoming)
	return &Node{
		ID:       src.ID,
		Type:     src.Type,
		Attrs:    attrs,
		Outgoing: outgoing,
		Incoming: incoming,
	}
}

// Helper functions for edge removal
func removeEdgeFrom(edges []Edge, from NodeID) []Edge {
	result := edges[:0]
	for _, e := range edges {
		if e.From != from {
			result = append(result, e)
		}
	}
	return result
}

func removeEdgeTo(edges []Edge, to NodeID) []Edge {
	result := edges[:0]
	for _, e := range edges {
		if e.To != to {
			result = append(result, e)
		}
	}
	return result
}

func removeEdgeByTarget(edges []Edge, to NodeID, label EdgeLabel) []Edge {
	result := edges[:0]
	for _, e := range edges {
		if !(e.To == to && e.Label == label) {
			result = append(result, e)
		}
	}
	return result
}

func removeEdgeBySource(edges []Edge, from NodeID, label EdgeLabel) []Edge {
	result := edges[:0]
	for _, e := range edges {
		if !(e.From == from && e.Label == label) {
			result = append(result, e)
		}
	}
	return result
}
