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

func TestLoadProgramsCSV(t *testing.T) {
	submittersPath := filepath.Join("../testdata/stats/submitters.csv")
	programsPath := filepath.Join("../testdata/stats/programs.csv")
	submitters, err := LoadSubmittersCSV(submittersPath)
	if err != nil {
		t.Fatalf("LoadSubmitters failed: %v", err)
	}
	index := loadTestIndex(t)
	programs, err := LoadProgramsCSV(programsPath, submitters, index)
	if err != nil {
		t.Fatalf("LoadProgramsCSV failed: %v", err)
	}
	if len(programs) != 10 {
		t.Errorf("expected 10 programs, got %d", len(programs))
	}
	// Check a few known values
	p := programs[0]
	if p.Id.String() != "A000002" || p.Length != 10 || p.Usages != 20 || p.IncEval != true || p.LogEval != false {
		t.Errorf("unexpected program[0]: %+v", p)
	}
	if p.Submitter == nil || p.Submitter.Name != "" {
		t.Errorf("unexpected submitter for program[0]: %+v", p.Submitter)
	}
	p = programs[2]
	if p.Id.String() != "A000006" || p.Length != 2 || p.Usages != 1 || p.IncEval != false || p.LogEval != true {
		t.Errorf("unexpected program[2]: %+v", p)
	}
	if p.Submitter == nil || p.Submitter.Name != "Nova_Sky" {
		t.Errorf("unexpected submitter for program[2]: %+v", p.Submitter)
	}
	p = programs[9]
	if p.Id.String() != "A000016" || p.Length != 15 || p.Usages != 4 {
		t.Errorf("unexpected program[9]: %+v", p)
	}
	if p.Submitter == nil || p.Submitter.Name != "@Pixel$Hero" {
		t.Errorf("unexpected submitter for program[9]: %+v", p.Submitter)
	}
}
