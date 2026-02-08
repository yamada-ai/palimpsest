package palimpsest

import (
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
