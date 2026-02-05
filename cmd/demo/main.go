package main

import (
	"context"
	"fmt"

	p "github.com/user/palimpsest"
)

func main() {
	fmt.Println("=== Palimpsest PoC Demo ===")
	fmt.Println()

	// Build a realistic low-code configuration graph
	// Scenario: Order management system
	log := p.NewEventLog()

	// --- Entities ---
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "entity:order", NodeType: p.NodeEntity,
		Attrs: p.Attrs{"name": "受注", "description": "受注管理エンティティ"}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "entity:customer", NodeType: p.NodeEntity,
		Attrs: p.Attrs{"name": "顧客", "description": "顧客マスタ"}})

	// --- Fields ---
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order.subtotal", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": "小計", "type": "currency"}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order.tax_rate", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": "税率", "type": "percent", "default": 0.1}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order.tax", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": "消費税", "type": "currency"}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order.total", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": "合計", "type": "currency"}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:customer.name", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": "顧客名", "type": "text"}})

	// --- Expressions (computed fields) ---
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "expr:calc_tax", NodeType: p.NodeExpression,
		Attrs: p.Attrs{"formula": "subtotal * tax_rate"}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "expr:calc_total", NodeType: p.NodeExpression,
		Attrs: p.Attrs{"formula": "subtotal + tax"}})

	// --- Forms ---
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "form:order_entry", NodeType: p.NodeForm,
		Attrs: p.Attrs{"name": "受注入力フォーム"}})

	// --- Lists ---
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "list:order_list", NodeType: p.NodeList,
		Attrs: p.Attrs{"name": "受注一覧", "columns": []string{"subtotal", "tax", "total"}}})

	// --- Roles ---
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "role:sales", NodeType: p.NodeRole,
		Attrs: p.Attrs{"name": "営業担当"}})

	// --- Edges (provider → consumer) ---
	// Entity owns fields
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order.subtotal", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order.tax_rate", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order.tax", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order.total", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:customer", ToNode: "field:customer.name", Label: p.LabelDerives})

	// Expression dependencies
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.subtotal", ToNode: "expr:calc_tax", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.tax_rate", ToNode: "expr:calc_tax", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "expr:calc_tax", ToNode: "field:order.tax", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.subtotal", ToNode: "expr:calc_total", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.tax", ToNode: "expr:calc_total", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "expr:calc_total", ToNode: "field:order.total", Label: p.LabelDerives})

	// Form uses fields
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.subtotal", ToNode: "form:order_entry", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.total", ToNode: "form:order_entry", Label: p.LabelUses})

	// List uses fields
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.subtotal", ToNode: "list:order_list", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.tax", ToNode: "list:order_list", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.total", ToNode: "list:order_list", Label: p.LabelUses})

	// Role controls form access
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "role:sales", ToNode: "form:order_entry", Label: p.LabelControls})

	// Transaction marker
	log.Append(p.Event{Type: p.EventTransactionMarker, TxID: "tx-initial-setup",
		TxMeta: map[string]string{"user": "admin", "reason": "初期構築"}})

	fmt.Printf("Event log: %d events\n", log.Len())

	// Replay to build graph
	g := p.ReplayLatest(log)
	fmt.Printf("Graph: %d nodes at revision %d\n", g.NodeCount(), g.Revision())
	fmt.Println()

	// Validate
	ctx := context.Background()
	vr := p.Validate(ctx, g)
	fmt.Printf("Validation: valid=%v, errors=%d\n", vr.Valid, len(vr.Errors))
	fmt.Println()

	// --- Scenario 1: Change tax_rate field type ---
	fmt.Println("=== Scenario 1: 税率フィールドの型変更 ===")
	changeEvent := p.Event{
		Type:   p.EventAttrUpdated,
		NodeID: "field:order.tax_rate",
		Attrs:  p.Attrs{"type": "decimal", "precision": 4},
	}
	fmt.Printf("Event: %s on %s\n", changeEvent.Type, changeEvent.NodeID)

	impact := p.ImpactFromEvent(ctx, g, changeEvent)
	fmt.Printf("Impact: %d nodes affected\n", len(impact.Impacted))
	fmt.Println("Affected nodes:")
	for nodeID := range impact.Impacted {
		fmt.Printf("  - %s: %s\n", nodeID, impact.Explain(nodeID))
	}
	fmt.Println()

	// --- Scenario 2: Change subtotal field ---
	fmt.Println("=== Scenario 2: 小計フィールドの変更 ===")
	changeEvent2 := p.Event{
		Type:   p.EventAttrUpdated,
		NodeID: "field:order.subtotal",
		Attrs:  p.Attrs{"validation": "required"},
	}
	fmt.Printf("Event: %s on %s\n", changeEvent2.Type, changeEvent2.NodeID)

	impact2 := p.ImpactFromEvent(ctx, g, changeEvent2)
	fmt.Printf("Impact: %d nodes affected\n", len(impact2.Impacted))
	fmt.Println("Affected nodes:")
	for nodeID := range impact2.Impacted {
		fmt.Printf("  - %s: %s\n", nodeID, impact2.Explain(nodeID))
	}
	fmt.Println()

	// --- Scenario 3: Add new edge (role controls list) ---
	fmt.Println("=== Scenario 3: 権限設定の追加 ===")
	edgeEvent := p.Event{
		Type:     p.EventEdgeAdded,
		FromNode: "role:sales",
		ToNode:   "list:order_list",
		Label:    p.LabelControls,
	}
	fmt.Printf("Event: %s (%s -> %s, label=%s)\n", edgeEvent.Type, edgeEvent.FromNode, edgeEvent.ToNode, edgeEvent.Label)

	// Note: controls label includes both endpoints in ImpactSeeds
	fmt.Printf("Impact Seeds: %v\n", edgeEvent.ImpactSeeds())
	fmt.Printf("Validation Seeds: %v\n", edgeEvent.ValidationSeeds())

	// Apply the event and compute impact
	log.Append(edgeEvent)
	p.IncrementalReplay(g, log, log.Len()-1)

	impact3 := p.ImpactFromEvent(ctx, g, edgeEvent)
	fmt.Printf("Impact: %d nodes affected\n", len(impact3.Impacted))
	fmt.Println()

	// --- Evidence Path Demo ---
	fmt.Println("=== Evidence Path: なぜ form:order_entry が影響を受けるか ===")
	subtotalChange := p.Event{
		Type:   p.EventAttrUpdated,
		NodeID: "field:order.subtotal",
	}
	impactDemo := p.ImpactFromEvent(ctx, g, subtotalChange)
	if impactDemo.Impacted["form:order_entry"] {
		evidence := impactDemo.Evidence["form:order_entry"]
		fmt.Printf("Path: ")
		for i, node := range evidence.Path {
			if i > 0 {
				fmt.Print(" → ")
			}
			fmt.Print(node)
		}
		fmt.Println()
		fmt.Println()
		fmt.Println("説明: 小計フィールドの変更は、計算式、消費税、合計、")
		fmt.Println("      そして受注入力フォームと受注一覧に影響を与えます。")
	}

	fmt.Println()
	fmt.Println("=== Demo Complete ===")
}
