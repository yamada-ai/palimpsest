package palimpsest

import "testing"

func TestValueKinds(t *testing.T) {
	tests := []struct {
		name string
		v    Value
		k    ValueKind
	}{
		{name: "null", v: VNull(), k: ValueNull},
		{name: "bool", v: VBool(true), k: ValueBool},
		{name: "number", v: VNumber(1.5), k: ValueNumber},
		{name: "string", v: VString("x"), k: ValueString},
		{name: "array", v: VArray([]Value{VNumber(1)}), k: ValueArray},
		{name: "object", v: VObject(map[string]Value{"a": VNumber(1)}), k: ValueObject},
	}

	for _, tt := range tests {
		if tt.v.Kind() != tt.k {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.k, tt.v.Kind())
		}
	}
}

func TestValueString(t *testing.T) {
	v := VObject(map[string]Value{
		"a": VNumber(1),
		"b": VString("x"),
	})
	out := v.String()
	if out == "" {
		t.Fatalf("expected non-empty string")
	}
}

func TestFromAny(t *testing.T) {
	cases := []struct {
		in   any
		kind ValueKind
	}{
		{in: nil, kind: ValueNull},
		{in: true, kind: ValueBool},
		{in: 1, kind: ValueNumber},
		{in: int64(2), kind: ValueNumber},
		{in: uint(3), kind: ValueNumber},
		{in: float32(1.25), kind: ValueNumber},
		{in: 1.25, kind: ValueNumber},
		{in: "x", kind: ValueString},
		{in: []string{"a"}, kind: ValueArray},
		{in: []any{"a", 1}, kind: ValueArray},
		{in: map[string]any{"a": "b"}, kind: ValueObject},
	}
	for _, c := range cases {
		val, err := FromAny(c.in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val.Kind() != c.kind {
			t.Fatalf("expected %v, got %v", c.kind, val.Kind())
		}
	}
}

func TestFromAnyUnsupported(t *testing.T) {
	_, err := FromAny(struct{}{})
	if err == nil {
		t.Fatalf("expected error for unsupported type")
	}
}
