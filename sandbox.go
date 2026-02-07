package palimpsest

import (
	"context"
	"errors"
)

var ErrSandboxNoGraph = errors.New("sandbox: no graph available")

// Sandbox builds request-local graphs for speculative evaluation.
// It never mutates shared state; each simulation uses a fresh graph instance.
type Sandbox struct {
	snapshot *Snapshot
	log      *EventLog
	revision int
}

// NewSandbox creates a sandbox using a snapshot and an event log.
// revision is the target log revision for replay.
func NewSandbox(snapshot *Snapshot, log *EventLog, revision int) *Sandbox {
	return &Sandbox{snapshot: snapshot, log: log, revision: revision}
}

// BuildGraph constructs a request-local graph from snapshot + tail replay.
func (s *Sandbox) BuildGraph() *Graph {
	if s == nil || s.log == nil {
		return nil
	}
	// If snapshot is ahead of log, fall back to full replay.
	if s.snapshot != nil && s.snapshot.Revision() > s.log.Len()-1 {
		return Replay(s.log, s.revision)
	}
	return ReplayFromSnapshot(s.snapshot, s.log, s.revision)
}

// SimulateEvent runs a speculative simulation for a single event.
func (s *Sandbox) SimulateEvent(ctx context.Context, e Event) *SimulationResult {
	g := s.BuildGraph()
	if g == nil {
		return &SimulationResult{Event: e, Error: ErrSandboxNoGraph}
	}
	return SimulateEvent(ctx, g, e)
}

// SimulateTx runs a speculative simulation for a transaction (multiple events).
func (s *Sandbox) SimulateTx(ctx context.Context, events []Event) *SimulationTxResult {
	g := s.BuildGraph()
	if g == nil {
		return &SimulationTxResult{Events: events, Error: ErrSandboxNoGraph}
	}
	return SimulateTx(ctx, g, events)
}
