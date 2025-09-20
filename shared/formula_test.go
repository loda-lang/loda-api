package shared

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestFormulaRoundTrip(t *testing.T) {
	f, err := os.Open("../testdata/formula.txt")
	if err != nil {
		t.Fatalf("failed to open formula.txt: %v", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lineno := 0
	for scanner.Scan() {
		lineno++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		formula, err := ParseFormulaLine(line)
		if err != nil {
			t.Errorf("Parse error on line %d: %v\ninput: %q", lineno, err, line)
			continue
		}
		if formula == nil {
			t.Errorf("ParseFormulaLine returned nil for non-empty line %d: %q", lineno, line)
			continue
		}
		out := formula.String()
		// Normalize whitespace for comparison
		norm := func(s string) string {
			return strings.Join(strings.Fields(s), " ")
		}
		if norm(out) != norm(line) {
			t.Errorf("Round-trip mismatch on line %d:\ninput:    %q\nparsed:   %q", lineno, line, out)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
}
