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
