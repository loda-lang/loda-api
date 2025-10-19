package shared

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checkProgramMeta checks the ID, name prefix, and submitter of a Program.
func checkProgramMeta(t *testing.T, prog Program, wantID, wantNamePrefix, wantSubmitter string) {
	t.Helper()
	if prog.Id.String() != wantID {
		t.Errorf("expected Id %s, got %q", wantID, prog.Id.String())
	}
	if !strings.HasPrefix(prog.Name, wantNamePrefix) {
		t.Errorf("unexpected Name: %q", prog.Name)
	}
	if len(wantSubmitter) > 0 && prog.Submitter.Name != wantSubmitter {
		t.Errorf("expected Submitter %q, got %q", wantSubmitter, prog.Submitter)
	}
}

// checkOperationTypes checks that OpsMask is initialized to 0 and verifies the extracted operation types.
func checkOperationTypes(t *testing.T, prog Program, expectedOpTypes []string) {
	t.Helper()
	
	// Check OpsMask is initialized to 0
	if prog.OpsMask != 0 {
		t.Errorf("expected OpsMask to be 0, got %d", prog.OpsMask)
	}
	
	// Extract and verify operation types
	opTypes := extractOperationTypes(prog.Operations)
	if len(opTypes) != len(expectedOpTypes) {
		t.Errorf("expected %d operation types, got %d: %v", len(expectedOpTypes), len(opTypes), opTypes)
	}
	for _, expected := range expectedOpTypes {
		found := false
		for _, actual := range opTypes {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected operation type %q not found in %v", expected, opTypes)
		}
	}
}

// loadProgramFromTestFile reads a .asm test file and returns the parsed Program.
func loadProgramFromTestFile(filename string) (Program, error) {
	path := filepath.Join("../testdata/programs", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return Program{}, err
	}
	code := string(data)
	return NewProgramFromCode(code)
}

func TestNewProgramFromText_A000030(t *testing.T) {
	prog, err := loadProgramFromTestFile("A000030.asm")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}
	checkProgramMeta(t, prog, "A000030", "Initial digit of n", "Penguin")
	if len(prog.Operations) == 0 || prog.Operations[0] != "mov $1,$0" {
		t.Errorf("unexpected Operations: %v", prog.Operations)
	}
	checkOperationTypes(t, prog, []string{"mov", "lpb", "div", "sub", "lpe"})
}

func TestNewProgramFromText_A000042(t *testing.T) {
	prog, err := loadProgramFromTestFile("A000042.asm")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}
	checkProgramMeta(t, prog, "A000042", "Unary representation of natural numbers", "Foo Bar")
	if len(prog.Operations) == 0 || prog.Operations[0] != "mov $1,10" {
		t.Errorf("unexpected Operations: %v", prog.Operations)
	}
	checkOperationTypes(t, prog, []string{"mov", "pow", "div"})
}

func TestNewProgramFromText_A000168(t *testing.T) {
	prog, err := loadProgramFromTestFile("A000168.asm")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}
	checkProgramMeta(t, prog, "A000168", "a(n) = 2*3^n", "")
	if len(prog.Operations) == 0 || prog.Operations[0] != "mov $1,$0" {
		t.Errorf("unexpected Operations: %v", prog.Operations)
	}
	checkOperationTypes(t, prog, []string{"mov", "add", "seq", "mul", "div"})
}

func TestProgramMarshalUnmarshalJSON(t *testing.T) {
	prog, err := loadProgramFromTestFile("A000030.asm")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}
	data, err := json.Marshal(prog)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var out Program
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	checkProgramMeta(t, out, "A000030", "Initial digit of n", "Penguin")
	if out.Code != prog.Code {
		t.Errorf("Code mismatch after roundtrip")
	}
	if len(out.Operations) != len(prog.Operations) {
		t.Errorf("Operations length mismatch after roundtrip")
	}
}


