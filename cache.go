package palimpsest

import (
	"container/list"
	"sync"
)

// SnapshotCache is an LRU cache for immutable snapshots.
// Snapshots are safe to share across requests; callers should build request-local graphs
// via BaseGraph() / ReplayFromSnapshot().
// It is safe for concurrent use.
type SnapshotCache struct {
	cap int
	ll  *list.List
	m   map[int]*list.Element
	mu  sync.Mutex
}

type cacheEntry struct {
	rev  int
	snap *Snapshot
}

// NewSnapshotCache creates a cache with a fixed capacity.
func NewSnapshotCache(capacity int) *SnapshotCache {
	if capacity <= 0 {
		capacity = 1
	}
	return &SnapshotCache{
		cap: capacity,
		ll:  list.New(),
		m:   make(map[int]*list.Element),
	}
}

// Get returns a snapshot for the given revision if present.
// The snapshot is immutable and safe to share; do not mutate it directly.
func (c *SnapshotCache) Get(rev int) (*Snapshot, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.m[rev]; ok {
		c.ll.MoveToFront(ele)
		snap := ele.Value.(*cacheEntry).snap
		return snap, snap != nil
	}
	return nil, false
}

// Put inserts or updates a snapshot in the cache.
func (c *SnapshotCache) Put(snap *Snapshot) {
	if snap == nil {
		return
	}
	rev := snap.Revision()
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.m[rev]; ok {
		c.ll.MoveToFront(ele)
		ele.Value.(*cacheEntry).snap = snap
		return
	}
	entry := &cacheEntry{rev: rev, snap: snap}
	ele := c.ll.PushFront(entry)
	c.m[rev] = ele
	if c.ll.Len() > c.cap {
		c.evict()
	}
}

// Len returns the number of cached snapshots.
func (c *SnapshotCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

func (c *SnapshotCache) evict() {
	ele := c.ll.Back()
	if ele == nil {
		return
	}
	c.ll.Remove(ele)
	entry := ele.Value.(*cacheEntry)
	delete(c.m, entry.rev)
}
