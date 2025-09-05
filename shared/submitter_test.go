package shared

import (
	"path/filepath"
	"testing"
)

func TestLoadSubmitters(t *testing.T) {
	path := filepath.Join("../../testdata/stats/submitters.csv")
	submitters, err := LoadSubmitters(path)
	if err != nil {
		t.Fatalf("LoadSubmitters failed: %v", err)
	}
	if len(submitters) != 11 { // max ref_id is 10
		t.Errorf("expected 11 submitters, got %d", len(submitters))
	}
	// Check a few known values
	if submitters[1] == nil || submitters[1].Name != "" || submitters[1].NumPrograms != 8762 {
		t.Errorf("unexpected submitter[1]: %+v", submitters[1])
	}
	if submitters[2] == nil || submitters[2].Name != "Star*Gazer" || submitters[2].NumPrograms != 432 {
		t.Errorf("unexpected submitter[2]: %+v", submitters[2])
	}
	if submitters[8] == nil || submitters[8].Name != "Quantum^Leap" || submitters[8].NumPrograms != 69322 {
		t.Errorf("unexpected submitter[8]: %+v", submitters[8])
	}
	if submitters[10] == nil || submitters[10].Name != "Velvet Rose" || submitters[10].NumPrograms != 0 {
		t.Errorf("unexpected submitter[10]: %+v", submitters[10])
	}
}
