package shared

import (
	"fmt"
	"regexp"
	"strings"
)

// Expr is the interface for all formula AST nodes.
type Expr interface{}

// Various AST node types:
type (
	// Constant value, e.g. 2, -1
	ConstExpr struct {
		Value string // keep as string for now (can be int, float, etc.)
	}
	// Variable, e.g. n, a, b
	VarExpr struct {
		Name string
	}
	// Indexed variable, e.g. a(n-1), b(n+2)
	IndexedVarExpr struct {
		Name  string
		Index Expr
	}
	// Function call, e.g. binomial(x, y), floor(x)
	FuncCallExpr struct {
		FuncName string
		Args     []Expr
	}
	// Binary operation, e.g. x + y, x * y
	BinaryExpr struct {
		Op    string // '+', '-', '*', '/', '%', '^', etc.
		Left  Expr
		Right Expr
	}
	// Unary operation, e.g. -x
	UnaryExpr struct {
		Op   string // '-', 'abs', etc.
		Expr Expr
	}
	// Assignment, e.g. a(n) = ...
	AssignExpr struct {
		LHS Expr // usually IndexedVarExpr
		RHS Expr
	}
	// Comparison, e.g. x == y, x <= y
	CompareExpr struct {
		Op    string // '==', '!=', '<', '<=', '>', '>='
		Left  Expr
		Right Expr
	}
	// Conditional, e.g. if cond then x else y (rare in these formulas)
	IfExpr struct {
		Cond Expr
		Then Expr
		Else Expr
	}
)

// ParseExpr parses a formula expression string into an AST (Expr).
// This is a stub; full parsing logic should be implemented as needed.
func ParseExpr(expr string) Expr {
	expr = strings.TrimSpace(expr)
	// For now, just return as ConstExpr or VarExpr if simple, else as raw string ConstExpr
	if expr == "n" || regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(expr) {
		return VarExpr{Name: expr}
	}
	if regexp.MustCompile(`^-?\d+$`).MatchString(expr) {
		return ConstExpr{Value: expr}
	}
	// TODO: Implement full parser for arithmetic, function calls, etc.
	return ConstExpr{Value: expr}
}

// ExprToString converts an Expr AST node back to a string representation.
func ExprToString(e Expr) string {
	switch v := e.(type) {
	case ConstExpr:
		return v.Value
	case VarExpr:
		return v.Name
	case IndexedVarExpr:
		return fmt.Sprintf("%s(%s)", v.Name, ExprToString(v.Index))
	case FuncCallExpr:
		var args []string
		for _, arg := range v.Args {
			args = append(args, ExprToString(arg))
		}
		return fmt.Sprintf("%s(%s)", v.FuncName, strings.Join(args, ","))
	case BinaryExpr:
		// Add parentheses for clarity
		return fmt.Sprintf("(%s%s%s)", ExprToString(v.Left), v.Op, ExprToString(v.Right))
	case UnaryExpr:
		return fmt.Sprintf("%s%s", v.Op, ExprToString(v.Expr))
	case AssignExpr:
		return fmt.Sprintf("%s = %s", ExprToString(v.LHS), ExprToString(v.RHS))
	case CompareExpr:
		return fmt.Sprintf("(%s%s%s)", ExprToString(v.Left), v.Op, ExprToString(v.Right))
	case IfExpr:
		return fmt.Sprintf("if %s then %s else %s", ExprToString(v.Cond), ExprToString(v.Then), ExprToString(v.Else))
	default:
		// fallback for unknown or nil
		return "?"
	}
}
