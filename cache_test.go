package palimpsest

import "testing"

func TestSnapshotCachePutGetEvict(t *testing.T) {
	c := NewSnapshotCache(2)
	if c.Len() != 0 {
		t.Fatalf("expected empty cache")
	}

	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})

	s1 := SnapshotFromLog(log, 0)
	s2 := SnapshotFromLog(log, 1)
	s3 := SnapshotFromLog(log, 1)

	c.Put(s1)
	c.Put(s2)
	if c.Len() != 2 {
		t.Fatalf("expected cache size 2")
	}

	if _, ok := c.Get(s1.Revision()); !ok {
		t.Fatalf("expected to get s1")
	}

	c.Put(s3) // same revision should update
	if c.Len() != 2 {
		t.Fatalf("expected cache size 2 after update")
	}

	s4 := SnapshotFromLog(log, 0)
	c.Put(s4) // will evict LRU
	if c.Len() != 2 {
		t.Fatalf("expected cache size 2 after eviction")
	}
}

func TestSnapshotCacheLRUOrder(t *testing.T) {
	c := NewSnapshotCache(2)
	log := NewEventLog()
	log.Append(Event{Type: EventNodeAdded, NodeID: "a", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "b", NodeType: NodeField})
	log.Append(Event{Type: EventNodeAdded, NodeID: "c", NodeType: NodeField})

	s0 := SnapshotFromLog(log, 0)
	s1 := SnapshotFromLog(log, 1)
	s2 := SnapshotFromLog(log, 2)

	c.Put(s0)
	c.Put(s1)
	// Touch s0 to make s1 LRU
	if _, ok := c.Get(s0.Revision()); !ok {
		t.Fatalf("expected to get s0")
	}
	c.Put(s2) // should evict s1
	if _, ok := c.Get(s1.Revision()); ok {
		t.Fatalf("expected s1 to be evicted")
	}
}
