package palimpsest

import (
	"context"
	"sort"
)

type Severity int

const (
	SeverityCritical Severity = iota
	SeverityHigh
	SeverityMedium
	SeverityLow
)

func (s Severity) String() string {
	switch s {
	case SeverityCritical:
		return "critical"
	case SeverityHigh:
		return "high"
	case SeverityMedium:
		return "medium"
	case SeverityLow:
		return "low"
	default:
		return "unknown"
	}
}

type RepairSuggestion struct {
	NodeID   NodeID
	NodeType NodeType
	Severity Severity
	Message  string
	Evidence string
}

type RepairPlan struct {
	Event       Event
	Summary     string
	Suggestions []RepairSuggestion
}

// ComputeRepairPlan builds a rule-based repair plan from an impact result.
// It prioritizes nodes by type and uses evidence paths for explanation.
func ComputeRepairPlan(ctx context.Context, g *Graph, e Event) *RepairPlan {
	impact := ImpactFromEvent(ctx, g, e)
	return ComputeRepairPlanFromImpact(ctx, g, e, impact)
}

// ComputeRepairPlanFromImpact builds a rule-based repair plan from a precomputed impact.
func ComputeRepairPlanFromImpact(ctx context.Context, g *Graph, e Event, impact *ImpactResult) *RepairPlan {
	plan := &RepairPlan{Event: e}
	if impact == nil {
		plan.Summary = "no impact result"
		return plan
	}

	seedSet := make(map[NodeID]bool)
	for _, s := range impact.Seeds {
		seedSet[s] = true
	}

	suggestions := make([]RepairSuggestion, 0, len(impact.Impacted))
	for nodeID := range impact.Impacted {
		select {
		case <-ctx.Done():
			plan.Summary = "cancelled"
			plan.Suggestions = suggestions
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
		msg := messageForType(nodeType)
		var evidence string
		if explain := impact.Explain(nodeID); explain != "not impacted" {
			evidence = explain
		}
		suggestions = append(suggestions, RepairSuggestion{
			NodeID:   nodeID,
			NodeType: nodeType,
			Severity: sev,
			Message:  msg,
			Evidence: evidence,
		})
	}

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Severity != suggestions[j].Severity {
			return suggestions[i].Severity < suggestions[j].Severity
		}
		return suggestions[i].NodeID < suggestions[j].NodeID
	})

	plan.Suggestions = suggestions
	plan.Summary = buildSummary(suggestions)
	return plan
}

func severityForType(t NodeType) Severity {
	switch t {
	case NodeExpression:
		return SeverityCritical
	case NodeField:
		return SeverityHigh
	case NodeForm, NodeList:
		return SeverityMedium
	case NodeRole, NodeEntity, NodeParam:
		return SeverityLow
	default:
		return SeverityLow
	}
}

func messageForType(t NodeType) string {
	switch t {
	case NodeExpression:
		return "式が影響を受けるため、計算式の見直しが必要です"
	case NodeField:
		return "フィールドが影響を受けるため、型/制約/既定値を確認してください"
	case NodeForm:
		return "フォームが影響を受けるため、表示/入力の整合性を確認してください"
	case NodeList:
		return "一覧が影響を受けるため、列定義/表示内容を確認してください"
	case NodeRole:
		return "権限が影響を受けるため、アクセス制御を確認してください"
	case NodeEntity:
		return "エンティティが影響を受けるため、関連する構造を確認してください"
	case NodeParam:
		return "パラメータ変更の影響があるため、依存先を確認してください"
	default:
		return "影響を受けるため、関連する設定を確認してください"
	}
}

func buildSummary(suggestions []RepairSuggestion) string {
	if len(suggestions) == 0 {
		return "no impacted nodes (excluding seeds)"
	}
	counts := map[Severity]int{}
	for _, s := range suggestions {
		counts[s.Severity]++
	}
	return "repair suggestions generated"
}
