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
