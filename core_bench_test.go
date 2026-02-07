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
			event := Event{Type: EventAttrUpdated, Attrs: Attrs{"touched": true}}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				event.NodeID = nodeIDs[i%len(nodeIDs)]
				_ = ImpactFromEvent(ctx, g, event)
			}
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
			event := Event{Type: EventAttrUpdated, NodeID: NodeID("n:42"), Attrs: Attrs{"touched": true}}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = SimulateEvent(ctx, g, event)
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
			event := Event{Type: EventAttrUpdated, NodeID: NodeID("n:42"), Attrs: Attrs{"touched": true}}
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
