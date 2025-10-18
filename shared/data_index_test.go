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
		authors  []string
	}{
		"A000001": {
			name:     "Number of groups of order n.",
			terms:    ",0,1,1,1,2,1,2,1,5,2,2,1,5,1,2,1,14,1,5,1,5,2,2,1,15,2,2,5,4,1,4,1,51,1,2,1,14,1,2,2,14,1,6,1,4,2,2,1,52,2,5,1,5,1,15,2,13,2,2,1,13,1,2,4,267,1,4,1,5,1,4,1,50,1,2,3,4,1,6,1,52,15,2,1,15,1,2,1,12,1,10,1,4,2,",
			keywords: []string{"nonn", "core", "nice", "hard", "conjecture", "formula"},
			authors:  []string{"N. J. A. Sloane"},
		},
		"A000002": {
			name:     "Kolakoski sequence: a(n) is length of n-th run; a(1) = 1; sequence consists just of 1's and 2's.",
			terms:    ",1,2,2,1,1,2,1,2,2,1,2,2,1,1,2,1,1,2,2,1,2,1,1,2,1,2,2,1,1,2,1,1,2,1,2,2,1,2,2,1,1,2,1,2,2,1,2,1,1,2,1,1,2,2,1,2,2,1,1,2,1,2,2,1,2,2,1,1,2,1,1,2,1,2,2,1,2,1,1,2,2,1,2,2,1,1,2,1,2,2,1,2,2,1,1,2,1,1,2,2,1,2,1,1,2,1,2,2,",
			keywords: []string{"nonn", "core", "easy", "loda", "loda-inceval", "loda-loop", "loda-formula", "nice", "conjecture", "formula", "pari"},
			authors:  []string{"N. J. A. Sloane", "Simon Plouffe"},
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
			// Check authors
			var gotAuthors []string
			for _, a := range seq.Authors {
				gotAuthors = append(gotAuthors, a.Name)
			}
			sort.Strings(gotAuthors)
			sort.Strings(w.authors)
			if !slices.Equal(gotAuthors, w.authors) {
				t.Errorf("Sequence %s: got authors %v, want %v", seq.Id, gotAuthors, w.authors)
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
	if p.OpsMask != 805308526 {
		t.Errorf("unexpected ops_bitmask for program[0]: got %d, want 805308526", p.OpsMask)
	}
	// Verify operation types can be decoded
	opTypes := DecodeOperationTypes(p.OpsMask)
	if len(opTypes) == 0 {
		t.Errorf("expected operation types to be decoded from ops_bitmask, got empty list")
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
	if p.OpsMask != 813695726 {
		t.Errorf("unexpected ops_bitmask for program[2]: got %d, want 813695726", p.OpsMask)
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
	if p.OpsMask != 2 {
		t.Errorf("unexpected ops_bitmask for program[9]: got %d, want 2", p.OpsMask)
	}
	// Verify bit 1 (mov) is set in ops_bitmask
	if !HasOperationType(p.OpsMask, "mov") {
		t.Errorf("expected 'mov' operation type to be present in ops_bitmask")
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
	if p.OpsMask != 822086766 {
		t.Errorf("unexpected ops_bitmask for program[12]: got %d, want 822086766", p.OpsMask)
	}
}

func TestLoadOperationTypesCSV(t *testing.T) {
	path := filepath.Join("../testdata/stats/operation_types.csv")
	opTypes, err := LoadOperationTypesCSV(path)
	if err != nil {
		t.Fatalf("LoadOperationTypesCSV failed: %v", err)
	}
	if len(opTypes) != 34 {
		t.Errorf("expected 34 operation types, got %d", len(opTypes))
	}
	// Check a few known values
	if opTypes[0].Name != "mov" || opTypes[0].RefId != 1 || opTypes[0].Count != 667789 {
		t.Errorf("unexpected operation type[0]: %+v", opTypes[0])
	}
	if opTypes[1].Name != "add" || opTypes[1].RefId != 2 || opTypes[1].Count != 490252 {
		t.Errorf("unexpected operation type[1]: %+v", opTypes[1])
	}
	if opTypes[33].Name != "seq" || opTypes[33].RefId != 34 || opTypes[33].Count != 60327 {
		t.Errorf("unexpected operation type[33]: %+v", opTypes[33])
	}
	// Verify ref_id matches position in OperationTypeList
	for _, op := range opTypes {
		if op.RefId <= 0 || op.RefId >= len(OperationTypeList) || OperationTypeList[op.RefId] != op.Name {
			t.Errorf("operation type %s with ref_id %d does not match OperationTypeList", op.Name, op.RefId)
		}
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

func TestAuthorsLoaded(t *testing.T) {
	idx := loadTestIndex(t)

	// Expected authors and their sequence counts from testdata/seqs/oeis/authors
	want := map[string]int{
		"N. J. A. Sloane":  6, // appears in A000001, A000002, A000003, A000006, A000007, A000008
		"Simon Plouffe":    1, // A000002
		"Clark Kimberling": 1, // A000004
		"Thomas L. York":   2, // A000005, A000006
		"R. K. Guy":        2, // A000007, A000008
		"Philippe Deléham": 1, // A000009
		"R. H. Hardin":     1, // A000010
	}

	got := make(map[string]int)
	for _, a := range idx.Authors {
		got[a.Name] = a.NumSequences
	}

	// Check all expected authors are present with correct counts
	for name, count := range want {
		if gotCount, ok := got[name]; !ok {
			t.Errorf("Author %q missing from index", name)
		} else if gotCount != count {
			t.Errorf("Author %q: got %d sequences, want %d", name, gotCount, count)
		}
	}

	// Check no unexpected authors
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("Unexpected author %q found in index", name)
		}
	}

	// Check author names are cleaned (no underscores, trimmed)
	for name := range got {
		if strings.Contains(name, "_") {
			t.Errorf("Author name %q contains underscore", name)
		}
		if strings.TrimSpace(name) != name {
			t.Errorf("Author name %q is not trimmed", name)
		}
	}
}
