package expr

import (
	"errors"
	"fmt"
	"math"
	"strings"

	core "github.com/user/palimpsest"
)

var ErrEval = errors.New("eval error")

// ValueResolver resolves references to runtime values.
type ValueResolver interface {
	ResolveValue(namespace string, path []string) (core.Value, bool)
}

// Eval evaluates the AST and returns a Value.
func Eval(root *Node, resolver ValueResolver) (core.Value, error) {
	return evalNode(root, resolver)
}

func evalNode(n *Node, resolver ValueResolver) (core.Value, error) {
	if n == nil {
		return nil, fmt.Errorf("%w: nil node", ErrEval)
	}
	switch n.Kind {
	case KindLiteral:
		return n.Literal, nil
	case KindRef:
		if resolver == nil {
			return nil, fmt.Errorf("%w: resolver not provided", ErrEval)
		}
		v, ok := resolver.ResolveValue(n.RefNamespace, n.RefPath)
		if !ok {
			return nil, fmt.Errorf("%w: unresolved ref", ErrEval)
		}
		return v, nil
	case KindIdentifier:
		return nil, fmt.Errorf("%w: identifier not callable", ErrEval)
	case KindGroup:
		return evalNode(n.Inner, resolver)
	case KindUnary:
		val, err := evalNode(n.Expr, resolver)
		if err != nil {
			return nil, err
		}
		switch n.Op {
		case "-":
			num, err := toNumber(val)
			if err != nil {
				return nil, err
			}
			return core.NumberValue(-num), nil
		case "!":
			b, err := toBool(val)
			if err != nil {
				return nil, err
			}
			return core.BoolValue(!b), nil
		default:
			return nil, fmt.Errorf("%w: unknown unary op", ErrEval)
		}
	case KindBinary:
		left, err := evalNode(n.Left, resolver)
		if err != nil {
			return nil, err
		}
		right, err := evalNode(n.Right, resolver)
		if err != nil {
			return nil, err
		}
		return evalBinary(n.Op, left, right)
	case KindIf:
		cond, err := evalNode(n.Cond, resolver)
		if err != nil {
			return nil, err
		}
		ok, err := toBool(cond)
		if err != nil {
			return nil, err
		}
		if ok {
			return evalNode(n.Then, resolver)
		}
		return evalNode(n.Else, resolver)
	case KindCall:
		if n.Callee == nil || n.Callee.Kind != KindIdentifier {
			return nil, fmt.Errorf("%w: invalid call target", ErrEval)
		}
		return evalCall(n.Callee.Name, n.Args, resolver)
	case KindProperty:
		obj, err := evalNode(n.Object, resolver)
		if err != nil {
			return nil, err
		}
		m, ok := obj.(core.ObjectValue)
		if !ok {
			return nil, fmt.Errorf("%w: property access on non-object", ErrEval)
		}
		if v, ok := m[n.Property]; ok {
			return v, nil
		}
		return core.NullValue{}, nil
	default:
		return nil, fmt.Errorf("%w: unknown node kind", ErrEval)
	}
}

func evalBinary(op string, left, right core.Value) (core.Value, error) {
	switch op {
	case "+", "-", "*", "/", "%":
		a, err := toNumber(left)
		if err != nil {
			return nil, err
		}
		b, err := toNumber(right)
		if err != nil {
			return nil, err
		}
		switch op {
		case "+":
			return core.NumberValue(a + b), nil
		case "-":
			return core.NumberValue(a - b), nil
		case "*":
			return core.NumberValue(a * b), nil
		case "/":
			return core.NumberValue(a / b), nil
		case "%":
			return core.NumberValue(math.Mod(a, b)), nil
		}
	case "==", "!=", "<", ">", "<=", ">=":
		return compare(op, left, right)
	case "&&", "||":
		a, err := toBool(left)
		if err != nil {
			return nil, err
		}
		b, err := toBool(right)
		if err != nil {
			return nil, err
		}
		if op == "&&" {
			return core.BoolValue(a && b), nil
		}
		return core.BoolValue(a || b), nil
	}
	return nil, fmt.Errorf("%w: unknown op", ErrEval)
}

func compare(op string, left, right core.Value) (core.Value, error) {
	switch l := left.(type) {
	case core.NumberValue:
		r, ok := right.(core.NumberValue)
		if !ok {
			return nil, fmt.Errorf("%w: type mismatch", ErrEval)
		}
		return core.BoolValue(cmpFloat(op, float64(l), float64(r))), nil
	case core.StringValue:
		r, ok := right.(core.StringValue)
		if !ok {
			return nil, fmt.Errorf("%w: type mismatch", ErrEval)
		}
		return core.BoolValue(cmpString(op, string(l), string(r))), nil
	case core.BoolValue:
		r, ok := right.(core.BoolValue)
		if !ok {
			return nil, fmt.Errorf("%w: type mismatch", ErrEval)
		}
		return core.BoolValue(cmpBool(op, bool(l), bool(r))), nil
	case core.NullValue:
		_, ok := right.(core.NullValue)
		if !ok {
			if op == "!=" {
				return core.BoolValue(true), nil
			}
			if op == "==" {
				return core.BoolValue(false), nil
			}
		}
		if op == "==" {
			return core.BoolValue(ok), nil
		}
		if op == "!=" {
			return core.BoolValue(!ok), nil
		}
		return nil, fmt.Errorf("%w: invalid null comparison", ErrEval)
	default:
		return nil, fmt.Errorf("%w: unsupported comparison", ErrEval)
	}
}

func cmpFloat(op string, a, b float64) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case "<":
		return a < b
	case ">":
		return a > b
	case "<=":
		return a <= b
	case ">=":
		return a >= b
	default:
		return false
	}
}

func cmpString(op string, a, b string) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case "<":
		return a < b
	case ">":
		return a > b
	case "<=":
		return a <= b
	case ">=":
		return a >= b
	default:
		return false
	}
}

func cmpBool(op string, a, b bool) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	default:
		return false
	}
}

func evalCall(name string, args []*Node, resolver ValueResolver) (core.Value, error) {
	switch name {
	case "IF":
		if len(args) != 3 {
			return nil, fmt.Errorf("%w: IF arity", ErrEval)
		}
		cond, err := evalNode(args[0], resolver)
		if err != nil {
			return nil, err
		}
		ok, err := toBool(cond)
		if err != nil {
			return nil, err
		}
		if ok {
			return evalNode(args[1], resolver)
		}
		return evalNode(args[2], resolver)
	case "COALESCE":
		for _, arg := range args {
			v, err := evalNode(arg, resolver)
			if err != nil {
				return nil, err
			}
			if _, isNull := v.(core.NullValue); !isNull {
				return v, nil
			}
		}
		return core.NullValue{}, nil
	case "SUM":
		var total float64
		for _, arg := range args {
			v, err := evalNode(arg, resolver)
			if err != nil {
				return nil, err
			}
			n, err := toNumber(v)
			if err != nil {
				return nil, err
			}
			total += n
		}
		return core.NumberValue(total), nil
	case "MIN":
		return evalMinMax(args, resolver, true)
	case "MAX":
		return evalMinMax(args, resolver, false)
	case "ABS":
		if len(args) < 1 {
			return nil, fmt.Errorf("%w: ABS arity", ErrEval)
		}
		v, err := evalNode(firstArg(args), resolver)
		if err != nil {
			return nil, err
		}
		n, err := toNumber(v)
		if err != nil {
			return nil, fmt.Errorf("%w: ABS expects number", ErrEval)
		}
		return core.NumberValue(math.Abs(n)), nil
	case "ROUND":
		if len(args) < 1 {
			return nil, fmt.Errorf("%w: ROUND arity", ErrEval)
		}
		v, err := evalNode(firstArg(args), resolver)
		if err != nil {
			return nil, err
		}
		n, err := toNumber(v)
		if err != nil {
			return nil, fmt.Errorf("%w: ROUND expects number", ErrEval)
		}
		return core.NumberValue(math.Round(n)), nil
	case "FLOOR":
		if len(args) < 1 {
			return nil, fmt.Errorf("%w: FLOOR arity", ErrEval)
		}
		v, err := evalNode(firstArg(args), resolver)
		if err != nil {
			return nil, err
		}
		n, err := toNumber(v)
		if err != nil {
			return nil, fmt.Errorf("%w: FLOOR expects number", ErrEval)
		}
		return core.NumberValue(math.Floor(n)), nil
	case "CEIL":
		if len(args) < 1 {
			return nil, fmt.Errorf("%w: CEIL arity", ErrEval)
		}
		v, err := evalNode(firstArg(args), resolver)
		if err != nil {
			return nil, err
		}
		n, err := toNumber(v)
		if err != nil {
			return nil, fmt.Errorf("%w: CEIL expects number", ErrEval)
		}
		return core.NumberValue(math.Ceil(n)), nil
	case "CONCAT":
		var out strings.Builder
		for _, arg := range args {
			v, err := evalNode(arg, resolver)
			if err != nil {
				return nil, err
			}
			s, err := toString(v)
			if err != nil {
				return nil, err
			}
			out.WriteString(s)
		}
		return core.StringValue(out.String()), nil
	case "LEFT":
		if len(args) < 2 {
			return nil, fmt.Errorf("%w: LEFT arity", ErrEval)
		}
		s, err := evalNode(firstArg(args), resolver)
		if err != nil {
			return nil, err
		}
		nv, err := evalNode(secondArg(args), resolver)
		if err != nil {
			return nil, err
		}
		str, err := toString(s)
		if err != nil {
			return nil, fmt.Errorf("%w: LEFT expects string", ErrEval)
		}
		n, err := toNumber(nv)
		if err != nil {
			return nil, fmt.Errorf("%w: LEFT expects number", ErrEval)
		}
		runes := []rune(str)
		if int(n) > len(runes) {
			return core.StringValue(str), nil
		}
		return core.StringValue(string(runes[:int(n)])), nil
	case "RIGHT":
		if len(args) < 2 {
			return nil, fmt.Errorf("%w: RIGHT arity", ErrEval)
		}
		s, err := evalNode(firstArg(args), resolver)
		if err != nil {
			return nil, err
		}
		nv, err := evalNode(secondArg(args), resolver)
		if err != nil {
			return nil, err
		}
		str, err := toString(s)
		if err != nil {
			return nil, fmt.Errorf("%w: RIGHT expects string", ErrEval)
		}
		n, err := toNumber(nv)
		if err != nil {
			return nil, fmt.Errorf("%w: RIGHT expects number", ErrEval)
		}
		runes := []rune(str)
		if int(n) > len(runes) {
			return core.StringValue(str), nil
		}
		return core.StringValue(string(runes[len(runes)-int(n):])), nil
	case "LEN":
		if len(args) < 1 {
			return nil, fmt.Errorf("%w: LEN arity", ErrEval)
		}
		s, err := evalNode(firstArg(args), resolver)
		if err != nil {
			return nil, err
		}
		str, err := toString(s)
		if err != nil {
			return nil, fmt.Errorf("%w: LEN expects string", ErrEval)
		}
		return core.NumberValue(float64(len([]rune(str)))), nil
	case "TRIM":
		if len(args) < 1 {
			return nil, fmt.Errorf("%w: TRIM arity", ErrEval)
		}
		s, err := evalNode(firstArg(args), resolver)
		if err != nil {
			return nil, err
		}
		str, err := toString(s)
		if err != nil {
			return nil, fmt.Errorf("%w: TRIM expects string", ErrEval)
		}
		return core.StringValue(strings.TrimSpace(str)), nil
	case "UPPER":
		if len(args) < 1 {
			return nil, fmt.Errorf("%w: UPPER arity", ErrEval)
		}
		s, err := evalNode(firstArg(args), resolver)
		if err != nil {
			return nil, err
		}
		str, err := toString(s)
		if err != nil {
			return nil, fmt.Errorf("%w: UPPER expects string", ErrEval)
		}
		return core.StringValue(strings.ToUpper(str)), nil
	case "LOWER":
		if len(args) < 1 {
			return nil, fmt.Errorf("%w: LOWER arity", ErrEval)
		}
		s, err := evalNode(firstArg(args), resolver)
		if err != nil {
			return nil, err
		}
		str, err := toString(s)
		if err != nil {
			return nil, fmt.Errorf("%w: LOWER expects string", ErrEval)
		}
		return core.StringValue(strings.ToLower(str)), nil
	case "CONTAINS":
		if len(args) < 2 {
			return nil, fmt.Errorf("%w: CONTAINS arity", ErrEval)
		}
		s, err := evalNode(firstArg(args), resolver)
		if err != nil {
			return nil, err
		}
		sub, err := evalNode(secondArg(args), resolver)
		if err != nil {
			return nil, err
		}
		str, err := toString(s)
		if err != nil {
			return nil, fmt.Errorf("%w: CONTAINS expects string", ErrEval)
		}
		substr, err := toString(sub)
		if err != nil {
			return nil, fmt.Errorf("%w: CONTAINS expects string", ErrEval)
		}
		return core.BoolValue(strings.Contains(str, substr)), nil
	case "COUNT":
		if len(args) < 1 {
			return nil, fmt.Errorf("%w: COUNT arity", ErrEval)
		}
		v, err := evalNode(firstArg(args), resolver)
		if err != nil {
			return nil, err
		}
		arr, ok := v.(core.ArrayValue)
		if !ok {
			return nil, fmt.Errorf("%w: COUNT expects array", ErrEval)
		}
		return core.NumberValue(float64(len(arr))), nil
	default:
		return nil, fmt.Errorf("%w: unsupported function", ErrEval)
	}
}

func evalMinMax(args []*Node, resolver ValueResolver, isMin bool) (core.Value, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%w: empty args", ErrEval)
	}
	first, err := evalNode(args[0], resolver)
	if err != nil {
		return nil, err
	}
	best, err := toNumber(first)
	if err != nil {
		return nil, err
	}
	for _, arg := range args[1:] {
		v, err := evalNode(arg, resolver)
		if err != nil {
			return nil, err
		}
		n, err := toNumber(v)
		if err != nil {
			return nil, err
		}
		if isMin && n < best {
			best = n
		}
		if !isMin && n > best {
			best = n
		}
	}
	return core.NumberValue(best), nil
}

func firstArg(args []*Node) *Node {
	if len(args) == 0 {
		return nil
	}
	return args[0]
}

func secondArg(args []*Node) *Node {
	if len(args) < 2 {
		return nil
	}
	return args[1]
}

func toNumber(v core.Value) (float64, error) {
	if v == nil {
		return 0, fmt.Errorf("%w: nil number", ErrEval)
	}
	switch n := v.(type) {
	case core.NumberValue:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("%w: expected number", ErrEval)
	}
}

func toBool(v core.Value) (bool, error) {
	if v == nil {
		return false, fmt.Errorf("%w: nil bool", ErrEval)
	}
	switch b := v.(type) {
	case core.BoolValue:
		return bool(b), nil
	default:
		return false, fmt.Errorf("%w: expected bool", ErrEval)
	}
}

func toString(v core.Value) (string, error) {
	if v == nil {
		return "", fmt.Errorf("%w: nil string", ErrEval)
	}
	switch s := v.(type) {
	case core.StringValue:
		return string(s), nil
	default:
		return "", fmt.Errorf("%w: expected string", ErrEval)
	}
}
