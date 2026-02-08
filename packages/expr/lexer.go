package expr

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

type tokenType int

const (
	tokEOF tokenType = iota
	tokIdentifier
	tokNumber
	tokString
	tokBool
	tokNull
	tokDollar
	tokColon
	tokDot
	tokComma
	tokLParen
	tokRParen
	tokOp
)

type token struct {
	typ  tokenType
	lit  string
	span Span
}

type lexer struct {
	src string
	pos int // byte offset
}

func newLexer(input string) *lexer {
	return &lexer{src: input, pos: 0}
}

func (l *lexer) nextRune() (rune, int) {
	if l.pos >= len(l.src) {
		return 0, 0
	}
	r, size := utf8.DecodeRuneInString(l.src[l.pos:])
	l.pos += size
	return r, size
}

func (l *lexer) peekRune() (rune, int) {
	if l.pos >= len(l.src) {
		return 0, 0
	}
	return utf8.DecodeRuneInString(l.src[l.pos:])
}

func (l *lexer) emit(typ tokenType, start, end int, lit string) token {
	return token{typ: typ, lit: lit, span: Span{Start: start, End: end}}
}

func (l *lexer) skipSpaces() {
	for {
		r, _ := l.peekRune()
		if !unicode.IsSpace(r) {
			return
		}
		l.nextRune()
	}
}

func (l *lexer) nextToken() (token, error) {
	l.skipSpaces()
	start := l.pos
	ch, _ := l.nextRune()
	if ch == 0 {
		return l.emit(tokEOF, start, start, ""), nil
	}

	switch ch {
	case '$':
		return l.emit(tokDollar, start, l.pos, "$"), nil
	case ':':
		return l.emit(tokColon, start, l.pos, ":"), nil
	case '.':
		return l.emit(tokDot, start, l.pos, "."), nil
	case ',':
		return l.emit(tokComma, start, l.pos, ","), nil
	case '(':
		return l.emit(tokLParen, start, l.pos, "("), nil
	case ')':
		return l.emit(tokRParen, start, l.pos, ")"), nil
	case '"':
		for {
			n, _ := l.nextRune()
			if n == 0 {
				return token{}, fmt.Errorf("unterminated string")
			}
			if n == '"' {
				break
			}
			if n == '\\' {
				if r, _ := l.nextRune(); r == 0 {
					return token{}, fmt.Errorf("unterminated escape")
				}
			}
		}
		lit := l.src[start+1 : l.pos-1]
		return l.emit(tokString, start, l.pos, lit), nil
	}

	if unicode.IsDigit(ch) {
		for {
			r, _ := l.peekRune()
			if !unicode.IsDigit(r) {
				break
			}
			l.nextRune()
		}
		if r, _ := l.peekRune(); r == '.' {
			l.nextRune()
			for {
				r, _ := l.peekRune()
				if !unicode.IsDigit(r) {
					break
				}
				l.nextRune()
			}
		}
		return l.emit(tokNumber, start, l.pos, l.src[start:l.pos]), nil
	}

	if isIdentStart(ch) {
		for {
			r, _ := l.peekRune()
			if !isIdentPart(r) {
				break
			}
			l.nextRune()
		}
		lit := l.src[start:l.pos]
		switch lit {
		case "true", "false":
			return l.emit(tokBool, start, l.pos, lit), nil
		case "null":
			return l.emit(tokNull, start, l.pos, lit), nil
		default:
			return l.emit(tokIdentifier, start, l.pos, lit), nil
		}
	}

	// Operators
	op := string(ch)
	switch ch {
	case '+', '-', '*', '/', '%', '!', '<', '>', '=':
		if next, _ := l.peekRune(); next == '=' {
			l.nextRune()
			op = string([]rune{ch, next})
		}
		return l.emit(tokOp, start, l.pos, op), nil
	case '&':
		if next, _ := l.peekRune(); next == '&' {
			l.nextRune()
			return l.emit(tokOp, start, l.pos, "&&"), nil
		}
	case '|':
		if next, _ := l.peekRune(); next == '|' {
			l.nextRune()
			return l.emit(tokOp, start, l.pos, "||"), nil
		}
	}

	return token{}, fmt.Errorf("unexpected character: %q", ch)
}

func isIdentStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentPart(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}
