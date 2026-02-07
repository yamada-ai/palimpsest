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

	PostImpact   *ImpactResult
	PostValidate *ValidationResult
}

// SimulateEvent runs the pre-impact, pre-validate, apply, and post-validate flow.
// It never mutates the input graph; the event is applied to a cloned graph.
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

	// Clone and apply for post checks.
	after := g.Clone()
	applyEvent(after, e)
	result.Applied = true
	result.AfterRevision = result.BeforeRevision + 1

	result.PostValidate = Validate(ctx, after)
	if result.PostValidate.Cancelled {
		return result
	}

	result.PostImpact = ImpactFromEvent(ctx, after, e)
	return result
}
