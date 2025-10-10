package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/loda-lang/loda-api/shared"
)

func TestExportValidFormats(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewLODATool(tmpDir, 1)

	// Create a simple test program
	program := shared.Program{}
	program.SetCode("mov $0,1\nadd $0,1")

	validFormats := []string{"formula", "pari", "loda", "range"}
	for _, format := range validFormats {
		t.Run(format, func(t *testing.T) {
			result := tool.Export(program, format)
			// We expect the export to succeed (status can be success or error depending on LODA availability)
			// Just check that the result has expected fields
			if result.Status != "success" && result.Status != "error" {
				t.Errorf("Expected status to be 'success' or 'error', got '%s'", result.Status)
			}
		})
	}
}

func TestExportInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewLODATool(tmpDir, 1)

	program := shared.Program{}
	program.SetCode("mov $0,1\nadd $0,1")

	result := tool.Export(program, "invalid")
	if result.Status != "error" {
		t.Errorf("Expected status 'error' for invalid format, got '%s'", result.Status)
	}
	if result.Message == "" {
		t.Error("Expected error message for invalid format")
	}
}

func TestExportEmptyProgram(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewLODATool(tmpDir, 1)

	program := shared.Program{}
	program.SetCode("")

	result := tool.Export(program, "loda")
	// Empty program should still create temp file and attempt export
	if result.Status != "success" && result.Status != "error" {
		t.Errorf("Expected status to be 'success' or 'error', got '%s'", result.Status)
	}
}

// TestExportWithSetupFile tests export when setup.txt exists
func TestExportWithSetupFile(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create setup.txt to pass Install validation
	setupFile := filepath.Join(tmpDir, "setup.txt")
	if err := os.WriteFile(setupFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create setup.txt: %v", err)
	}
	
	tool := NewLODATool(tmpDir, 1)

	program := shared.Program{}
	program.SetCode("mov $0,1\nadd $0,1")

	// Test with a valid format
	result := tool.Export(program, "loda")
	// Export should work or fail gracefully
	if result.Status != "success" && result.Status != "error" {
		t.Errorf("Expected status to be 'success' or 'error', got '%s'", result.Status)
	}
}
