package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"time"

	p "github.com/user/palimpsest"
)

func main() {
	mode := flag.String("mode", "all", "demo mode: all|why|impact|remove|scale|repair|repair-cascade")
	depth := flag.Int("depth", 3, "impact tree depth")
	benchNodes := flag.Int("bench-nodes", 20000, "benchmark: number of nodes")
	benchEdges := flag.Int("bench-edges", 60000, "benchmark: number of edges")
	benchSeed := flag.Int("bench-seed", 42, "benchmark: seed index")
	flag.Parse()

	log := buildSampleLog()
	g := p.ReplayLatest(log)
	ctx := context.Background()

	switch *mode {
	case "why":
		runWhy(ctx, g)
	case "impact":
		runImpactTree(ctx, g, *depth)
	case "remove":
		runEdgeRemoval(ctx, g)
	case "scale":
		runScale(ctx)
	case "repair":
		runRepair(ctx, g)
	case "repair-cascade":
		runRepairCascade(ctx, g)
	case "bench":
		runBench(ctx, *benchNodes, *benchEdges, *benchSeed)
	case "all":
		fmt.Println("=== Palimpsest Visual Demo ===")
		fmt.Println("(1) Why: evidence path")
		runWhy(ctx, g)
		fmt.Println()
		fmt.Println("(2) Impact: reachable subgraph")
		runImpactTree(ctx, g, *depth)
		fmt.Println()
		fmt.Println("(3) EdgeRemoved: before evidence")
		runEdgeRemoval(ctx, g)
		fmt.Println()
		fmt.Println("(4) Scale: impact stays local")
		runScale(ctx)
		fmt.Println()
		fmt.Println("(5) Repair: suggestions")
		runRepair(ctx, g)
		fmt.Println()
		fmt.Println("(6) Repair-Cascade: applyable proposals")
		runRepairCascade(ctx, g)
		fmt.Println()
		fmt.Println("(7) Bench: impact time/memory")
		runBench(ctx, *benchNodes, *benchEdges, *benchSeed)
	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}

func runWhy(ctx context.Context, g *p.Graph) {
	e := p.Event{Type: p.EventAttrUpdated, NodeID: "field:order.subtotal"}
	res := p.ImpactFromEvent(ctx, g, e)
	path := res.Path("form:order_entry")
	fmt.Println("Event: AttrUpdated field:order.subtotal")
	fmt.Println("Why is form:order_entry impacted?")
	fmt.Printf("  %s\n", joinPath(path))
}

func runImpactTree(ctx context.Context, g *p.Graph, depth int) {
	e := p.Event{Type: p.EventAttrUpdated, NodeID: "field:order.tax_rate"}
	res := p.ImpactFromEvent(ctx, g, e)
	fmt.Printf("Event: AttrUpdated field:order.tax_rate (impact nodes=%d)\n", len(res.Impacted))
	printImpactTree(g, []p.NodeID{"field:order.tax_rate"}, depth)
}

func runEdgeRemoval(ctx context.Context, g *p.Graph) {
	e := p.Event{Type: p.EventEdgeRemoved, FromNode: "expr:calc_tax", ToNode: "field:order.tax", Label: p.LabelDerives}
	res := p.SimulateEvent(ctx, g, e)
	fmt.Println("Event: EdgeRemoved expr:calc_tax -> field:order.tax (derives)")
	fmt.Printf("PreValidate: valid=%v\n", res.PreValidate.Valid)
	fmt.Println("Evidence (before):")
	fmt.Printf("  %s\n", res.PreImpact.Explain("field:order.tax"))
}

func runScale(ctx context.Context) {
	// Build a larger graph to show impact locality.
	log := p.NewEventLog()
	root := p.NodeID("field:root")
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: root, NodeType: p.NodeField})

	// Build a fan-out chain with side branches.
	prev := root
	for i := 0; i < 500; i++ {
		id := p.NodeID(fmt.Sprintf("field:n%d", i))
		log.Append(p.Event{Type: p.EventNodeAdded, NodeID: id, NodeType: p.NodeField})
		log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: prev, ToNode: id, Label: p.LabelUses})
		// Side branch
		branch := p.NodeID(fmt.Sprintf("expr:b%d", i))
		log.Append(p.Event{Type: p.EventNodeAdded, NodeID: branch, NodeType: p.NodeExpression})
		log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: id, ToNode: branch, Label: p.LabelUses})
		prev = id
	}

	g := p.ReplayLatest(log)
	e := p.Event{Type: p.EventAttrUpdated, NodeID: "field:n42"}
	res := p.ImpactFromEvent(ctx, g, e)
	fmt.Printf("Large graph: nodes=%d, impacted=%d\n", g.NodeCount(), len(res.Impacted))
	fmt.Printf("Example evidence: %s\n", res.Explain("expr:b42"))
}

func runRepair(ctx context.Context, g *p.Graph) {
	e := p.Event{Type: p.EventAttrUpdated, NodeID: "field:order.subtotal", Attrs: p.Attrs{"type": "decimal"}}
	plan := p.ComputeRepairPlan(ctx, g, e)
	planTx := p.ComputeRepairPlanTx(ctx, g, e)
	fmt.Printf("Event: AttrUpdated field:order.subtotal\n")
	fmt.Printf("Summary: %s\n", plan.Summary)
	for _, s := range plan.Suggestions {
		fmt.Printf("  - [%s] %s: %s\n", s.Severity.String(), s.NodeID, s.Message)
		if s.Evidence != "" {
			fmt.Printf("      %s\n", s.Evidence)
		}
	}
	fmt.Printf("Proposed fixes: %d actions\n", len(planTx.Actions))
	if len(planTx.Actions) > 0 && len(planTx.Actions[0].Proposals) > 0 {
		p0 := planTx.Actions[0].Proposals[0]
		fmt.Printf("  e.g. %s (applyable=%v)\n", p0.Event.Type, p0.Applyable)
	}
}

func runRepairCascade(ctx context.Context, g *p.Graph) {
	e := p.Event{Type: p.EventNodeRemoved, NodeID: "field:order.tax"}
	plan := p.ComputeRepairPlanTx(ctx, g, e)
	fmt.Printf("Event: NodeRemoved field:order.tax\n")
	fmt.Printf("Summary: %s\n", plan.Summary)
	for _, a := range plan.Actions {
		fmt.Printf("  - [%s] %s: %s\n", a.Severity.String(), a.NodeID, a.Title)
		for _, p := range a.Proposals {
			fmt.Printf("      %s (applyable=%v) %s\n", p.Event.Type, p.Applyable, p.Note)
		}
	}
}

func runBench(ctx context.Context, nodes, edges, seed int) {
	fmt.Printf("Bench config: nodes=%d edges=%d seed=%d\n", nodes, edges, seed)
	log := p.NewEventLog()

	// Add nodes
	for i := 0; i < nodes; i++ {
		id := p.NodeID(fmt.Sprintf("n:%d", i))
		log.Append(p.Event{Type: p.EventNodeAdded, NodeID: id, NodeType: p.NodeField})
	}

	// Add edges (simple deterministic pattern)
	// Connect i -> (i+1), and i -> (i+2) with wrap-around.
	for i := 0; i < edges; i++ {
		from := i % nodes
		to := (i + 1 + (i%2)) % nodes
		log.Append(p.Event{
			Type:     p.EventEdgeAdded,
			FromNode: p.NodeID(fmt.Sprintf("n:%d", from)),
			ToNode:   p.NodeID(fmt.Sprintf("n:%d", to)),
			Label:    p.LabelUses,
		})
	}

	start := time.Now()
	g := p.ReplayLatest(log)
	replayDur := time.Since(start)

	seedID := p.NodeID(fmt.Sprintf("n:%d", seed%nodes))
	event := p.Event{Type: p.EventAttrUpdated, NodeID: seedID, Attrs: p.Attrs{"touched": true}}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	memBefore := ms.Alloc

	start = time.Now()
	res := p.ImpactFromEvent(ctx, g, event)
	impactDur := time.Since(start)

	runtime.ReadMemStats(&ms)
	memAfter := ms.Alloc

	fmt.Printf("Replay: %s\n", replayDur)
	fmt.Printf("Impact: %s (impacted=%d)\n", impactDur, len(res.Impacted))
	fmt.Printf("Memory delta: %.2f MB\n", float64(memAfter-memBefore)/1024.0/1024.0)
}

func printImpactTree(g *p.Graph, seeds []p.NodeID, maxDepth int) {
	visited := make(map[p.NodeID]bool)
	type entry struct {
		id    p.NodeID
		depth int
	}
	queue := make([]entry, 0, len(seeds))
	for _, s := range seeds {
		queue = append(queue, entry{id: s, depth: 0})
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur.id] {
			continue
		}
		visited[cur.id] = true

		indent := ""
		for i := 0; i < cur.depth; i++ {
			indent += "  "
		}
		fmt.Printf("%s- %s\n", indent, cur.id)

		if cur.depth >= maxDepth {
			continue
		}
		succ := g.Successors(cur.id)
		sort.Slice(succ, func(i, j int) bool { return succ[i] < succ[j] })
		for _, next := range succ {
			queue = append(queue, entry{id: next, depth: cur.depth + 1})
		}
	}
}

func joinPath(path []p.NodeID) string {
	if len(path) == 0 {
		return "(no path)"
	}
	out := string(path[0])
	for i := 1; i < len(path); i++ {
		out += " -> " + string(path[i])
	}
	return out
}

func buildSampleLog() *p.EventLog {
	log := p.NewEventLog()

	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "entity:order", NodeType: p.NodeEntity})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "entity:customer", NodeType: p.NodeEntity})

	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order.subtotal", NodeType: p.NodeField})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order.tax_rate", NodeType: p.NodeField})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order.tax", NodeType: p.NodeField})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order.total", NodeType: p.NodeField})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:customer.name", NodeType: p.NodeField})

	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "expr:calc_tax", NodeType: p.NodeExpression})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "expr:calc_total", NodeType: p.NodeExpression})

	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "form:order_entry", NodeType: p.NodeForm})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "list:order_list", NodeType: p.NodeList})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "role:sales", NodeType: p.NodeRole})

	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order.subtotal", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order.tax_rate", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order.tax", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order.total", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:customer", ToNode: "field:customer.name", Label: p.LabelDerives})

	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.subtotal", ToNode: "expr:calc_tax", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.tax_rate", ToNode: "expr:calc_tax", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "expr:calc_tax", ToNode: "field:order.tax", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.subtotal", ToNode: "expr:calc_total", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.tax", ToNode: "expr:calc_total", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "expr:calc_total", ToNode: "field:order.total", Label: p.LabelDerives})

	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.subtotal", ToNode: "form:order_entry", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.total", ToNode: "form:order_entry", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.subtotal", ToNode: "list:order_list", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.tax", ToNode: "list:order_list", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.total", ToNode: "list:order_list", Label: p.LabelUses})

	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "role:sales", ToNode: "form:order_entry", Label: p.LabelControls})
	log.Append(p.Event{Type: p.EventTransactionMarker, TxID: "tx-initial"})

	return log
}
