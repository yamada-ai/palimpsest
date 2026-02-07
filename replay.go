package palimpsest

// Replay builds a graph by applying events from the log.
// G_r = Replay([e_0, ..., e_r]) を構成する射影。
func Replay(log *EventLog, upToRevision int) *Graph {
	g := NewGraph()
	if upToRevision < 0 {
		return g
	}
	if upToRevision >= log.Len() {
		upToRevision = log.Len() - 1
	}

	events := log.Range(0, upToRevision+1)
	for _, e := range events {
		applyEvent(g, e)
	}
	g.setRevision(upToRevision)
	return g
}

// ReplayLatest builds a graph from all events in the log.
// 最新リビジョンのグラフを得る。
func ReplayLatest(log *EventLog) *Graph {
	return Replay(log, log.Len()-1)
}

// IncrementalReplay applies events from fromRevision+1 to toRevision.
// グラフが fromRevision にある前提で差分適用する。
func IncrementalReplay(g *Graph, log *EventLog, toRevision int) {
	fromRevision := g.Revision()
	if toRevision <= fromRevision {
		return
	}
	if toRevision >= log.Len() {
		toRevision = log.Len() - 1
	}

	events := log.Range(fromRevision+1, toRevision+1)
	for _, e := range events {
		applyEvent(g, e)
	}
	g.setRevision(toRevision)
}

// applyEvent applies a single event to the graph.
// TxMarker は状態に影響しない（PoCではno-op）。
func applyEvent(g *Graph, e Event) {
	switch e.Type {
	case EventNodeAdded:
		g.addNode(e.NodeID, e.NodeType, e.Attrs)
	case EventNodeRemoved:
		g.removeNode(e.NodeID)
	case EventEdgeAdded:
		g.addEdge(e.FromNode, e.ToNode, e.Label)
	case EventEdgeRemoved:
		g.removeEdge(e.FromNode, e.ToNode, e.Label)
	case EventAttrUpdated:
		g.updateAttrs(e.NodeID, e.Attrs)
	case EventTransactionMarker:
		// No-op for graph state; used for audit boundaries
	}
}
