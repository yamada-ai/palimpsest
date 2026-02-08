package expr

import (
	"fmt"
	"strconv"

	core "github.com/user/palimpsest"
)

type parser struct {
	lex    *lexer
	tokens []token
	pos    int
	diags  []Diagnostic
}

func Parse(input string) (*Node, []Diagnostic) {
	p := &parser{lex: newLexer(input)}
	if err := p.lexAll(); err != nil {
		p.diags = append(p.diags, Diagnostic{Level: DiagError, Span: Span{}, Message: err.Error(), Code: "LEX_ERROR"})
		return nil, p.diags
	}
	node := p.parseExpression()
	if node == nil && len(p.diags) == 0 {
		p.diags = append(p.diags, Diagnostic{Level: DiagError, Span: Span{}, Message: "empty expression", Code: "PARSE_ERROR"})
	}
	if p.current().typ != tokEOF {
		t := p.current()
		p.diags = append(p.diags, Diagnostic{Level: DiagError, Span: t.span, Message: "unexpected trailing tokens", Code: "PARSE_ERROR"})
	}
	return node, p.diags
}

func (p *parser) lexAll() error {
	for {
		tok, err := p.lex.nextToken()
		if err != nil {
			return err
		}
		p.tokens = append(p.tokens, tok)
		if tok.typ == tokEOF {
			break
		}
	}
	return nil
}

func (p *parser) current() token {
	if p.pos >= len(p.tokens) {
		return token{typ: tokEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) next() token {
	t := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return t
}

func (p *parser) expect(tt tokenType, msg string) token {
	t := p.current()
	if t.typ != tt {
		p.diags = append(p.diags, Diagnostic{Level: DiagError, Span: t.span, Message: msg, Code: "PARSE_ERROR"})
		return token{typ: tt, span: t.span}
	}
	p.next()
	return t
}

func (p *parser) parseExpression() *Node {
	return p.parseConditional()
}

func (p *parser) parseConditional() *Node {
	// IF(...) special form
	if p.current().typ == tokIdentifier && p.current().lit == "IF" {
		if p.peekType(1) == tokLParen {
			start := p.current().span.Start
			p.next() // IF
			p.expect(tokLParen, "expected '(' after IF")
			cond := p.parseExpression()
			p.expect(tokComma, "expected ',' after IF condition")
			thenExpr := p.parseExpression()
			p.expect(tokComma, "expected ',' after IF then")
			elseExpr := p.parseExpression()
			endTok := p.expect(tokRParen, "expected ')' to close IF")
			return &Node{
				Kind: KindIf,
				Span: Span{Start: start, End: endTok.span.End},
				Cond: cond,
				Then: thenExpr,
				Else: elseExpr,
			}
		}
	}
	return p.parseLogicalOr()
}

func (p *parser) parseLogicalOr() *Node {
	left := p.parseLogicalAnd()
	for p.current().typ == tokOp && p.current().lit == "||" {
		op := p.next()
		right := p.parseLogicalAnd()
		if left == nil || right == nil {
			p.diags = append(p.diags, Diagnostic{Level: DiagError, Span: op.span, Message: "missing operand", Code: "PARSE_ERROR"})
			return left
		}
		left = &Node{Kind: KindBinary, Span: Span{Start: left.Span.Start, End: right.Span.End}, Op: op.lit, Left: left, Right: right}
	}
	return left
}

func (p *parser) parseLogicalAnd() *Node {
	left := p.parseComparison()
	for p.current().typ == tokOp && p.current().lit == "&&" {
		op := p.next()
		right := p.parseComparison()
		if left == nil || right == nil {
			p.diags = append(p.diags, Diagnostic{Level: DiagError, Span: op.span, Message: "missing operand", Code: "PARSE_ERROR"})
			return left
		}
		left = &Node{Kind: KindBinary, Span: Span{Start: left.Span.Start, End: right.Span.End}, Op: op.lit, Left: left, Right: right}
	}
	return left
}

func (p *parser) parseComparison() *Node {
	left := p.parseAdditive()
	for p.current().typ == tokOp && (p.current().lit == "==" || p.current().lit == "!=" || p.current().lit == "<" || p.current().lit == ">" || p.current().lit == "<=" || p.current().lit == ">=") {
		op := p.next()
		right := p.parseAdditive()
		if left == nil || right == nil {
			p.diags = append(p.diags, Diagnostic{Level: DiagError, Span: op.span, Message: "missing operand", Code: "PARSE_ERROR"})
			return left
		}
		left = &Node{Kind: KindBinary, Span: Span{Start: left.Span.Start, End: right.Span.End}, Op: op.lit, Left: left, Right: right}
	}
	return left
}

func (p *parser) parseAdditive() *Node {
	left := p.parseMultiplicative()
	for p.current().typ == tokOp && (p.current().lit == "+" || p.current().lit == "-") {
		op := p.next()
		right := p.parseMultiplicative()
		if left == nil || right == nil {
			p.diags = append(p.diags, Diagnostic{Level: DiagError, Span: op.span, Message: "missing operand", Code: "PARSE_ERROR"})
			return left
		}
		left = &Node{Kind: KindBinary, Span: Span{Start: left.Span.Start, End: right.Span.End}, Op: op.lit, Left: left, Right: right}
	}
	return left
}

func (p *parser) parseMultiplicative() *Node {
	left := p.parseUnary()
	for p.current().typ == tokOp && (p.current().lit == "*" || p.current().lit == "/" || p.current().lit == "%") {
		op := p.next()
		right := p.parseUnary()
		if left == nil || right == nil {
			p.diags = append(p.diags, Diagnostic{Level: DiagError, Span: op.span, Message: "missing operand", Code: "PARSE_ERROR"})
			return left
		}
		left = &Node{Kind: KindBinary, Span: Span{Start: left.Span.Start, End: right.Span.End}, Op: op.lit, Left: left, Right: right}
	}
	return left
}

func (p *parser) parseUnary() *Node {
	if p.current().typ == tokOp && (p.current().lit == "-" || p.current().lit == "!") {
		op := p.next()
		expr := p.parseUnary()
		return &Node{Kind: KindUnary, Span: Span{Start: op.span.Start, End: expr.Span.End}, Op: op.lit, Expr: expr}
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() *Node {
	node := p.parsePrimary()
	for {
		switch p.current().typ {
		case tokDot:
			p.next()
			id := p.expect(tokIdentifier, "expected identifier after '.'")
			node = &Node{Kind: KindProperty, Span: Span{Start: node.Span.Start, End: id.span.End}, Object: node, Property: id.lit}
		case tokLParen:
			start := node.Span.Start
			p.next()
			args := []*Node{}
			if p.current().typ != tokRParen {
				args = append(args, p.parseExpression())
				for p.current().typ == tokComma {
					p.next()
					args = append(args, p.parseExpression())
				}
			}
			end := p.expect(tokRParen, "expected ')' after arguments")
			node = &Node{Kind: KindCall, Span: Span{Start: start, End: end.span.End}, Callee: node, Args: args}
		default:
			return node
		}
	}
}

func (p *parser) parsePrimary() *Node {
	t := p.current()
	switch t.typ {
	case tokDollar:
		start := t.span.Start
		p.next()
		ns := p.expect(tokIdentifier, "expected namespace").lit
		p.expect(tokColon, "expected ':' after namespace")
		nameTok := p.expect(tokIdentifier, "expected name")
		path := []string{nameTok.lit}
		end := nameTok.span.End
		for p.current().typ == tokDot {
			p.next()
			part := p.expect(tokIdentifier, "expected name segment")
			path = append(path, part.lit)
			end = part.span.End
		}
		return &Node{Kind: KindRef, Span: Span{Start: start, End: end}, RefNamespace: ns, RefPath: path}
	case tokNumber:
		p.next()
		num, _ := strconv.ParseFloat(t.lit, 64)
		return &Node{Kind: KindLiteral, Span: t.span, Literal: core.NumberValue(num)}
	case tokString:
		p.next()
		return &Node{Kind: KindLiteral, Span: t.span, Literal: core.StringValue(t.lit)}
	case tokBool:
		p.next()
		val := t.lit == "true"
		return &Node{Kind: KindLiteral, Span: t.span, Literal: core.BoolValue(val)}
	case tokNull:
		p.next()
		return &Node{Kind: KindLiteral, Span: t.span, Literal: core.NullValue{}}
	case tokIdentifier:
		p.next()
		return &Node{Kind: KindIdentifier, Span: t.span, Name: t.lit}
	case tokLParen:
		start := t.span.Start
		p.next()
		expr := p.parseExpression()
		end := p.expect(tokRParen, "expected ')'")
		return &Node{Kind: KindGroup, Span: Span{Start: start, End: end.span.End}, Inner: expr}
	default:
		p.diags = append(p.diags, Diagnostic{Level: DiagError, Span: t.span, Message: fmt.Sprintf("unexpected token: %s", t.lit), Code: "PARSE_ERROR"})
		p.next()
		return nil
	}
}

func (p *parser) peekType(offset int) tokenType {
	idx := p.pos + offset
	if idx >= len(p.tokens) {
		return tokEOF
	}
	return p.tokens[idx].typ
}
