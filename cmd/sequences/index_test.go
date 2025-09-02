package main

import (
	"path/filepath"
	"testing"
)

func TestIndexLoad(t *testing.T) {
	idx := NewIndex()
	testdataDir := filepath.Join("..", "..", "testdata")
	err := idx.Load(testdataDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

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
			keywords: []string{"nonn", "core", "easy", "nice", "changed"},
		},
	}
	for _, seq := range idx.Sequences {
		if w, ok := want[seq.Id]; ok {
			if seq.Name != w.name {
				t.Errorf("Sequence %s: got name %q, want %q", seq.Id, seq.Name, w.name)
			}
			if seq.Terms != w.terms {
				t.Errorf("Sequence %s: got terms %q, want %q", seq.Id, seq.Terms, w.terms)
			}
			if len(seq.Keywords) != len(w.keywords) {
				t.Errorf("Sequence %s: got %d keywords, want %d", seq.Id, len(seq.Keywords), len(w.keywords))
			} else {
				for i := range w.keywords {
					if seq.Keywords[i] != w.keywords[i] {
						t.Errorf("Sequence %s: keyword %d: got %q, want %q", seq.Id, i, seq.Keywords[i], w.keywords[i])
					}
				}
			}
			delete(want, seq.Id)
		}
	}
	for id := range want {
		t.Errorf("Sequence %s not found in loaded index", id)
	}
}

func TestFindById(t *testing.T) {
	idx := NewIndex()
	testdataDir := filepath.Join("..", "..", "testdata")
	err := idx.Load(testdataDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Test known sequence
	seq := idx.FindById("A001")
	if seq == nil {
		t.Errorf("FindById: did not find A000001")
	} else if seq.Id != "A000001" {
		t.Errorf("FindById: got %q, want %q", seq.Id, "A000001")
	}

	// Test non-existent ID
	seq = idx.FindById("A999999")
	if seq != nil {
		t.Errorf("FindById: expected nil for non-existent ID, got %q", seq.Id)
	}

	// Test first and last
	first := idx.Sequences[0].Id
	last := idx.Sequences[len(idx.Sequences)-1].Id
	if idx.FindById(first) == nil {
		t.Errorf("FindById: did not find first sequence %q", first)
	}
	if idx.FindById(last) == nil {
		t.Errorf("FindById: did not find last sequence %q", last)
	}

	// Test all loaded sequences
	for _, seq := range idx.Sequences {
		got := idx.FindById(seq.Id)
		if got == nil || got.Id != seq.Id {
			t.Errorf("FindById: failed for %q", seq.Id)
		}
		got2 := idx.FindById(string(seq.Id[0]) + "0" + seq.Id[1:])
		if got2 == nil || got2.Id != seq.Id {
			t.Errorf("FindById: failed for %q", seq.Id)
		}
	}
}
