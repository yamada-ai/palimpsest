package palimpsest

import (
	"context"
	"sort"
)

// ProposedEvent is a suggested event with optional notes.
type ProposedEvent struct {
	Event     Event
	Note      string
	// Applyable indicates whether this proposal can be applied as-is.
	// false means it is a placeholder hint and needs human review.
	Applyable bool
	// AutoLevel indicates how safely this proposal can be auto-applied.
	AutoLevel AutoLevel
}

// RepairAction is a concrete suggestion with optional event proposals.
type RepairAction struct {
	NodeID    NodeID
	NodeType  NodeType
	Severity  Severity
	Title     string
	Detail    string
	Evidence  string
	Proposals []ProposedEvent
}

// RepairPlanTx is a rich repair plan with concrete (but possibly non-applyable) proposals.
type RepairPlanTx struct {
	Event   Event
	Summary string
	Actions []RepairAction
}

// AutoLevel indicates how safely a proposal can be auto-applied.
type AutoLevel int

const (
	NeedsReview AutoLevel = iota
	AutoFixable
	ManualOnly
)

func (a AutoLevel) String() string {
	switch a {
	case NeedsReview:
		return "needs-review"
	case AutoFixable:
		return "auto-fixable"
	case ManualOnly:
		return "manual-only"
	default:
		return "unknown"
	}
}

// ComputeRepairPlanTx builds a rule-based repair plan with concrete proposals.
func ComputeRepairPlanTx(ctx context.Context, g *Graph, e Event) *RepairPlanTx {
	impact := ImpactFromEvent(ctx, g, e)
	return ComputeRepairPlanTxFromImpact(ctx, g, e, impact)
}

// ComputeRepairPlanTxFromImpact builds a plan from a precomputed impact result.
func ComputeRepairPlanTxFromImpact(ctx context.Context, g *Graph, e Event, impact *ImpactResult) *RepairPlanTx {
	plan := &RepairPlanTx{Event: e}
	if impact == nil {
		plan.Summary = "no impact result"
		return plan
	}
	if impact.Cancelled {
		plan.Summary = "cancelled"
		plan.Actions = nil
		return plan
	}

	// Special case: propose cascade delete if the event is NodeRemoved and has impact.
	if e.Type == EventNodeRemoved {
		if len(impact.Impacted) == 0 {
			plan.Summary = "no impacted nodes (excluding seeds)"
			return plan
		}
		if !impact.Impacted[e.NodeID] {
			// If the target isn't in impact, skip cascade proposals.
			return plan
		}
		if actions := proposeCascadeDelete(g, e.NodeID); len(actions) > 0 {
			plan.Actions = actions
			plan.Summary = buildSummaryFromActions(actions)
			return plan
		}
	}

	seedSet := make(map[NodeID]bool)
	for _, s := range impact.Seeds {
		seedSet[s] = true
	}

	actions := make([]RepairAction, 0, len(impact.Impacted))
	for nodeID := range impact.Impacted {
		select {
		case <-ctx.Done():
			plan.Summary = "cancelled"
			plan.Actions = nil
			return plan
		default:
		}

		if seedSet[nodeID] {
			continue
		}
		nodeType, ok := g.NodeTypeOf(nodeID)
		if !ok {
			continue
		}
		sev := severityForType(nodeType)
		title, detail, proposals := proposeForType(nodeID, nodeType)
		evidence := ""
		if explain := impact.Explain(nodeID); explain != "not impacted" {
			evidence = explain
		}

		actions = append(actions, RepairAction{
			NodeID:    nodeID,
			NodeType:  nodeType,
			Severity:  sev,
			Title:     title,
			Detail:    detail,
			Evidence:  evidence,
			Proposals: proposals,
		})
	}

	sortRepairActions(actions)
	plan.Actions = actions
	plan.Summary = buildSummaryFromActions(actions)
	return plan
}

func proposeCascadeDelete(g *Graph, nodeID NodeID) []RepairAction {
	if g == nil {
		return nil
	}
	if !g.HasNode(nodeID) {
		return nil
	}
	edges := append(g.IncomingEdges(nodeID), g.OutgoingEdges(nodeID)...)
	if len(edges) == 0 {
		return nil
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Label < edges[j].Label
	})
	proposals := make([]ProposedEvent, 0, len(edges)+1)
	for _, edge := range edges {
		auto := autoLevelForEdge(g, edge)
		proposals = append(proposals, ProposedEvent{
			Event:     Event{Type: EventEdgeRemoved, FromNode: edge.From, ToNode: edge.To, Label: edge.Label},
			Note:      "参照エッジを削除",
			Applyable: true,
			AutoLevel: auto,
		})
	}
	proposals = append(proposals, ProposedEvent{
		Event:     Event{Type: EventNodeRemoved, NodeID: nodeID},
		Note:      "依存解除後に削除",
		Applyable: true,
		AutoLevel: NeedsReview,
	})

	nodeType, ok := g.NodeTypeOf(nodeID)
	if !ok {
		nodeType = NodeField
	}
	return []RepairAction{{
		NodeID:    nodeID,
		NodeType:  nodeType,
		Severity:  SeverityCritical,
		Title:     "カスケード削除の提案",
		Detail:    "参照エッジを先に削除し、その後に対象ノードを削除します",
		Proposals: proposals,
	}}
}

func autoLevelForEdge(g *Graph, edge Edge) AutoLevel {
	// Conservative defaults: expressions and constraints require review.
	if edge.Label == LabelControls || edge.Label == LabelConstrains {
		return NeedsReview
	}
	toType, ok := g.NodeTypeOf(edge.To)
	if !ok {
		return NeedsReview
	}
	switch toType {
	case NodeForm, NodeList:
		return AutoFixable
	case NodeExpression:
		return NeedsReview
	case NodeField, NodeEntity, NodeRelation, NodeRole, NodeParam:
		return NeedsReview
	default:
		return NeedsReview
	}
}

func proposeForType(nodeID NodeID, nodeType NodeType) (string, string, []ProposedEvent) {
	switch nodeType {
	case NodeExpression:
		return "式の更新", "計算式が影響を受けるため再検討が必要", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "update formula"}}, Note: "式の更新が必要", Applyable: false, AutoLevel: ManualOnly},
		}
	case NodeField:
		return "フィールドの確認", "型/制約/既定値を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review field"}}, Note: "影響を受けるため確認", Applyable: false, AutoLevel: NeedsReview},
		}
	case NodeForm:
		return "フォームの確認", "表示/入力の整合性を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review form"}}, Note: "フォームの整合性確認", Applyable: false, AutoLevel: NeedsReview},
		}
	case NodeList:
		return "一覧の確認", "列定義/表示内容を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review list"}}, Note: "一覧の整合性確認", Applyable: false, AutoLevel: NeedsReview},
		}
	case NodeRole:
		return "権限の確認", "アクセス制御を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review role"}}, Note: "権限の見直し", Applyable: false, AutoLevel: NeedsReview},
		}
	case NodeEntity:
		return "エンティティの確認", "構造/関連の整合性を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review entity"}}, Note: "構造の見直し", Applyable: false, AutoLevel: NeedsReview},
		}
	case NodeRelation:
		return "リレーションの確認", "関係の整合性を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review relation"}}, Note: "関係の見直し", Applyable: false, AutoLevel: NeedsReview},
		}
	case NodeParam:
		return "パラメータの確認", "依存先との整合性を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review param"}}, Note: "パラメータの確認", Applyable: false, AutoLevel: NeedsReview},
		}
	default:
		return "確認", "影響を受けるため関連する設定を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review"}}, Note: "影響の確認", Applyable: false, AutoLevel: NeedsReview},
		}
	}
}

func sortRepairActions(actions []RepairAction) {
	// Severity asc, then NodeID
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Severity != actions[j].Severity {
			return actions[i].Severity < actions[j].Severity
		}
		return actions[i].NodeID < actions[j].NodeID
	})
}

func buildSummaryFromActions(actions []RepairAction) string {
	if len(actions) == 0 {
		return "no impacted nodes (excluding seeds)"
	}
	counts := map[Severity]int{}
	for _, a := range actions {
		counts[a.Severity]++
	}
	return formatSummary(counts)
}
