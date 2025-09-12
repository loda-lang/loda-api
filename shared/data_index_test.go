package shared

import (
	"path/filepath"
	"sort"
	"testing"
)

func loadTestIndex(t *testing.T) *DataIndex {
	idx := NewDataIndex()
	testdataDir := filepath.Join("..", "testdata")
	err := idx.Load(testdataDir)
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
			keywords: []string{"nonn", "core", "nice", "hard"},
		},
		"A000002": {
			name:     "Kolakoski sequence: a(n) is length of n-th run; a(1) = 1; sequence consists just of 1's and 2's.",
			terms:    ",1,2,2,1,1,2,1,2,2,1,2,2,1,1,2,1,1,2,2,1,2,1,1,2,1,2,2,1,1,2,1,1,2,1,2,2,1,2,2,1,1,2,1,2,2,1,2,1,1,2,1,1,2,2,1,2,2,1,1,2,1,2,2,1,2,2,1,1,2,1,1,2,1,2,2,1,2,1,1,2,2,1,2,2,1,1,2,1,2,2,1,2,2,1,1,2,1,1,2,2,1,2,1,1,2,1,2,2,",
			keywords: []string{"nonn", "core", "easy", "nice"},
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
			if len(gotKeywords) != len(w.keywords) {
				t.Errorf("Sequence %s: got %d keywords, want %d", seq.Id, len(gotKeywords), len(w.keywords))
			} else {
				for i := range w.keywords {
					if gotKeywords[i] != w.keywords[i] {
						t.Errorf("Sequence %s: keyword %d: got %q, want %q", seq.Id, i, gotKeywords[i], w.keywords[i])
					}
				}
			}
			delete(want, seq.Id.String())
		}
	}
	for id := range want {
		t.Errorf("Sequence %s not found in loaded index", id)
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
	if p.Id.String() != "A000002" || p.Length != 10 || p.Usages != 20 {
		t.Errorf("unexpected program[0]: %+v", p)
	}
	if !HasKeyword(p.Keywords, "loda") || !HasKeyword(p.Keywords, "loda-inceval") || HasKeyword(p.Keywords, "loda-logeval") {
		t.Errorf("unexpected keywords for program[0]: %v", DecodeKeywords(p.Keywords))
	}
	if p.Submitter == nil || p.Submitter.Name != "" {
		t.Errorf("unexpected submitter for program[0]: %+v", p.Submitter)
	}

	p = programs[2]
	if p.Id.String() != "A000006" || p.Length != 2 || p.Usages != 1 {
		t.Errorf("unexpected program[2]: %+v", p)
	}
	if !HasKeyword(p.Keywords, "loda") || HasKeyword(p.Keywords, "loda-inceval") || !HasKeyword(p.Keywords, "loda-logeval") {
		t.Errorf("unexpected keywords for program[2]: %v", DecodeKeywords(p.Keywords))
	}
	if p.Submitter == nil || p.Submitter.Name != "Nova_Sky" {
		t.Errorf("unexpected submitter for program[2]: %+v", p.Submitter)
	}

	p = programs[9]
	if p.Id.String() != "A000016" || p.Length != 15 || p.Usages != 4 {
		t.Errorf("unexpected program[9]: %+v", p)
	}
	if !HasKeyword(p.Keywords, "loda") {
		t.Errorf("unexpected keywords for program[9]: %v", DecodeKeywords(p.Keywords))
	}
	if p.Submitter == nil || p.Submitter.Name != "@Pixel$Hero" {
		t.Errorf("unexpected submitter for program[9]: %+v", p.Submitter)
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
