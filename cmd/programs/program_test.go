package main

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
	if prog.Submitter != wantSubmitter {
		t.Errorf("expected Submitter %q, got %q", wantSubmitter, prog.Submitter)
	}
}

// loadProgramFromTestFile reads a .asm test file and returns the parsed Program.
func loadProgramFromTestFile(filename string) (Program, error) {
	path := filepath.Join("../../testdata/programs", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return Program{}, err
	}
	code := string(data)
	return NewProgramFromText(code), nil
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
