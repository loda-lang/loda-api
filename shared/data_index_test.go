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
	err := idx.Load(true)
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
			keywords: []string{"nonn", "core", "easy", "loda", "loda-inceval", "nice", "conjecture", "formula"},
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
}

func TestLoadCommentsFile(t *testing.T) {
	commentsPath := filepath.Join("..", "testdata", "seqs", "oeis", "comments")
	comments, err := LoadOeisTextFile(commentsPath)
	if err != nil {
		t.Fatalf("LoadCommentsFile failed: %v", err)
	}
	// Check that some known UIDs exist and have expected content
	a1 := comments["A000001"]
	if a1 == "" {
		t.Errorf("A000001 comments missing")
	} else if want := "Also, number of nonisomorphic primitives of the combinatorial species Lin[n-1]. - _Nicolae Boicu_, Apr 29 2011\nAlso, number of nonisomorphic subgroups of order n in symmetric group S_n. - _Lekraj Beedassy_, Dec 16 2004\nI conjecture that a(i) * a(j) <= a(i*j) for all nonnegative integers i and j. - _Jorge R. F. F. Lopes_, Apr 21 2024"; !strings.HasPrefix(a1, want[:60]) {
		t.Errorf("A000001 comments do not start as expected: got %q", a1)
	}
	a2 := comments["A000002"]
	if a2 == "" {
		t.Errorf("A000002 comments missing")
	} else if !strings.Contains(a2, "Kolakoski sequence") {
		t.Errorf("A000002 comments missing expected content: got %q", a2)
	}
	// Check that multiple comments are concatenated with newlines
	if strings.Count(a1, "\n") < 2 {
		t.Errorf("A000001 should have multiple concatenated comments, got: %q", a1)
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
