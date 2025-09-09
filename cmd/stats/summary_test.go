package main

import (
	"path/filepath"
	"testing"
)

func TestLoadSummaryCSV(t *testing.T) {
	path := filepath.Join("../..", "testdata", "stats", "summary.csv")
	summary, err := LoadSummaryCSV(path)
	if err != nil {
		t.Fatalf("failed to load summary: %v", err)
	}
	if summary.NumSequences != 374825 {
		t.Errorf("NumSequences: got %d, want %d", summary.NumSequences, 374825)
	}
	if summary.NumPrograms != 136561 {
		t.Errorf("NumPrograms: got %d, want %d", summary.NumPrograms, 136561)
	}
	if summary.NumFormulas != 63554 {
		t.Errorf("NumFormulas: got %d, want %d", summary.NumFormulas, 63554)
	}
}
