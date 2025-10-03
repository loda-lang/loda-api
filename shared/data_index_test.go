package shared

import (
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

func loadTestIndex(t *testing.T) *DataIndex {
	testdataDir := filepath.Join("..", "testdata")
	idx := NewDataIndex(testdataDir)
	err := idx.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	return idx
}

func TestIndexLoad(t *testing.T) {
	idx := loadTestIndex(t)

	if len(idx.Sequences) != 10 {
		t.Fatalf("Expected 10 sequences, got %d", len(idx.Sequences))
	}

	// Check a few known sequences
	want := map[string]struct {
		name     string
		terms    string
		keywords []string
	}{
		"A000001": {
			name:     "Number of groups of order n.",
			terms:    ",0,1,1,1,2,1,2,1,5,2,2,1,5,1,2,1,14,1,5,1,5,2,2,1,15,2,2,5,4,1,4,1,51,1,2,1,14,1,2,2,14,1,6,1,4,2,2,1,52,2,5,1,5,1,15,2,13,2,2,1,13,1,2,4,267,1,4,1,5,1,4,1,50,1,2,3,4,1,6,1,52,15,2,1,15,1,2,1,12,1,10,1,4,2,",
			keywords: []string{"nonn", "core", "nice", "hard", "conjecture", "formula"},
		},
		"A000002": {
			name:     "Kolakoski sequence: a(n) is length of n-th run; a(1) = 1; sequence consists just of 1's and 2's.",
			terms:    ",1,2,2,1,1,2,1,2,2,1,2,2,1,1,2,1,1,2,2,1,2,1,1,2,1,2,2,1,1,2,1,1,2,1,2,2,1,2,2,1,1,2,1,2,2,1,2,1,1,2,1,1,2,2,1,2,2,1,1,2,1,2,2,1,2,2,1,1,2,1,1,2,1,2,2,1,2,1,1,2,2,1,2,2,1,1,2,1,2,2,1,2,2,1,1,2,1,1,2,2,1,2,1,1,2,1,2,2,",
			keywords: []string{"nonn", "core", "easy", "loda", "loda-inceval", "loda-loop", "loda-formula", "nice", "conjecture", "formula", "pari"},
		},
	}
	for _, seq := range idx.Sequences {
		if w, ok := want[seq.Id.String()]; ok {
			if seq.Name != w.name {
				t.Errorf("Sequence %s: got name %q, want %q", seq.Id, seq.Name, w.name)
			}
			if seq.Terms != w.terms {
				t.Errorf("Sequence %s: got terms %q, want %q", seq.Id, seq.Terms, w.terms)
			}
			gotKeywords := DecodeKeywords(seq.Keywords)
			sort.Strings(gotKeywords)
			sort.Strings(w.keywords)
			if !slices.Equal(gotKeywords, w.keywords) {
				t.Errorf("Sequence %s: got keywords %v, want %v", seq.Id, gotKeywords, w.keywords)
			}
			delete(want, seq.Id.String())
		}
	}
	for id := range want {
		t.Errorf("Sequence %s not found in loaded index", id)
	}

	// Check actual used program IDs and usage counts loaded from testdata/stats/call_graph.csv
	wantUsedIDs := map[string]string{
		"A000005": "A001234 A003987 A000001",
		"A000001": "A005467",
		"A000003": "A009876 A007632",
	}
	for _, p := range idx.Programs {
		id := p.Id.String()
		if want, ok := wantUsedIDs[id]; ok {
			gotIDs := strings.Fields(p.Usages)
			wantIDs := strings.Fields(want)
			sort.Strings(gotIDs)
			sort.Strings(wantIDs)
			if !slices.Equal(gotIDs, wantIDs) {
				t.Errorf("Program %s: got used IDs %v, want %v", id, gotIDs, wantIDs)
			}
			// Calculate expected usage count from wantUsedIDs
			wantCount := len(wantIDs)
			gotCount := idx.NumUsages[id]
			if gotCount != wantCount {
				t.Errorf("Usages[%s]: got %d, want %d", id, gotCount, wantCount)
			}
		}
	}
}

func TestLoadProgramsCSV(t *testing.T) {
	submittersPath := filepath.Join("../testdata/stats/submitters.csv")
	programsPath := filepath.Join("../testdata/stats/programs.csv")
	submitters, err := LoadSubmittersCSV(submittersPath)
	if err != nil {
		t.Fatalf("LoadSubmitters failed: %v", err)
	}
	programs, err := LoadProgramsCSV(programsPath, submitters)
	if err != nil {
		t.Fatalf("LoadProgramsCSV failed: %v", err)
	}
	if len(programs) != 13 {
		t.Errorf("expected 13 programs, got %d", len(programs))
	}
	// Check a few known values (based on the new CSV and submitter mapping)
	p := programs[0]
	if p.Id.String() != "A000002" || p.Length != 10 {
		t.Errorf("unexpected program[0]: %+v", p)
	}
	if !HasKeyword(p.Keywords, "loda") || !HasKeyword(p.Keywords, "loda-inceval") || !HasKeyword(p.Keywords, "loda-loop") || !HasKeyword(p.Keywords, "loda-formula") || HasKeyword(p.Keywords, "loda-logeval") {
		t.Errorf("unexpected keywords for program[0]: %v", DecodeKeywords(p.Keywords))
	}
	if p.Submitter == nil || p.Submitter.Name != "" {
		t.Errorf("unexpected submitter for program[0]: %+v", p.Submitter)
	}

	p = programs[2]
	if p.Id.String() != "A000005" || p.Length != 22 {
		t.Errorf("unexpected program[2]: %+v", p)
	}
	if !HasKeyword(p.Keywords, "loda") || !HasKeyword(p.Keywords, "loda-loop") || HasKeyword(p.Keywords, "loda-inceval") || HasKeyword(p.Keywords, "loda-logeval") || HasKeyword(p.Keywords, "loda-formula") {
		t.Errorf("unexpected keywords for program[2]: %v", DecodeKeywords(p.Keywords))
	}
	if p.Submitter == nil || p.Submitter.Name != "Luna Moon" {
		t.Errorf("unexpected submitter for program[2]: %+v", p.Submitter)
	}

	p = programs[9]
	if p.Id.String() != "A000012" || p.Length != 1 {
		t.Errorf("unexpected program[9]: %+v", p)
	}
	if !HasKeyword(p.Keywords, "loda") || !HasKeyword(p.Keywords, "loda-formula") || HasKeyword(p.Keywords, "loda-inceval") || HasKeyword(p.Keywords, "loda-logeval") || HasKeyword(p.Keywords, "loda-loop") {
		t.Errorf("unexpected keywords for program[9]: %v", DecodeKeywords(p.Keywords))
	}
	if p.Submitter == nil || p.Submitter.Name != "Star*Gazer" {
		t.Errorf("unexpected submitter for program[9]: %+v", p.Submitter)
	}

	p = programs[12]
	if p.Id.String() != "A000016" || p.Length != 15 {
		t.Errorf("unexpected program[12]: %+v", p)
	}
	if !HasKeyword(p.Keywords, "loda") || !HasKeyword(p.Keywords, "loda-loop") || HasKeyword(p.Keywords, "loda-inceval") || HasKeyword(p.Keywords, "loda-logeval") || HasKeyword(p.Keywords, "loda-formula") {
		t.Errorf("unexpected keywords for program[12]: %v", DecodeKeywords(p.Keywords))
	}
	if p.Submitter == nil || p.Submitter.Name != "@Pixel$Hero" {
		t.Errorf("unexpected submitter for program[12]: %+v", p.Submitter)
	}
}

func TestLoadSubmittersCSV(t *testing.T) {
	path := filepath.Join("../testdata/stats/submitters.csv")
	submitters, err := LoadSubmittersCSV(path)
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
