package expr

import (
	"testing"

	core "github.com/user/palimpsest"
)

type testResolver struct {
	refs   map[string]core.NodeID
	fields map[string]core.NodeID
}

func (r testResolver) ResolveRef(namespace string, path []string) (core.NodeID, bool) {
	key := namespace + ":" + joinPath(path)
	id, ok := r.refs[key]
	return id, ok
}

func (r testResolver) ResolveEntityField(entityID core.NodeID, column string) (core.NodeID, bool) {
	key := string(entityID) + "." + column
	id, ok := r.fields[key]
	return id, ok
}

func joinPath(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "."
		}
		out += p
	}
	return out
}

func TestAnalyzeLookupLiteralColumn(t *testing.T) {
	ast, diags := Parse(`LOOKUP($entity:products, $field:order.product_id, "unit_price")`)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	resolver := testResolver{
		refs: map[string]core.NodeID{
			"entity:products":        "entity:products",
			"field:order.product_id": "field:order.product_id",
		},
		fields: map[string]core.NodeID{
			"entity:products.unit_price": "field:products.unit_price",
		},
	}
	summary := Analyze(ast, resolver, "expr:x", "field:order.total")
	if len(summary.ExactDeps) != 2 {
		t.Fatalf("expected exact deps, got %v", summary.ExactDeps)
	}
}

func TestAnalyzeLookupDynamicColumn(t *testing.T) {
	ast, diags := Parse(`LOOKUP($entity:products, $field:order.product_id, $param:target_column)`)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	resolver := testResolver{
		refs: map[string]core.NodeID{
			"entity:products":        "entity:products",
			"field:order.product_id": "field:order.product_id",
			"param:target_column":    "param:target_column",
		},
		fields: map[string]core.NodeID{},
	}
	summary := Analyze(ast, resolver, "expr:x", "field:order.total")
	if len(summary.SchemaDeps) != 1 {
		t.Fatalf("expected schema deps, got %v", summary.SchemaDeps)
	}
}

func TestAnalyzeFilterDeps(t *testing.T) {
	ast, diags := Parse(`FILTER($entity:orders, $field:order.total > 0)`)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	resolver := testResolver{
		refs: map[string]core.NodeID{
			"entity:orders":     "entity:orders",
			"field:order.total": "field:order.total",
		},
	}
	summary := Analyze(ast, resolver, "expr:x", "field:y")
	if len(summary.ExactDeps) < 2 {
		t.Fatalf("expected filter deps, got %v", summary.ExactDeps)
	}
}

func TestAnalyzeUnknownNamespace(t *testing.T) {
	ast, _ := Parse(`$unknown:foo`)
	resolver := testResolver{refs: map[string]core.NodeID{}}
	summary := Analyze(ast, resolver, "expr:x", "field:y")
	if len(summary.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics")
	}
}

func TestAnalyzeRelRequiresAttr(t *testing.T) {
	ast, _ := Parse(`$rel:order_product`)
	resolver := testResolver{refs: map[string]core.NodeID{"rel:order_product": "rel:order_product"}}
	summary := Analyze(ast, resolver, "expr:x", "field:y")
	if len(summary.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics")
	}
}

func TestAnalyzeIdentifierAlone(t *testing.T) {
	ast, _ := Parse(`foo`)
	resolver := testResolver{refs: map[string]core.NodeID{}}
	summary := Analyze(ast, resolver, "expr:x", "field:y")
	if len(summary.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics")
	}
}

func TestAnalyzeUnknownFunction(t *testing.T) {
	ast, _ := Parse(`FOO($field:x)`)
	resolver := testResolver{refs: map[string]core.NodeID{"field:x": "field:x"}}
	summary := Analyze(ast, resolver, "expr:x", "field:y")
	if len(summary.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics")
	}
}
