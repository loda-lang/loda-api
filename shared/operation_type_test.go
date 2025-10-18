package shared

import (
	"path/filepath"
	"slices"
	"sort"
	"testing"
)

// Initialize global operation type index for tests
func init() {
	path := filepath.Join("../testdata/stats/operation_types.csv")
	opTypes, err := LoadOperationTypesCSV(path)
	if err != nil {
		panic("Failed to load operation types for tests: " + err.Error())
	}
	_, err = NewOperationTypeIndex(opTypes)
	if err != nil {
		panic("Failed to create operation type index for tests: " + err.Error())
	}
}

func TestEncodeDecodeOperationTypes(t *testing.T) {
	tests := []struct {
		name string
		ops  []string
	}{
		{"single", []string{"mov"}},
		{"multiple", []string{"add", "sub", "mul"}},
		{"all", []string{"mov", "add", "sub", "trn", "mul", "div", "dif", "dir", "mod", "pow",
			"gcd", "lex", "bin", "fac", "log", "nrt", "dgs", "dgr", "equ", "neq",
			"leq", "geq", "min", "max", "ban", "bor", "bxo", "lpb", "lpe", "clr",
			"fil", "rol", "ror", "seq"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := EncodeOperationTypes(tt.ops)
			if err != nil {
				t.Fatalf("EncodeOperationTypes failed: %v", err)
			}
			decoded := DecodeOperationTypes(encoded)

			// Sort both for comparison
			want := make([]string, len(tt.ops))
			copy(want, tt.ops)
			sort.Strings(want)
			sort.Strings(decoded)

			if !slices.Equal(decoded, want) {
				t.Errorf("got %v, want %v", decoded, want)
			}
		})
	}
}

func TestEncodeOperationTypesUnknown(t *testing.T) {
	_, err := EncodeOperationTypes([]string{"unknown"})
	if err == nil {
		t.Error("expected error for unknown operation type")
	}
}

func TestHasOperationType(t *testing.T) {
	bits, _ := EncodeOperationTypes([]string{"mov", "add", "mul"})

	tests := []struct {
		op   string
		want bool
	}{
		{"mov", true},
		{"add", true},
		{"mul", true},
		{"sub", false},
		{"div", false},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			got := HasOperationType(bits, tt.op)
			if got != tt.want {
				t.Errorf("HasOperationType(%q) = %v, want %v", tt.op, got, tt.want)
			}
		})
	}
}

func TestHasAllOperationTypes(t *testing.T) {
	bits1, _ := EncodeOperationTypes([]string{"mov", "add", "sub", "mul"})
	bits2, _ := EncodeOperationTypes([]string{"mov", "add"})
	bits3, _ := EncodeOperationTypes([]string{"div", "mod"})

	if !HasAllOperationTypes(bits1, bits2) {
		t.Error("expected bits1 to contain all of bits2")
	}
	if HasAllOperationTypes(bits1, bits3) {
		t.Error("expected bits1 to not contain all of bits3")
	}
}

func TestHasNoOperationTypes(t *testing.T) {
	bits1, _ := EncodeOperationTypes([]string{"mov", "add"})
	bits2, _ := EncodeOperationTypes([]string{"div", "mod"})
	bits3, _ := EncodeOperationTypes([]string{"add", "mul"})

	if !HasNoOperationTypes(bits1, bits2) {
		t.Error("expected bits1 to have none of bits2")
	}
	if HasNoOperationTypes(bits1, bits3) {
		t.Error("expected bits1 to have some of bits3")
	}
}

func TestIsOperationType(t *testing.T) {
	tests := []struct {
		op   string
		want bool
	}{
		{"mov", true},
		{"add", true},
		{"seq", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			got := IsOperationType(tt.op)
			if got != tt.want {
				t.Errorf("IsOperationType(%q) = %v, want %v", tt.op, got, tt.want)
			}
		})
	}
}
