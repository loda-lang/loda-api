package shared

import (
	"fmt"
	"unicode"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenNumber
	TokenIdent
	TokenOperator
	TokenParen
	TokenComma
)

type Token struct {
	Type  TokenType
	Value string
}

type Tokenizer struct {
	input string
	pos   int
	curr  Token
}

func NewTokenizer(input string) *Tokenizer {
	t := &Tokenizer{input: input, pos: 0}
	t.next()
	return t
}

func (t *Tokenizer) next() {
	t.skipWhitespace()
	if t.pos >= len(t.input) {
		t.curr = Token{Type: TokenEOF}
		return
	}
	ch := t.input[t.pos]
	// Numbers (integer, negative, float)
	if unicode.IsDigit(rune(ch)) || (ch == '-' && t.pos+1 < len(t.input) && unicode.IsDigit(rune(t.input[t.pos+1]))) {
		start := t.pos
		t.pos++ // skip first digit or '-'
		for t.pos < len(t.input) && (unicode.IsDigit(rune(t.input[t.pos])) || t.input[t.pos] == '.') {
			t.pos++
		}
		t.curr = Token{Type: TokenNumber, Value: t.input[start:t.pos]}
		return
	}
	// Identifiers (variables, function names)
	if unicode.IsLetter(rune(ch)) || ch == '_' {
		start := t.pos
		t.pos++
		for t.pos < len(t.input) && (unicode.IsLetter(rune(t.input[t.pos])) || unicode.IsDigit(rune(t.input[t.pos])) || t.input[t.pos] == '_') {
			t.pos++
		}
		t.curr = Token{Type: TokenIdent, Value: t.input[start:t.pos]}
		return
	}
	// Operators and punctuation
	// Multi-char operators: ==, <=, >=, !=
	if t.pos+1 < len(t.input) {
		op2 := t.input[t.pos : t.pos+2]
		switch op2 {
		case "==", "<=", ">=", "!=":
			t.curr = Token{Type: TokenOperator, Value: op2}
			t.pos += 2
			return
		}
	}
	switch ch {
	case '+', '-', '*', '/', '%', '^', '=', '<', '>', '!':
		t.curr = Token{Type: TokenOperator, Value: string(ch)}
		t.pos++
		return
	case '(', ')':
		t.curr = Token{Type: TokenParen, Value: string(ch)}
		t.pos++
		return
	case ',':
		t.curr = Token{Type: TokenComma, Value: ","}
		t.pos++
		return
	}
	t.curr = Token{Type: TokenOperator, Value: string(ch)}
	t.pos++
}

func (t *Tokenizer) skipWhitespace() {
	for t.pos < len(t.input) && unicode.IsSpace(rune(t.input[t.pos])) {
		t.pos++
	}
}

func (t *Tokenizer) Peek() Token {
	return t.curr
}

func (t *Tokenizer) Next() Token {
	tok := t.curr
	t.next()
	return tok
}

func (t *Tokenizer) Expect(tt TokenType) Token {
	tok := t.Next()
	if tok.Type != tt {
		panic(fmt.Sprintf("expected token %v, got %v", tt, tok.Type))
	}
	return tok
}
