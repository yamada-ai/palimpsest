package palimpsest

import (
	"context"
	"fmt"
	"testing"
)

type benchSpec struct {
	name  string
	nodes int
	edges int
}

var benchSpecs = []benchSpec{
	{name: "N10k_M30k", nodes: 10000, edges: 30000},
	{name: "N50k_M150k", nodes: 50000, edges: 150000},
}

func buildBenchLog(nodes, edges int) *EventLog {
	log := NewEventLog()
	for i := 0; i < nodes; i++ {
		id := NodeID(fmt.Sprintf("n:%d", i))
		log.Append(Event{Type: EventNodeAdded, NodeID: id, NodeType: NodeField})
	}
	for i := 0; i < edges; i++ {
		from := i % nodes
		to := (i*7 + 1) % nodes
		if from == to {
			to = (to + 1) % nodes
		}
		log.Append(Event{Type: EventEdgeAdded, FromNode: NodeID(fmt.Sprintf("n:%d", from)), ToNode: NodeID(fmt.Sprintf("n:%d", to)), Label: LabelUses})
	}
	return log
}

func buildAttrLog(nodes, edges, attrKeys int) *EventLog {
	log := NewEventLog()
	for i := 0; i < nodes; i++ {
		id := NodeID(fmt.Sprintf("n:%d", i))
		attrs := make(Attrs, attrKeys)
		for k := 0; k < attrKeys; k++ {
			attrs[fmt.Sprintf("k%d", k)] = VNumber(float64(k))
		}
		log.Append(Event{Type: EventNodeAdded, NodeID: id, NodeType: NodeField, Attrs: attrs})
	}
	for i := 0; i < edges; i++ {
		from := i % nodes
		to := (i*7 + 1) % nodes
		if from == to {
			to = (to + 1) % nodes
		}
		log.Append(Event{Type: EventEdgeAdded, FromNode: NodeID(fmt.Sprintf("n:%d", from)), ToNode: NodeID(fmt.Sprintf("n:%d", to)), Label: LabelUses})
	}
	return log
}

func buildChainLog(nodes int) *EventLog {
	// Deep path: n:0 -> n:1 -> ... -> n:(nodes-1)
	log := NewEventLog()
	for i := 0; i < nodes; i++ {
		id := NodeID(fmt.Sprintf("n:%d", i))
		log.Append(Event{Type: EventNodeAdded, NodeID: id, NodeType: NodeField})
	}
	for i := 0; i < nodes-1; i++ {
		log.Append(Event{
			Type:     EventEdgeAdded,
			FromNode: NodeID(fmt.Sprintf("n:%d", i)),
			ToNode:   NodeID(fmt.Sprintf("n:%d", i+1)),
			Label:    LabelUses,
		})
	}
	return log
}

func buildHubLog(nodes int) *EventLog {
	// Worst-case-ish: one hub node depends to all others.
	log := NewEventLog()
	for i := 0; i < nodes; i++ {
		id := NodeID(fmt.Sprintf("n:%d", i))
		log.Append(Event{Type: EventNodeAdded, NodeID: id, NodeType: NodeField})
	}
	// Hub: n:0 -> n:1..n:(nodes-1)
	for i := 1; i < nodes; i++ {
		log.Append(Event{
			Type:     EventEdgeAdded,
			FromNode: "n:0",
			ToNode:   NodeID(fmt.Sprintf("n:%d", i)),
			Label:    LabelUses,
		})
	}
	return log
}

func BenchmarkReplay(b *testing.B) {
	for _, spec := range benchSpecs {
		spec := spec
		b.Run(spec.name, func(b *testing.B) {
			log := buildBenchLog(spec.nodes, spec.edges)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ReplayLatest(log)
			}
		})
	}
}

func BenchmarkReplayWithAttrs(b *testing.B) {
	attrKeys := []int{1, 5, 20}
	for _, spec := range benchSpecs {
		spec := spec
		b.Run(spec.name, func(b *testing.B) {
			for _, keys := range attrKeys {
				keys := keys
				b.Run(fmt.Sprintf("Attrs%d", keys), func(b *testing.B) {
					log := buildAttrLog(spec.nodes, spec.edges, keys)
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_ = ReplayLatest(log)
					}
				})
			}
		})
	}
}

func BenchmarkImpact(b *testing.B) {
	ctx := context.Background()
	for _, spec := range benchSpecs {
		spec := spec
		b.Run(spec.name, func(b *testing.B) {
			log := buildBenchLog(spec.nodes, spec.edges)
			g := ReplayLatest(log)
			nodeIDs := make([]NodeID, 0, spec.nodes)
			for i := 0; i < spec.nodes; i++ {
				nodeIDs = append(nodeIDs, NodeID(fmt.Sprintf("n:%d", i)))
			}
			event := Event{Type: EventAttrUpdated, Attrs: Attrs{"touched": VBool(true)}}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				event.NodeID = nodeIDs[i%len(nodeIDs)]
				_ = ImpactFromEvent(ctx, g, event)
			}
		})
	}
}

func BenchmarkImpactWorstCase(b *testing.B) {
	ctx := context.Background()
	for _, spec := range benchSpecs {
		spec := spec
		b.Run(spec.name, func(b *testing.B) {
			b.Run("Hub", func(b *testing.B) {
				log := buildHubLog(spec.nodes)
				g := ReplayLatest(log)
				event := Event{Type: EventAttrUpdated, NodeID: "n:0", Attrs: Attrs{"touched": VBool(true)}}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = ImpactFromEvent(ctx, g, event)
				}
			})
			b.Run("Chain", func(b *testing.B) {
				log := buildChainLog(spec.nodes)
				g := ReplayLatest(log)
				event := Event{Type: EventAttrUpdated, NodeID: "n:0", Attrs: Attrs{"touched": VBool(true)}}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = ImpactFromEvent(ctx, g, event)
				}
			})
		})
	}
}

func BenchmarkSimulateEvent(b *testing.B) {
	ctx := context.Background()
	for _, spec := range benchSpecs {
		spec := spec
		b.Run(spec.name, func(b *testing.B) {
			log := buildBenchLog(spec.nodes, spec.edges)
			g := ReplayLatest(log)
			event := Event{Type: EventAttrUpdated, NodeID: NodeID("n:42"), Attrs: Attrs{"touched": VBool(true)}}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = SimulateEvent(ctx, g, event)
			}
		})
	}
}

func BenchmarkSimulateTx(b *testing.B) {
	ctx := context.Background()
	txSizes := []int{1, 10, 100}
	for _, spec := range benchSpecs {
		spec := spec
		b.Run(spec.name, func(b *testing.B) {
			nodeIDs := make([]NodeID, 0, spec.nodes)
			for i := 0; i < spec.nodes; i++ {
				nodeIDs = append(nodeIDs, NodeID(fmt.Sprintf("n:%d", i)))
			}
			for _, size := range txSizes {
				size := size
				b.Run(fmt.Sprintf("Tx%d", size), func(b *testing.B) {
					log := buildBenchLog(spec.nodes, spec.edges)
					g := ReplayLatest(log)
					events := make([]Event, 0, size)
					for i := 0; i < size; i++ {
						events = append(events, Event{
							Type:   EventAttrUpdated,
							NodeID: nodeIDs[i%len(nodeIDs)],
							Attrs:  Attrs{"touched": VBool(true)},
						})
					}
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						start := i % len(nodeIDs)
						for j := range events {
							events[j].NodeID = nodeIDs[(start+j)%len(nodeIDs)]
						}
						_ = SimulateTx(ctx, g, events)
					}
				})
			}
		})
	}
}

func BenchmarkValidateEvent(b *testing.B) {
	ctx := context.Background()
	for _, spec := range benchSpecs {
		spec := spec
		b.Run(spec.name, func(b *testing.B) {
			log := buildBenchLog(spec.nodes, spec.edges)
			g := ReplayLatest(log)
			event := Event{Type: EventAttrUpdated, NodeID: NodeID("n:42"), Attrs: Attrs{"touched": VBool(true)}}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ValidateEvent(ctx, g, event)
			}
		})
	}
}

func BenchmarkValidateFull(b *testing.B) {
	ctx := context.Background()
	for _, spec := range benchSpecs {
		spec := spec
		b.Run(spec.name, func(b *testing.B) {
			log := buildBenchLog(spec.nodes, spec.edges)
			g := ReplayLatest(log)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = Validate(ctx, g)
			}
		})
	}
}

func BenchmarkBuildGraphFromSnapshot(b *testing.B) {
	spec := benchSpec{name: "N50k_M150k", nodes: 50000, edges: 150000}
	intervals := []int{1000, 5000, 10000}
	for _, interval := range intervals {
		interval := interval
		b.Run(fmt.Sprintf("SnapEvery%d", interval), func(b *testing.B) {
			log := buildBenchLog(spec.nodes, spec.edges)
			latest := log.Len() - 1
			if latest < 0 {
				return
			}
			// Pick the latest snapshot boundary (k*interval - 1).
			k := (latest + 1) / interval
			rev := k*interval - 1
			if rev < 0 {
				rev = latest
			}
			snap := SnapshotFromLog(log, rev)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ReplayFromSnapshot(snap, log, latest)
			}
		})
	}
}
