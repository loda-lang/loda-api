package shared

import (
	"strings"
	"testing"

	"github.com/loda-lang/loda-api/util"
)

func TestFindSequenceById(t *testing.T) {
	idx := loadTestIndex(t)

	// Test known sequence
	id, _ := util.NewUIDFromString("A001")
	seq := FindSequenceById(idx, id)
	if seq == nil {
		t.Errorf("FindById: did not find A000001")
	} else if seq.Id.String() != "A000001" {
		t.Errorf("FindById: got %q, want %q", seq.Id, "A000001")
	}

	// Test non-existent ID
	id, _ = util.NewUIDFromString("A999999")
	seq = FindSequenceById(idx, id)
	if seq != nil {
		t.Errorf("FindById: expected nil for non-existent ID, got %q", seq.Id)
	}

	// Test first and last
	first := idx.Sequences[0].Id
	last := idx.Sequences[len(idx.Sequences)-1].Id
	if FindSequenceById(idx, first) == nil {
		t.Errorf("FindById: did not find first sequence %q", first)
	}
	if FindSequenceById(idx, last) == nil {
		t.Errorf("FindById: did not find last sequence %q", last)
	}

	// Test all loaded sequences
	for _, seq := range idx.Sequences {
		got := FindSequenceById(idx, seq.Id)
		if got == nil || got.Id != seq.Id {
			t.Errorf("FindById: failed for %q", seq.Id)
		}
		s := seq.Id.String()
		id2, _ := util.NewUIDFromString(string(s[0]) + "0" + s[1:])
		got2 := FindSequenceById(idx, id2)
		if got2 == nil || got2.Id != seq.Id {
			t.Errorf("FindById: failed for %q", seq.Id)
		}
	}
}

func TestSearchSequences(t *testing.T) {
	idx := loadTestIndex(t)

	// Search by query string (name substring)
	results, total := SearchSequences(idx, "Kolakoski", 0, 0)
	if total != 1 || len(results) != 1 {
		t.Errorf("Search Kolakoski: got %d results, want 1", total)
	} else if !strings.Contains(results[0].Name, "Kolakoski") {
		t.Errorf("Search Kolakoski: wrong sequence name %q", results[0].Name)
	}

	// Search by included keyword (as +core)
	results, total = SearchSequences(idx, "+core", 0, 0)
	if total != 7 {
		t.Errorf("Search +core: got total=%d, want 7", total)
	}
	for _, seq := range results {
		gotKeywords := DecodeKeywords(seq.Keywords)
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

	// Search by excluded keyword (as -hard)
	results, total = SearchSequences(idx, "-hard", 0, 0)
	if total != 9 {
		t.Errorf("Search -hard: got total=%d, want 9", total)
	}
	for _, seq := range results {
		gotKeywords := DecodeKeywords(seq.Keywords)
		for _, kw := range gotKeywords {
			if kw == "hard" {
				t.Errorf("Search forbidden hard: sequence %q contains forbidden keyword", seq.Id)
			}
		}
	}

	// Search with query tokens (all must match)
	results, total = SearchSequences(idx, "groups order", 0, 0)
	if total != 1 || len(results) != 1 || !strings.Contains(results[0].Name, "groups") || !strings.Contains(results[0].Name, "order") {
		t.Errorf("Search groups order: got %d results, want 1 with correct name", total)
	}

	// Pagination: skip and limit
	allResults, allTotal := SearchSequences(idx, "", 0, 0)
	if allTotal != 10 || len(allResults) != 10 {
		t.Fatalf("All results: got %d results, want 10", allTotal)
	}
	paged, _ := SearchSequences(idx, "", 2, 1)
	if len(paged) != 2 {
		t.Errorf("Pagination: got %d results, want 2", len(paged))
	}
	if paged[0].Id != allResults[1].Id || paged[1].Id != allResults[2].Id {
		t.Errorf("Pagination: unexpected sequence IDs")
	}
}

func checkSearchByID(t *testing.T, idx *DataIndex, query string, expectedID string) {
	results, total := SearchSequences(idx, query, 0, 0)
	if total != 1 || len(results) != 1 {
		t.Errorf("SearchSequences by ID (%s): got %d results, want 1", query, total)
	} else if results[0].Id.String() != expectedID {
		t.Errorf("SearchSequences by ID (%s): got %q, want %q", query, results[0].Id.String(), expectedID)
	}
}

func TestSearchSequencesByID(t *testing.T) {
	idx := loadTestIndex(t)
	checkSearchByID(t, idx, "A000001", "A000001")
	checkSearchByID(t, idx, "A1", "A000001")
	checkSearchByID(t, idx, "A000002", "A000002")
	checkSearchByID(t, idx, "A2", "A000002")
}
