package expr

import core "github.com/user/palimpsest"

// Span represents a range in the source string (UTF-8 byte offsets).
type Span struct {
	Start int
	End   int
}

// NodeKind represents AST node type.
type NodeKind int

const (
	KindRef NodeKind = iota
	KindLiteral
	KindIdentifier
	KindBinary
	KindUnary
	KindCall
	KindIf
	KindProperty
	KindGroup
)

// Node is a generic AST node.
type Node struct {
	Kind NodeKind
	Span Span

	// Ref
	RefNamespace string
	RefPath      []string
	ResolvedID   core.NodeID
	Resolved     bool

	// Literal
	Literal core.Value

	// Identifier
	Name string

	// Operators
	Op string

	// Binary
	Left  *Node
	Right *Node

	// Unary
	Expr *Node

	// Call
	Callee *Node // Identifier node
	Args   []*Node

	// If
	Cond *Node
	Then *Node
	Else *Node

	// Property
	Object   *Node
	Property string

	// Group
	Inner *Node
}
