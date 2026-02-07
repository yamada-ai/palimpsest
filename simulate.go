package palimpsest

import "context"

// SimulationResult aggregates pre/post impact and validation around a single event.
// PreValidate failure means the event was not applied.
type SimulationResult struct {
	Event Event

	// BeforeRevision is the revision of the input graph.
	// AfterRevision is a virtual revision (BeforeRevision+1) for UI/explanation only.
	BeforeRevision int
	AfterRevision  int

	PreImpact   *ImpactResult
	PreValidate *ValidationResult

	Applied bool

	// Error captures Apply/Rollback failures (should be rare if ValidateEvent passes).
	// If rollback fails, the graph should be treated as corrupted and discarded.
	Error error

	PostImpact   *ImpactResult
	PostValidate *ValidationResult
}

// SimulateEvent runs the pre-impact, pre-validate, apply, and post-validate flow.
// It temporarily mutates the provided graph and then rolls back via delta.
// If the graph is shared, callers must ensure exclusive access during this call.
// NOTE: PreImpact may be empty for NodeAdded/EdgeAdded because the seed does not exist yet.
// In that case, rely on PostImpact for the "after" view.
func SimulateEvent(ctx context.Context, g *Graph, e Event) *SimulationResult {
	result := &SimulationResult{
		Event:          e,
		BeforeRevision: g.Revision(),
		AfterRevision:  g.Revision(),
	}

	result.PreImpact = ImpactFromEvent(ctx, g, e)
	if result.PreImpact.Cancelled {
		return result
	}

	result.PreValidate = ValidateEvent(ctx, g, e)
	if result.PreValidate.Cancelled || !result.PreValidate.Valid {
		return result
	}

	delta, err := ApplyEvent(g, e)
	if err != nil {
		result.Error = err
		return result
	}
	result.Applied = true
	result.AfterRevision = result.BeforeRevision + 1
	defer func() {
		if err := RollbackDelta(g, delta); err != nil && result.Error == nil {
			result.Error = err
		}
	}()

	result.PostValidate = Validate(ctx, g)
	if result.PostValidate.Cancelled {
		return result
	}

	result.PostImpact = ImpactFromEvent(ctx, g, e)
	return result
}
