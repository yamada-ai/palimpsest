package palimpsest

// EventType represents the type of configuration change event.
// ここでのイベントは「最小単位の変更」を表す。
type EventType int

const (
	EventNodeAdded EventType = iota
	EventNodeRemoved
	EventEdgeAdded
	EventEdgeRemoved
	EventAttrUpdated
	EventTransactionMarker
)

func (e EventType) String() string {
	switch e {
	case EventNodeAdded:
		return "NodeAdded"
	case EventNodeRemoved:
		return "NodeRemoved"
	case EventEdgeAdded:
		return "EdgeAdded"
	case EventEdgeRemoved:
		return "EdgeRemoved"
	case EventAttrUpdated:
		return "AttrUpdated"
	case EventTransactionMarker:
		return "TransactionMarker"
	default:
		return "Unknown"
	}
}

// EdgeLabel represents the type of dependency relationship.
// provider → consumer の関係を、影響の性質に応じてラベル付けする。
type EdgeLabel string

const (
	LabelUses       EdgeLabel = "uses"       // data dependency
	LabelDerives    EdgeLabel = "derives"    // structural inheritance
	LabelControls   EdgeLabel = "controls"   // behavioral control
	LabelConstrains EdgeLabel = "constrains" // validation constraint
)

// NodeID is a unique identifier for a node.
// テナント内で一意であることを想定する。
type NodeID string

// NodeType represents the kind of configuration element.
// MVPで扱う構成要素の種類。
type NodeType string

const (
	NodeEntity     NodeType = "Entity"
	NodeField      NodeType = "Field"
	NodeForm       NodeType = "Form"
	NodeList       NodeType = "List"
	NodeExpression NodeType = "Expression"
	NodeRole       NodeType = "Role"
)

// Attrs holds arbitrary node attributes.
// Contract: values must be JSON-serializable scalars (string/number/bool) or simple arrays.
// Do not store nested maps, slices of maps, or pointers; callers must treat values as immutable.
type Attrs map[string]any

// Event represents a single atomic change to the configuration graph.
// イベントは自己完結的で、ログの追記のみで運用する。
type Event struct {
	Type EventType

	// For NodeAdded/NodeRemoved/AttrUpdated
	NodeID   NodeID
	NodeType NodeType
	Attrs    Attrs

	// For EdgeAdded/EdgeRemoved
	FromNode NodeID
	ToNode   NodeID
	Label    EdgeLabel

	// For TransactionMarker
	TxID    string
	TxMeta  map[string]string
}

// Seeds extracts the impact seeds from an event.
// 仕様: Impact Seeds = 基本は consumer (ToNode)、Validation Seeds = 両端。
func (e Event) ImpactSeeds() []NodeID {
	switch e.Type {
	case EventNodeAdded, EventNodeRemoved, EventAttrUpdated:
		return []NodeID{e.NodeID}
	case EventEdgeAdded, EventEdgeRemoved:
		// Default: only the consumer (ToNode) is affected
		// Exception: if label is controls/constrains, include FromNode
		if e.Label == LabelControls || e.Label == LabelConstrains {
			return []NodeID{e.FromNode, e.ToNode}
		}
		return []NodeID{e.ToNode}
	case EventTransactionMarker:
		return nil
	default:
		return nil
	}
}

// ValidationSeeds returns both endpoints for constraint checking.
// 参照整合性などの局所検査に使う。
func (e Event) ValidationSeeds() []NodeID {
	switch e.Type {
	case EventNodeAdded, EventNodeRemoved, EventAttrUpdated:
		return []NodeID{e.NodeID}
	case EventEdgeAdded, EventEdgeRemoved:
		return []NodeID{e.FromNode, e.ToNode}
	case EventTransactionMarker:
		return nil
	default:
		return nil
	}
}

// EventLog is an append-only sequence of events.
// Source of Truth として扱う。
type EventLog struct {
	events []Event
}

// NewEventLog creates an empty event log
func NewEventLog() *EventLog {
	return &EventLog{events: make([]Event, 0)}
}

// Append adds an event to the log and returns its offset (revision).
// Revision はログ内オフセットとして扱う。
func (l *EventLog) Append(e Event) int {
	l.events = append(l.events, e)
	return len(l.events) - 1
}

// Len returns the current length (latest revision + 1)
func (l *EventLog) Len() int {
	return len(l.events)
}

// Get returns the event at a given offset
func (l *EventLog) Get(offset int) (Event, bool) {
	if offset < 0 || offset >= len(l.events) {
		return Event{}, false
	}
	return l.events[offset], true
}

// Range returns events from start (inclusive) to end (exclusive)
func (l *EventLog) Range(start, end int) []Event {
	if start < 0 {
		start = 0
	}
	if end > len(l.events) {
		end = len(l.events)
	}
	if start >= end {
		return nil
	}
	result := make([]Event, end-start)
	copy(result, l.events[start:end])
	return result
}
