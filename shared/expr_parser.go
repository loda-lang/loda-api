package shared

import (
	"strings"
)

// ParseExpr parses a formula expression string into an AST (Expr).
func ParseExpr(expr string) Expr {
	tokenizer := NewTokenizer(strings.TrimSpace(expr))
	return parseAssignment(tokenizer)
}

// assignment = compare ( '=' assignment )?
func parseAssignment(t *Tokenizer) Expr {
	lhs := parseCompare(t)
	if t.Peek().Type == TokenOperator && t.Peek().Value == "=" {
		t.Next()
		rhs := parseAssignment(t)
		return AssignExpr{LHS: lhs, RHS: rhs}
	}
	return lhs
}

// compare = add ( ('=='|'!='|'<'|'<='|'>'|'>=') add )*
func parseCompare(t *Tokenizer) Expr {
	lhs := parseAdd(t)
	for t.Peek().Type == TokenOperator && (t.Peek().Value == "==" || t.Peek().Value == "!=" || t.Peek().Value == "<" || t.Peek().Value == "<=" || t.Peek().Value == ">" || t.Peek().Value == ">=") {
		op := t.Next().Value
		rhs := parseAdd(t)
		lhs = CompareExpr{Op: op, Left: lhs, Right: rhs}
	}
	return lhs
}

// add = mul ( ('+'|'-') mul )*
func parseAdd(t *Tokenizer) Expr {
	lhs := parseMul(t)
	for t.Peek().Type == TokenOperator && (t.Peek().Value == "+" || t.Peek().Value == "-") {
		op := t.Next().Value
		rhs := parseMul(t)
		lhs = BinaryExpr{Op: op, Left: lhs, Right: rhs}
	}
	return lhs
}

// mul = pow ( ('*'|'/'|'%') pow )*
func parseMul(t *Tokenizer) Expr {
	lhs := parsePow(t)
	for t.Peek().Type == TokenOperator && (t.Peek().Value == "*" || t.Peek().Value == "/" || t.Peek().Value == "%") {
		op := t.Next().Value
		rhs := parsePow(t)
		lhs = BinaryExpr{Op: op, Left: lhs, Right: rhs}
	}
	return lhs
}

// pow = unary ( '^' pow )?
func parsePow(t *Tokenizer) Expr {
	lhs := parseUnary(t)
	if t.Peek().Type == TokenOperator && t.Peek().Value == "^" {
		op := t.Next().Value
		rhs := parsePow(t)
		return BinaryExpr{Op: op, Left: lhs, Right: rhs}
	}
	return lhs
}

// unary = ('-'|'+') unary | primary
func parseUnary(t *Tokenizer) Expr {
	if t.Peek().Type == TokenOperator && (t.Peek().Value == "-" || t.Peek().Value == "+") {
		op := t.Next().Value
		expr := parseUnary(t)
		return UnaryExpr{Op: op, Expr: expr}
	}
	return parsePrimary(t)
}

// primary = number | ident ( '(' args ')' )? | '(' expr ')'
func parsePrimary(t *Tokenizer) Expr {
	tok := t.Peek()
	switch tok.Type {
	case TokenNumber:
		t.Next()
		return ConstExpr{Value: tok.Value}
	case TokenIdent:
		name := t.Next().Value
		// Function call or indexed variable
		if t.Peek().Type == TokenParen && t.Peek().Value == "(" {
			t.Next() // consume '('
			var args []Expr
			if t.Peek().Type != TokenParen || t.Peek().Value != ")" {
				for {
					args = append(args, parseAssignment(t))
					if t.Peek().Type == TokenComma {
						t.Next()
					} else {
						break
					}
				}
			}
			t.Expect(TokenParen) // consume ')'
			return FuncCallExpr{FuncName: name, Args: args}
		}
		return VarExpr{Name: name}
	case TokenParen:
		if tok.Value == "(" {
			t.Next()
			expr := parseAssignment(t)
			t.Expect(TokenParen) // consume ')'
			return expr
		}
	}
	// fallback: treat as constant
	t.Next()
	return ConstExpr{Value: tok.Value}
}
