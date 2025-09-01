package main

import (
	"path/filepath"
	"testing"
)

func TestTermsList(t *testing.T) {
	seq := Sequence{
		Id:    "A000001",
		Name:  "Number of groups of order n.",
		Terms: ",0,1,1,1,2,1,2,1,5,2,2,1,5,1,2,1,14,1,5,1,5,2,2,1,15,2,2,5,4,1,4,1,51,1,2,1,14,1,2,2,14,1,6,1,4,2,2,1,52,2,5,1,5,1,15,2,13,2,2,1,13,1,2,4,267,1,4,1,5,1,4,1,50,1,2,3,4,1,6,1,52,15,2,1,15,1,2,1,12,1,10,1,4,2,",
	}
	got := seq.TermsList()
	want := []string{"0", "1", "1", "1", "2", "1", "2", "1", "5", "2", "2", "1", "5", "1", "2", "1", "14", "1", "5", "1", "5", "2", "2", "1", "15", "2", "2", "5", "4", "1", "4", "1", "51", "1", "2", "1", "14", "1", "2", "2", "14", "1", "6", "1", "4", "2", "2", "1", "52", "2", "5", "1", "5", "1", "15", "2", "13", "2", "2", "1", "13", "1", "2", "4", "267", "1", "4", "1", "5", "1", "4", "1", "50", "1", "2", "3", "4", "1", "6", "1", "52", "15", "2", "1", "15", "1", "2", "1", "12", "1", "10", "1", "4", "2"}
	if len(got) != len(want) {
		t.Fatalf("TermsList: got %d terms, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("TermsList: term %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

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
