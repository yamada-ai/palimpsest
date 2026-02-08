package expr

import "testing"

func TestParseRefAndCall(t *testing.T) {
	ast, diags := Parse(`IF($field:order.subtotal > 0, SUM($field:order.subtotal), 0)`)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if ast == nil {
		t.Fatalf("expected AST")
	}
	if ast.Kind != KindIf {
		t.Fatalf("expected IF node")
	}
}

func TestParseUnknownToken(t *testing.T) {
	_, diags := Parse(`$field:order.subtotal #`)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics")
	}
}

func TestOperatorPrecedence(t *testing.T) {
	ast, diags := Parse(`1 + 2 * 3`)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if ast.Kind != KindBinary || ast.Op != "+" {
		t.Fatalf("expected root +")
	}
	if ast.Right == nil || ast.Right.Kind != KindBinary || ast.Right.Op != "*" {
		t.Fatalf("expected right to be *")
	}

	ast2, diags := Parse(`1 * 2 + 3`)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if ast2.Kind != KindBinary || ast2.Op != "+" {
		t.Fatalf("expected root + for 1*2+3")
	}
	if ast2.Left == nil || ast2.Left.Kind != KindBinary || ast2.Left.Op != "*" {
		t.Fatalf("expected left to be *")
	}
}

func TestLogicalPrecedence(t *testing.T) {
	ast, diags := Parse(`1 < 2 && 3 < 4`)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if ast.Kind != KindBinary || ast.Op != "&&" {
		t.Fatalf("expected root &&")
	}
}

func TestPostfixChain(t *testing.T) {
	ast, diags := Parse(`foo.bar(baz).qux`)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if ast.Kind != KindProperty || ast.Property != "qux" {
		t.Fatalf("expected trailing property qux")
	}
	if ast.Object == nil || ast.Object.Kind != KindCall {
		t.Fatalf("expected call before property")
	}
	if ast.Object.Callee == nil || ast.Object.Callee.Kind != KindProperty {
		t.Fatalf("expected callee property")
	}
}

func TestTrailingTokens(t *testing.T) {
	_, diags := Parse(`1 2`)
	if len(diags) == 0 {
		t.Fatalf("expected trailing token diagnostics")
	}
}

func TestSpanByteOffset(t *testing.T) {
	ast, diags := Parse(`$field:顧客名`)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if ast.Kind != KindRef {
		t.Fatalf("expected ref")
	}
	if ast.Span.Start != 0 || ast.Span.End != len([]byte(`$field:顧客名`)) {
		t.Fatalf("unexpected span: %+v", ast.Span)
	}
}

func TestMissingOperand(t *testing.T) {
	_, diags := Parse(`1 +`)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics for missing operand")
	}
}

func TestIfMissingComma(t *testing.T) {
	_, diags := Parse(`IF(1, 2,)`)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics for missing operand")
	}
}
