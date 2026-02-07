package palimpsest

import "context"

// SimulationTxResult aggregates pre/post impact and validation around a transaction.
// PreValidate failure means the transaction was not applied.
type SimulationTxResult struct {
	Events []Event

	BeforeRevision int
	AfterRevision  int

	PreImpact   *ImpactResult
	PreValidate *ValidationResult

	Applied bool

	// Error captures Apply/Rollback failures (should be rare if ValidateEvent passes).
	// Rollbackに失敗した場合、Graphは破棄前提で扱う。
	Error error

	PostImpact   *ImpactResult
	PostValidate *ValidationResult
}

// SimulateTx runs the pre-impact, pre-validate, apply, and post-validate flow for a set of events.
// 一時的にGraphを変更してからDeltaで巻き戻す。
// 共有Graphを渡す場合は排他が必要。リクエスト専有なら不要。
func SimulateTx(ctx context.Context, g *Graph, events []Event) *SimulationTxResult {
	result := &SimulationTxResult{
		Events:         events,
		BeforeRevision: g.Revision(),
		AfterRevision:  g.Revision(),
	}

	result.PreImpact = ImpactFromEvents(ctx, g, events)
	if result.PreImpact.Cancelled {
		return result
	}

	// Apply all events (validating against the evolving graph) and collect deltas for rollback.
	deltas := make([]Delta, 0, len(events))
	for _, e := range events {
		vr := ValidateEvent(ctx, g, e)
		if vr.Cancelled {
			result.PreValidate = vr
			rollbackDeltas(g, deltas, result)
			return result
		}
		if !vr.Valid {
			result.PreValidate = vr
			rollbackDeltas(g, deltas, result)
			return result
		}
		delta, err := ApplyEvent(g, e)
		if err != nil {
			result.Error = err
			rollbackDeltas(g, deltas, result)
			return result
		}
		deltas = append(deltas, delta)
	}
	result.PreValidate = &ValidationResult{Valid: true, Errors: []ValidationError{}, Revision: g.Revision()}
	result.Applied = true
	result.AfterRevision = result.BeforeRevision + len(events)
	defer func() {
		rollbackDeltas(g, deltas, result)
	}()

	result.PostValidate = Validate(ctx, g)
	if result.PostValidate.Cancelled {
		return result
	}

	result.PostImpact = ImpactFromEvents(ctx, g, events)
	return result
}

func rollbackDeltas(g *Graph, deltas []Delta, result *SimulationTxResult) {
	for i := len(deltas) - 1; i >= 0; i-- {
		if err := RollbackDelta(g, deltas[i]); err != nil && result.Error == nil {
			result.Error = err
			return
		}
	}
}
