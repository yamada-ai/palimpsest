package palimpsest

// Snapshot is an immutable graph checkpoint at a specific revision.
// リクエスト間で共有してよいが、直接ミューテートしてはいけない。
type Snapshot struct {
	revision int
	graph    *Graph
}

// SnapshotFromLog builds a snapshot at the given revision by replaying the log.
// 指定リビジョンまでのReplay結果をもとにスナップショットを作る。
func SnapshotFromLog(log *EventLog, revision int) *Snapshot {
	g := Replay(log, revision)
	return &Snapshot{revision: g.Revision(), graph: g.Clone()}
}

// SnapshotFromGraph captures an immutable snapshot of the given graph.
// The returned snapshot must be treated as read-only.
// 既存Graphの状態を読み取り専用スナップショットとして保持する。
func SnapshotFromGraph(g *Graph) *Snapshot {
	if g == nil {
		return nil
	}
	return &Snapshot{revision: g.Revision(), graph: g.Clone()}
}

// Revision returns the snapshot revision.
// スナップショットの基準リビジョンを返す。
func (s *Snapshot) Revision() int {
	if s == nil {
		return -1
	}
	return s.revision
}

// BaseGraph returns a cloned graph for safe use.
// Callers must treat the returned graph as request-local and mutable.
// 返却されるGraphはリクエストローカルとして扱い、自由に変更してよい。
func (s *Snapshot) BaseGraph() *Graph {
	if s == nil || s.graph == nil {
		return nil
	}
	return s.graph.Clone()
}

// ReplayFromSnapshot replays tail events on top of a snapshot to build a new graph.
// If toRevision is before the snapshot revision, it falls back to full replay.
// snapshot以降のイベントだけ適用して独立Graphを作る。
func ReplayFromSnapshot(s *Snapshot, log *EventLog, toRevision int) *Graph {
	if s == nil || s.graph == nil {
		return Replay(log, toRevision)
	}
	if toRevision < s.revision {
		return Replay(log, toRevision)
	}
	g := s.graph.Clone()
	IncrementalReplay(g, log, toRevision)
	return g
}
