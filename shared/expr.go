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
	// Function call or indexed variable, e.g. binomial(x, y), floor(x), a(n-1), b(n+2)
	FuncCallExpr struct {
		FuncName string
		Args     []Expr // for indexed variable, Args has one element (the index)
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
	// Function call or indexed variable: e.g. binomial(x, y), floor(x), a(n-1), b(n+2)
	funcCallRe := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\((.*)\)$`)
	if m := funcCallRe.FindStringSubmatch(expr); m != nil {
		funcName := m[1]
		argsStr := m[2]
		// Split argsStr by commas, but handle nested parentheses
		var args []Expr
		var cur strings.Builder
		depth := 0
		for i := 0; i < len(argsStr); i++ {
			c := argsStr[i]
			if c == '(' {
				depth++
			} else if c == ')' {
				depth--
			} else if c == ',' && depth == 0 {
				arg := strings.TrimSpace(cur.String())
				if arg != "" {
					args = append(args, ParseExpr(arg))
				}
				cur.Reset()
				continue
			}
			cur.WriteByte(c)
		}
		arg := strings.TrimSpace(cur.String())
		if arg != "" {
			args = append(args, ParseExpr(arg))
		}
		return FuncCallExpr{FuncName: funcName, Args: args}
	}
	// Variable
	if expr == "n" || regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(expr) {
		return VarExpr{Name: expr}
	}
	// Constant integer
	if regexp.MustCompile(`^-?\d+$`).MatchString(expr) {
		return ConstExpr{Value: expr}
	}
	// TODO: Implement full parser for arithmetic, indexed vars, etc.
	return ConstExpr{Value: expr}
}

// ExprToString converts an Expr AST node back to a string representation.
func ExprToString(e Expr) string {
	switch v := e.(type) {
	case ConstExpr:
		return v.Value
	case VarExpr:
		return v.Name
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
