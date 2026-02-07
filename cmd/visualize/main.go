package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"

	p "github.com/user/palimpsest"
)

func main() {
	mode := flag.String("mode", "all", "demo mode: all|why|impact|remove|scale")
	depth := flag.Int("depth", 3, "impact tree depth")
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
	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}

func runWhy(ctx context.Context, g *p.Graph) {
	e := p.Event{Type: p.EventAttrUpdated, NodeID: "field:order.subtotal"}
	res := p.ImpactFromEvent(ctx, g, e)
	path := res.Evidence["form:order_entry"].Path
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
