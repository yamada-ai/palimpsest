package expr

import (
	"testing"

	core "github.com/user/palimpsest"
)

type evalResolver struct {
	vals map[string]core.Value
}

func (r evalResolver) ResolveValue(namespace string, path []string) (core.Value, bool) {
	key := namespace + ":" + joinPath(path)
	v, ok := r.vals[key]
	return v, ok
}

func TestEvalBasicMath(t *testing.T) {
	ast, diags := Parse(`1 + 2 * 3`)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	val, err := Eval(ast, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.(core.NumberValue) != core.NumberValue(7) {
		t.Fatalf("expected 7, got %v", val)
	}
}

func TestEvalIfCoalesce(t *testing.T) {
	ast, _ := Parse(`IF(true, 1, 2)`)
	val, err := Eval(ast, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.(core.NumberValue) != core.NumberValue(1) {
		t.Fatalf("expected 1, got %v", val)
	}

	ast2, _ := Parse(`COALESCE(null, "x")`)
	val2, err := Eval(ast2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val2.(core.StringValue) != core.StringValue("x") {
		t.Fatalf("expected x, got %v", val2)
	}
}

func TestEvalRef(t *testing.T) {
	ast, _ := Parse(`$field:order.total`)
	val, err := Eval(ast, evalResolver{vals: map[string]core.Value{"field:order.total": core.NumberValue(10)}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.(core.NumberValue) != core.NumberValue(10) {
		t.Fatalf("expected 10, got %v", val)
	}
}

func TestEvalTypeMismatch(t *testing.T) {
	ast, _ := Parse(`"a" + 1`)
	_, err := Eval(ast, nil)
	if err == nil {
		t.Fatalf("expected error for type mismatch")
	}
}

func TestEvalMissingArgs(t *testing.T) {
	ast, _ := Parse(`LEFT("x")`)
	_, err := Eval(ast, nil)
	if err == nil {
		t.Fatalf("expected arity error")
	}
}

func TestEvalUnresolvedRef(t *testing.T) {
	ast, _ := Parse(`$field:missing`)
	_, err := Eval(ast, evalResolver{vals: map[string]core.Value{}})
	if err == nil {
		t.Fatalf("expected unresolved ref error")
	}
}

func TestEvalCountNonArray(t *testing.T) {
	ast, _ := Parse(`COUNT(1)`)
	_, err := Eval(ast, nil)
	if err == nil {
		t.Fatalf("expected COUNT non-array error")
	}
}
