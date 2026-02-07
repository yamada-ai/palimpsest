package palimpsest

// buildRelationLog creates a small N:M relation log for tests.
// リレーション（N:M）を含む小さなEventLogを返す。
func buildRelationLog() *EventLog {
	log := NewEventLog()

	log.Append(Event{Type: EventNodeAdded, NodeID: "entity:product", NodeType: NodeEntity})
	log.Append(Event{Type: EventNodeAdded, NodeID: "entity:tag", NodeType: NodeEntity})
	log.Append(Event{Type: EventNodeAdded, NodeID: "rel:product_tag", NodeType: NodeRelation})
	log.Append(Event{Type: EventNodeAdded, NodeID: "field:product_tag.product_id", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "field:product_tag.tag_id", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "field:product_tag.quantity", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "expr:tagged_products.filter", NodeType: NodeExpression})
	log.Append(Event{Type: EventNodeAdded, NodeID: "list:tagged_products", NodeType: NodeList})

	// Relation structure
	log.Append(Event{Type: EventEdgeAdded, FromNode: "entity:product", ToNode: "rel:product_tag", Label: LabelDerives})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "entity:tag", ToNode: "rel:product_tag", Label: LabelDerives})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "entity:product", ToNode: "field:product_tag.product_id", Label: LabelConstrains})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "entity:tag", ToNode: "field:product_tag.tag_id", Label: LabelConstrains})

	// Dependency flow
	log.Append(Event{Type: EventEdgeAdded, FromNode: "field:product_tag.quantity", ToNode: "expr:tagged_products.filter", Label: LabelUses})
	log.Append(Event{Type: EventEdgeAdded, FromNode: "expr:tagged_products.filter", ToNode: "list:tagged_products", Label: LabelDerives})

	return log
}
