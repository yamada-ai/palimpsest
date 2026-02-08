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
		Attrs: p.Attrs{"name": p.VString("受注"), "description": p.VString("受注管理エンティティ")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "entity:customer", NodeType: p.NodeEntity,
		Attrs: p.Attrs{"name": p.VString("顧客"), "description": p.VString("顧客マスタ")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "entity:product", NodeType: p.NodeEntity,
		Attrs: p.Attrs{"name": p.VString("商品"), "description": p.VString("商品マスタ")}})

	// --- Fields ---
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order.subtotal", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": p.VString("小計"), "type": p.VString("currency")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order.tax_rate", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": p.VString("税率"), "type": p.VString("percent"), "default": p.VNumber(0.1)}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order.tax", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": p.VString("消費税"), "type": p.VString("currency")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order.total", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": p.VString("合計"), "type": p.VString("currency")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:customer.name", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": p.VString("顧客名"), "type": p.VString("text")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:product.id", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": p.VString("商品ID"), "type": p.VString("text")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:product.name", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": p.VString("商品名"), "type": p.VString("text")}})

	// --- Relation (N:M) ---
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "rel:order_product", NodeType: p.NodeRelation,
		Attrs: p.Attrs{"name": p.VString("受注明細")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order_product.order_id", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": p.VString("受注ID"), "type": p.VString("text")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order_product.product_id", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": p.VString("商品ID"), "type": p.VString("text")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order_product.quantity", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": p.VString("数量"), "type": p.VString("number")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "field:order_product.unit_price", NodeType: p.NodeField,
		Attrs: p.Attrs{"name": p.VString("単価"), "type": p.VString("currency")}})

	// --- Expressions (computed fields) ---
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "expr:calc_tax", NodeType: p.NodeExpression,
		Attrs: p.Attrs{"formula": p.VString("subtotal * tax_rate")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "expr:calc_total", NodeType: p.NodeExpression,
		Attrs: p.Attrs{"formula": p.VString("subtotal + tax")}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "expr:line_total", NodeType: p.NodeExpression,
		Attrs: p.Attrs{"formula": p.VString("quantity * unit_price")}})

	// --- Forms ---
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "form:order_entry", NodeType: p.NodeForm,
		Attrs: p.Attrs{"name": p.VString("受注入力フォーム")}})

	// --- Lists ---
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "list:order_list", NodeType: p.NodeList,
		Attrs: p.Attrs{"name": p.VString("受注一覧"), "columns": p.VStrings([]string{"subtotal", "tax", "total"})}})
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "list:order_lines", NodeType: p.NodeList,
		Attrs: p.Attrs{"name": p.VString("受注明細一覧"), "columns": p.VStrings([]string{"product_id", "quantity", "line_total"})}})

	// --- Roles ---
	log.Append(p.Event{Type: p.EventNodeAdded, NodeID: "role:sales", NodeType: p.NodeRole,
		Attrs: p.Attrs{"name": p.VString("営業担当")}})

	// --- Edges (provider → consumer) ---
	// Entity owns fields
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order.subtotal", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order.tax_rate", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order.tax", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order.total", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:customer", ToNode: "field:customer.name", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:product", ToNode: "field:product.id", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:product", ToNode: "field:product.name", Label: p.LabelDerives})

	// Relation edges
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "rel:order_product", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:product", ToNode: "rel:order_product", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:order", ToNode: "field:order_product.order_id", Label: p.LabelConstrains})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "entity:product", ToNode: "field:order_product.product_id", Label: p.LabelConstrains})

	// Expression dependencies
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.subtotal", ToNode: "expr:calc_tax", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.tax_rate", ToNode: "expr:calc_tax", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "expr:calc_tax", ToNode: "field:order.tax", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.subtotal", ToNode: "expr:calc_total", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.tax", ToNode: "expr:calc_total", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "expr:calc_total", ToNode: "field:order.total", Label: p.LabelDerives})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order_product.quantity", ToNode: "expr:line_total", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order_product.unit_price", ToNode: "expr:line_total", Label: p.LabelUses})

	// Form uses fields
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.subtotal", ToNode: "form:order_entry", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.total", ToNode: "form:order_entry", Label: p.LabelUses})

	// List uses fields
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.subtotal", ToNode: "list:order_list", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.tax", ToNode: "list:order_list", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order.total", ToNode: "list:order_list", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "field:order_product.quantity", ToNode: "list:order_lines", Label: p.LabelUses})
	log.Append(p.Event{Type: p.EventEdgeAdded, FromNode: "expr:line_total", ToNode: "list:order_lines", Label: p.LabelDerives})

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
		Attrs:  p.Attrs{"type": p.VString("decimal"), "precision": p.VNumber(4)},
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
		Attrs:  p.Attrs{"validation": p.VString("required")},
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

	// --- Scenario 4: Relation attribute change ---
	fmt.Println("=== Scenario 4: リレーション属性の変更 ===")
	relEvent := p.Event{
		Type:   p.EventAttrUpdated,
		NodeID: "field:order_product.quantity",
		Attrs:  p.Attrs{"type": p.VString("decimal")},
	}
	fmt.Printf("Event: %s on %s\n", relEvent.Type, relEvent.NodeID)

	impact4 := p.ImpactFromEvent(ctx, g, relEvent)
	fmt.Printf("Impact: %d nodes affected\n", len(impact4.Impacted))
	fmt.Println("Affected nodes:")
	for nodeID := range impact4.Impacted {
		fmt.Printf("  - %s: %s\n", nodeID, impact4.Explain(nodeID))
	}
	fmt.Println()

	// --- Evidence Path Demo ---
	fmt.Println("=== Evidence Path: なぜ form:order_entry が影響を受けるか ===")
	subtotalChange := p.Event{
		Type:   p.EventAttrUpdated,
		NodeID: "field:order.subtotal",
	}
	impactDemo := p.ImpactFromEvent(ctx, g, subtotalChange)
	if impactDemo.Impacted["form:order_entry"] {
		path := impactDemo.Path("form:order_entry")
		fmt.Printf("Path: ")
		for i, node := range path {
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
