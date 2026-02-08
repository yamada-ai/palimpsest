package expr

import core "github.com/user/palimpsest"

// BuildDepEvents converts DepSummary into core EdgeAdded events.
// Graph uses only "uses" edges; SchemaDeps/ExactDeps distinction stays in DepSummary.
// If SelfID or TargetField is empty, it returns nil.
func BuildDepEvents(summary *DepSummary) []core.Event {
	if summary == nil {
		return nil
	}
	if summary.SelfID == "" || summary.TargetField == "" {
		return nil
	}
	events := make([]core.Event, 0, len(summary.ExactDeps)+len(summary.SchemaDeps)+1)
	seen := make(map[string]bool)
	add := func(from, to core.NodeID) {
		key := string(from) + "->" + string(to)
		if seen[key] {
			return
		}
		seen[key] = true
		events = append(events, core.Event{
			Type:     core.EventEdgeAdded,
			FromNode: from,
			ToNode:   to,
			Label:    core.LabelUses,
		})
	}
	for _, dep := range summary.ExactDeps {
		add(dep.NodeID, summary.SelfID)
	}
	for _, dep := range summary.SchemaDeps {
		add(dep.NodeID, summary.SelfID)
	}
	add(summary.SelfID, summary.TargetField)
	return events
}
