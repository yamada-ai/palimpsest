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

func proposeForType(nodeID NodeID, nodeType NodeType) (string, string, []ProposedEvent) {
	switch nodeType {
	case NodeExpression:
		return "式の更新", "計算式が影響を受けるため再検討が必要", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "update formula"}}, Note: "式の更新が必要", Applyable: false},
		}
	case NodeField:
		return "フィールドの確認", "型/制約/既定値を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review field"}}, Note: "影響を受けるため確認", Applyable: false},
		}
	case NodeForm:
		return "フォームの確認", "表示/入力の整合性を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review form"}}, Note: "フォームの整合性確認", Applyable: false},
		}
	case NodeList:
		return "一覧の確認", "列定義/表示内容を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review list"}}, Note: "一覧の整合性確認", Applyable: false},
		}
	case NodeRole:
		return "権限の確認", "アクセス制御を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review role"}}, Note: "権限の見直し", Applyable: false},
		}
	case NodeEntity:
		return "エンティティの確認", "構造/関連の整合性を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review entity"}}, Note: "構造の見直し", Applyable: false},
		}
	case NodeParam:
		return "パラメータの確認", "依存先との整合性を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review param"}}, Note: "パラメータの確認", Applyable: false},
		}
	default:
		return "確認", "影響を受けるため関連する設定を確認", []ProposedEvent{
			{Event: Event{Type: EventAttrUpdated, NodeID: nodeID, Attrs: Attrs{"repair_hint": "review"}}, Note: "影響の確認", Applyable: false},
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
