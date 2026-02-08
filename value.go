package palimpsest

import (
	"errors"
	"sort"
	"strconv"
	"strings"
)

// ValueKind represents a JSON-like value kind.
type ValueKind int

const (
	ValueNull ValueKind = iota
	ValueBool
	ValueNumber
	ValueString
	ValueArray
	ValueObject
)

var ErrUnsupportedAttrValue = errors.New("unsupported Attrs value type")

// Value is a JSON-like value for attributes and expression evaluation.
// 値は不変扱い（immutable）を前提とする。
type Value interface {
	Kind() ValueKind
	String() string
}

// NullValue represents null.
type NullValue struct{}

func (NullValue) Kind() ValueKind { return ValueNull }
func (NullValue) String() string  { return "null" }

// BoolValue represents a boolean.
type BoolValue bool

func (v BoolValue) Kind() ValueKind { return ValueBool }
func (v BoolValue) String() string  { return strconv.FormatBool(bool(v)) }

// NumberValue represents a number (float64).
type NumberValue float64

func (v NumberValue) Kind() ValueKind { return ValueNumber }
func (v NumberValue) String() string  { return strconv.FormatFloat(float64(v), 'f', -1, 64) }

// StringValue represents a string.
type StringValue string

func (v StringValue) Kind() ValueKind { return ValueString }
func (v StringValue) String() string  { return strconv.Quote(string(v)) }

// ArrayValue represents an array of values.
type ArrayValue []Value

func (v ArrayValue) Kind() ValueKind { return ValueArray }
func (v ArrayValue) String() string {
	if len(v) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(v))
	for _, item := range v {
		if item == nil {
			parts = append(parts, "null")
			continue
		}
		parts = append(parts, item.String())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ObjectValue represents a map of string keys to values.
// NOTE: callers must treat the map as immutable.
type ObjectValue map[string]Value

func (v ObjectValue) Kind() ValueKind { return ValueObject }
func (v ObjectValue) String() string {
	if len(v) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(v))
	for key := range v {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := v[key]
		if value == nil {
			parts = append(parts, strconv.Quote(key)+": null")
			continue
		}
		parts = append(parts, strconv.Quote(key)+": "+value.String())
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// Constructors for convenience.
func VNull() Value            { return NullValue{} }
func VBool(v bool) Value      { return BoolValue(v) }
func VNumber(v float64) Value { return NumberValue(v) }
func VString(v string) Value  { return StringValue(v) }

// VArray and VObject do not copy; callers must not mutate after construction.
func VArray(v []Value) Value           { return ArrayValue(v) }
func VObject(v map[string]Value) Value { return ObjectValue(v) }

// VStrings converts a string slice to a Value array.
func VStrings(v []string) Value {
	out := make([]Value, 0, len(v))
	for _, s := range v {
		out = append(out, VString(s))
	}
	return ArrayValue(out)
}

// FromAny converts legacy Attrs values to Value.
// DEPRECATED: Use Value constructors in new code.
func FromAny(v any) (Value, error) {
	switch x := v.(type) {
	case nil:
		return NullValue{}, nil
	case Value:
		return x, nil
	case bool:
		return BoolValue(x), nil
	case int:
		return NumberValue(float64(x)), nil
	case int64:
		return NumberValue(float64(x)), nil
	case uint:
		return NumberValue(float64(x)), nil
	case float32:
		return NumberValue(float64(x)), nil
	case float64:
		return NumberValue(x), nil
	case string:
		return StringValue(x), nil
	case []string:
		return VStrings(x), nil
	case []Value:
		return ArrayValue(x), nil
	case []any:
		out := make([]Value, 0, len(x))
		for _, item := range x {
			vv, err := FromAny(item)
			if err != nil {
				return nil, err
			}
			out = append(out, vv)
		}
		return ArrayValue(out), nil
	case map[string]Value:
		return ObjectValue(x), nil
	case map[string]any:
		out := make(map[string]Value, len(x))
		for k, item := range x {
			vv, err := FromAny(item)
			if err != nil {
				return nil, err
			}
			out[k] = vv
		}
		return ObjectValue(out), nil
	default:
		return nil, ErrUnsupportedAttrValue
	}
}

// MustFromAny converts legacy values and panics on failure.
// Use only in tests or one-off migrations.
func MustFromAny(v any) Value {
	out, err := FromAny(v)
	if err != nil {
		panic(err)
	}
	return out
}

// DeepCopyValue copies nested values for defensive use.
func DeepCopyValue(v Value) Value {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case ArrayValue:
		out := make(ArrayValue, len(x))
		for i, item := range x {
			out[i] = DeepCopyValue(item)
		}
		return out
	case ObjectValue:
		out := make(ObjectValue, len(x))
		for k, item := range x {
			out[k] = DeepCopyValue(item)
		}
		return out
	default:
		return v
	}
}
