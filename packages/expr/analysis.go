package expr

import core "github.com/user/palimpsest"

// Resolver resolves references to NodeIDs.
type Resolver interface {
	ResolveRef(namespace string, path []string) (core.NodeID, bool)
	ResolveEntityField(entityID core.NodeID, column string) (core.NodeID, bool)
}

// DepEntry represents a single dependency edge.
type DepEntry struct {
	NodeID core.NodeID
	Span   Span
}

// UnresolvedRef is a reference that could not be resolved.
type UnresolvedRef struct {
	Namespace string
	Path      []string
	Span      Span
}

// DepSummary is the output of static analysis.
type DepSummary struct {
	SelfID      core.NodeID
	TargetField core.NodeID

	ExactDeps   []DepEntry
	SchemaDeps  []DepEntry
	Unresolved  []UnresolvedRef
	Diagnostics []Diagnostic
}

// Analyze parses, resolves, and extracts dependencies.
func Analyze(root *Node, resolver Resolver, selfID, targetField core.NodeID) *DepSummary {
	s := &DepSummary{
		SelfID:      selfID,
		TargetField: targetField,
		ExactDeps:   []DepEntry{},
		SchemaDeps:  []DepEntry{},
		Unresolved:  []UnresolvedRef{},
		Diagnostics: []Diagnostic{},
	}
	seenExact := make(map[core.NodeID]bool)
	seenSchema := make(map[core.NodeID]bool)
	walk := func(n *Node, inCall bool) {}
	var walkNode func(n *Node, inCall bool)
	walkNode = func(n *Node, inCall bool) {
		if n == nil {
			return
		}
		switch n.Kind {
		case KindRef:
			if n.RefNamespace == "rel" && len(n.RefPath) < 2 {
				s.Diagnostics = append(s.Diagnostics, Diagnostic{Level: DiagError, Span: n.Span, Message: "relation attribute required", Code: "REL_ATTR_REQUIRED"})
				return
			}
			nodeID, ok := resolver.ResolveRef(n.RefNamespace, n.RefPath)
			if !ok {
				s.Unresolved = append(s.Unresolved, UnresolvedRef{Namespace: n.RefNamespace, Path: n.RefPath, Span: n.Span})
				s.Diagnostics = append(s.Diagnostics, Diagnostic{Level: DiagError, Span: n.Span, Message: "unresolved reference", Code: "UNRESOLVED_REF"})
				return
			}
			n.Resolved = true
			n.ResolvedID = nodeID
			if !seenExact[nodeID] {
				seenExact[nodeID] = true
				s.ExactDeps = append(s.ExactDeps, DepEntry{NodeID: nodeID, Span: n.Span})
			}
		case KindIdentifier:
			if !inCall {
				s.Diagnostics = append(s.Diagnostics, Diagnostic{Level: DiagError, Span: n.Span, Message: "undefined identifier", Code: "UNDEFINED_IDENTIFIER"})
			}
		case KindCall:
			calleeName := ""
			if n.Callee != nil && n.Callee.Kind == KindIdentifier {
				calleeName = n.Callee.Name
			}
			if calleeName == "" {
				s.Diagnostics = append(s.Diagnostics, Diagnostic{Level: DiagError, Span: n.Span, Message: "invalid call target", Code: "INVALID_CALL"})
			} else if !isAllowedFunc(calleeName) {
				s.Diagnostics = append(s.Diagnostics, Diagnostic{Level: DiagError, Span: n.Callee.Span, Message: "unknown function", Code: "UNKNOWN_FUNCTION"})
			}

			switch calleeName {
			case "LOOKUP":
				analyzeLookup(n, resolver, s, seenExact, seenSchema)
				return
			case "FILTER":
				analyzeFilter(n, resolver, s, seenExact)
				return
			default:
				for _, arg := range n.Args {
					walkNode(arg, false)
				}
				return
			}
		case KindBinary:
			walkNode(n.Left, false)
			walkNode(n.Right, false)
		case KindUnary:
			walkNode(n.Expr, false)
		case KindIf:
			walkNode(n.Cond, false)
			walkNode(n.Then, false)
			walkNode(n.Else, false)
		case KindProperty:
			walkNode(n.Object, false)
		case KindGroup:
			walkNode(n.Inner, false)
		case KindLiteral:
			return
		}
	}
	walk = func(n *Node, inCall bool) { walkNode(n, inCall) }
	walk(root, false)
	return s
}

func analyzeFilter(n *Node, resolver Resolver, s *DepSummary, seenExact map[core.NodeID]bool) {
	if len(n.Args) < 1 {
		s.Diagnostics = append(s.Diagnostics, Diagnostic{Level: DiagError, Span: n.Span, Message: "FILTER requires table_ref", Code: "BAD_ARITY"})
		return
	}
	tableRef := n.Args[0]
	if tableRef.Kind == KindRef {
		if nodeID, ok := resolver.ResolveRef(tableRef.RefNamespace, tableRef.RefPath); ok {
			if !seenExact[nodeID] {
				seenExact[nodeID] = true
				s.ExactDeps = append(s.ExactDeps, DepEntry{NodeID: nodeID, Span: tableRef.Span})
			}
		}
	}
	for _, arg := range n.Args[1:] {
		walkDeps(arg, resolver, s, seenExact)
	}
}

func analyzeLookup(n *Node, resolver Resolver, s *DepSummary, seenExact, seenSchema map[core.NodeID]bool) {
	if len(n.Args) < 3 {
		s.Diagnostics = append(s.Diagnostics, Diagnostic{Level: DiagError, Span: n.Span, Message: "LOOKUP requires (table_ref, key, column)", Code: "BAD_ARITY"})
		return
	}
	tableRef := n.Args[0]
	keyExpr := n.Args[1]
	colExpr := n.Args[2]

	walkDeps(keyExpr, resolver, s, seenExact)

	// Resolve table ref
	var entityID core.NodeID
	if tableRef.Kind == KindRef {
		if nodeID, ok := resolver.ResolveRef(tableRef.RefNamespace, tableRef.RefPath); ok {
			entityID = nodeID
		} else {
			s.Unresolved = append(s.Unresolved, UnresolvedRef{Namespace: tableRef.RefNamespace, Path: tableRef.RefPath, Span: tableRef.Span})
		}
	}

	if colExpr.Kind == KindLiteral {
		if str, ok := colExpr.Literal.(core.StringValue); ok {
			column := string(str)
			if entityID != "" {
				if fieldID, ok := resolver.ResolveEntityField(entityID, column); ok {
					if !seenExact[fieldID] {
						seenExact[fieldID] = true
						s.ExactDeps = append(s.ExactDeps, DepEntry{NodeID: fieldID, Span: colExpr.Span})
					}
					return
				}
				s.Diagnostics = append(s.Diagnostics, Diagnostic{Level: DiagError, Span: colExpr.Span, Message: "unknown column", Code: "UNKNOWN_COLUMN"})
				return
			}
		}
	}

	// Dynamic column: include deps of column expr and schema dep on entity.
	walkDeps(colExpr, resolver, s, seenExact)
	if entityID != "" && !seenSchema[entityID] {
		seenSchema[entityID] = true
		s.SchemaDeps = append(s.SchemaDeps, DepEntry{NodeID: entityID, Span: tableRef.Span})
	}
}

func walkDeps(n *Node, resolver Resolver, s *DepSummary, seenExact map[core.NodeID]bool) {
	if n == nil {
		return
	}
	switch n.Kind {
	case KindRef:
		nodeID, ok := resolver.ResolveRef(n.RefNamespace, n.RefPath)
		if ok && !seenExact[nodeID] {
			seenExact[nodeID] = true
			s.ExactDeps = append(s.ExactDeps, DepEntry{NodeID: nodeID, Span: n.Span})
		}
	case KindBinary:
		walkDeps(n.Left, resolver, s, seenExact)
		walkDeps(n.Right, resolver, s, seenExact)
	case KindUnary:
		walkDeps(n.Expr, resolver, s, seenExact)
	case KindIf:
		walkDeps(n.Cond, resolver, s, seenExact)
		walkDeps(n.Then, resolver, s, seenExact)
		walkDeps(n.Else, resolver, s, seenExact)
	case KindCall:
		for _, arg := range n.Args {
			walkDeps(arg, resolver, s, seenExact)
		}
	case KindProperty:
		walkDeps(n.Object, resolver, s, seenExact)
	case KindGroup:
		walkDeps(n.Inner, resolver, s, seenExact)
	}
}

func isAllowedFunc(name string) bool {
	switch name {
	case "IF", "COALESCE", "ROUND", "FLOOR", "CEIL", "ABS", "MIN", "MAX", "SUM",
		"CONCAT", "LEFT", "RIGHT", "LEN", "TRIM", "UPPER", "LOWER", "CONTAINS",
		"TODAY", "NOW", "DATE_ADD", "DATE_DIFF", "FORMAT_DATE",
		"LOOKUP", "FILTER", "COUNT":
		return true
	default:
		return false
	}
}
