package shared

import (
	"fmt"
	"regexp"
	"strings"
)

// Formula represents a parsed formula line, possibly with multiple sequences.
type Formula struct {
	Parts []FormulaPart // order of all parts as parsed
}

// FormulaPart records a single assignment in the formula: "LHS = RHS".
type FormulaPart struct {
	LHS Expr // usually a function call like a(n), a(0), etc.
	RHS Expr
}

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

// ParseFormulaLine parses a single line from formula.txt into a Formula struct.
func ParseFormulaLine(line string) (*Formula, error) {
	// Remove comments and trim
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}
	// Split by commas, but only at top-level (not inside parentheses)
	parts := splitTopLevel(line, ',')
	var partsOrder []FormulaPart
	assignRe := regexp.MustCompile(`^(.+?)\s*=\s*(.+)$`)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		m := assignRe.FindStringSubmatch(part)
		if m == nil {
			return nil, fmt.Errorf("unrecognized formula part: %q", part)
		}
		lhs := ParseExpr(m[1])
		rhs := ParseExpr(m[2])
		partsOrder = append(partsOrder, FormulaPart{LHS: lhs, RHS: rhs})
	}
	return &Formula{Parts: partsOrder}, nil
}

// splitTopLevel splits s by sep, ignoring sep inside parentheses.
func splitTopLevel(s string, sep rune) []string {
	var res []string
	var buf strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '(':
			depth++
			buf.WriteRune(r)
		case ')':
			if depth > 0 {
				depth--
			}
			buf.WriteRune(r)
		case sep:
			if depth == 0 {
				res = append(res, buf.String())
				buf.Reset()
			} else {
				buf.WriteRune(r)
			}
		default:
			buf.WriteRune(r)
		}
	}
	if buf.Len() > 0 {
		res = append(res, buf.String())
	}
	return res
}

// String returns a string representation of the Formula, reconstructing the original formula line.
func (f *Formula) String() string {
	if f == nil {
		return ""
	}
	var out []string
	for _, p := range f.Parts {
		out = append(out, fmt.Sprintf("%s = %s", ExprToString(p.LHS), ExprToString(p.RHS)))
	}
	return strings.Join(out, ", ")
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
