package main

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/loda-lang/loda-api/shared"
	"github.com/loda-lang/loda-api/util"
)

func loadTestIndex(t *testing.T) *Index {
	idx := NewIndex()
	testdataDir := filepath.Join("..", "..", "testdata")
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
			gotKeywords := shared.DecodeKeywords(seq.Keywords)
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

func TestFindById(t *testing.T) {
	idx := loadTestIndex(t)

	// Test known sequence
	id, _ := util.NewUIDFromString("A001")
	seq := idx.FindById(id)
	if seq == nil {
		t.Errorf("FindById: did not find A000001")
	} else if seq.Id.String() != "A000001" {
		t.Errorf("FindById: got %q, want %q", seq.Id, "A000001")
	}

	// Test non-existent ID
	id, _ = util.NewUIDFromString("A999999")
	seq = idx.FindById(id)
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
		s := seq.Id.String()
		id2, _ := util.NewUIDFromString(string(s[0]) + "0" + s[1:])
		got2 := idx.FindById(id2)
		if got2 == nil || got2.Id != seq.Id {
			t.Errorf("FindById: failed for %q", seq.Id)
		}
	}
}

func TestIndexSearch(t *testing.T) {
	idx := loadTestIndex(t)

	// Search by query string (name substring)
	results := idx.Search("Kolakoski", nil, nil, 0, 0)
	if len(results) != 1 {
		t.Errorf("Search Kolakoski: got %d results, want 1", len(results))
	} else if !strings.Contains(results[0].Name, "Kolakoski") {
		t.Errorf("Search Kolakoski: wrong sequence name %q", results[0].Name)
	}

	// Search by required keyword
	results = idx.Search("", []string{"core"}, nil, 0, 0)
	for _, seq := range results {
		gotKeywords := shared.DecodeKeywords(seq.Keywords)
		found := false
		for _, kw := range gotKeywords {
			if kw == "core" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Search core: sequence %q missing required keyword", seq.Id)
		}
	}

	// Search by forbidden keyword
	results = idx.Search("", nil, []string{"hard"}, 0, 0)
	for _, seq := range results {
		gotKeywords := shared.DecodeKeywords(seq.Keywords)
		for _, kw := range gotKeywords {
			if kw == "hard" {
				t.Errorf("Search forbidden hard: sequence %q contains forbidden keyword", seq.Id)
			}
		}
	}

	// Search with query tokens (all must match)
	results = idx.Search("groups order", nil, nil, 0, 0)
	if len(results) != 1 || !strings.Contains(results[0].Name, "groups") || !strings.Contains(results[0].Name, "order") {
		t.Errorf("Search groups order: got %d results, want 1 with correct name", len(results))
	}

	// Pagination: skip and limit
	allResults := idx.Search("", nil, nil, 0, 0)
	if len(allResults) != 10 {
		t.Fatalf("All results: got %d results, want 10", len(allResults))
	}
	paged := idx.Search("", nil, nil, 2, 1)
	if len(paged) != 2 {
		t.Errorf("Pagination: got %d results, want 2", len(paged))
	}
	if paged[0].Id != allResults[1].Id || paged[1].Id != allResults[2].Id {
		t.Errorf("Pagination: unexpected sequence IDs")
	}
}
