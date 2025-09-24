package shared

import (
	"fmt"
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
	// Function call, e.g. a(n), b(n-2), floor(x), binomial(x, y)
	FuncCallExpr struct {
		FuncName string
		Args     []Expr
	}
	// Binary operation, e.g. x+y, x*y
	BinaryExpr struct {
		Op    string // '+', '-', '*', '/', '%', '^', etc.
		Left  Expr
		Right Expr
	}
	// Unary operation, e.g. -x
	UnaryExpr struct {
		Op   string // '-', etc.
		Expr Expr
	}
	// Comparison, e.g. x==y, x<=y
	CompareExpr struct {
		Op    string // '==', '!=', '<', '<=', '>', '>='
		Left  Expr
		Right Expr
	}
)

// ExprToString converts an Expr AST node back to a string representation.
func ExprToString(e Expr) string {
	switch v := e.(type) {
	case ConstExpr:
		return v.Value
	case VarExpr:
		return v.Name
	case FuncCallExpr:
		// If single-letter name and one arg, treat as indexed variable: a(n)
		if len(v.FuncName) == 1 && len(v.Args) == 1 {
			return fmt.Sprintf("%s(%s)", v.FuncName, ExprToString(v.Args[0]))
		}
		var args []string
		for _, arg := range v.Args {
			args = append(args, ExprToString(arg))
		}
		return fmt.Sprintf("%s(%s)", v.FuncName, strings.Join(args, ","))
	case BinaryExpr:
		left := ExprToString(v.Left)
		right := ExprToString(v.Right)
		if needsParensLeft(v.Op, v.Left) {
			left = "(" + left + ")"
		}
		if needsParensRight(v.Op, v.Right) {
			right = "(" + right + ")"
		}
		return fmt.Sprintf("%s%s%s", left, v.Op, right)
	case UnaryExpr:
		expr := ExprToString(v.Expr)
		if needsParens(v.Expr) {
			expr = "(" + expr + ")"
		}
		return fmt.Sprintf("%s%s", v.Op, expr)
	case CompareExpr:
		left := ExprToString(v.Left)
		right := ExprToString(v.Right)
		if needsParensLeft(v.Op, v.Left) {
			left = "(" + left + ")"
		}
		if needsParensRight(v.Op, v.Right) {
			right = "(" + right + ")"
		}
		return fmt.Sprintf("%s%s%s", left, v.Op, right)
	}
	return ""
}

func opPrec(op string) int {
	switch op {
	case "=":
		return 1
	case "==", "!=", "<", "<=", ">", ">=":
		return 2
	case "+", "-":
		return 3
	case "*", "/", "%":
		return 4
	case "^":
		return 5
	default:
		return 0
	}
}

func needsParensLeft(parentOp string, left Expr) bool {
	// Do not parenthesize left UnaryExpr for +, -, *, /
	if _, ok := left.(UnaryExpr); ok && (parentOp == "+" || parentOp == "-" || parentOp == "*" || parentOp == "/") {
		return false
	}
	be, ok := left.(BinaryExpr)
	if !ok {
		return needsParens(left)
	}
	return opPrec(be.Op) < opPrec(parentOp)
}

func needsParensRight(parentOp string, right Expr) bool {
	be, ok := right.(BinaryExpr)
	if !ok {
		return needsParens(right)
	}
	// For right-associative operators like '^', use <=
	if parentOp == "^" {
		return opPrec(be.Op) <= opPrec(parentOp)
	}
	return opPrec(be.Op) < opPrec(parentOp)
}

// needsParens returns true if the expr should be parenthesized when used as a subexpression
func needsParens(e Expr) bool {
	switch e.(type) {
	case BinaryExpr, CompareExpr, UnaryExpr:
		return true
	default:
		return false
	}
}
