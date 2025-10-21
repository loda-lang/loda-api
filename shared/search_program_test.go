package shared

import (
	"testing"

	"github.com/loda-lang/loda-api/util"
)

func makeTestData() *DataIndex {
	programs := []Program{
		{Id: mustUID("A000002"), Name: "Kolakoski sequence", Keywords: mustKeywords([]string{"nonn", "core", "easy", "nice"})},
		{Id: mustUID("A000004"), Name: "The zero sequence.", Keywords: mustKeywords([]string{"core", "easy", "nonn", "mult"})},
		{Id: mustUID("A000006"), Name: "Integer part of square root of n-th prime.", Keywords: mustKeywords([]string{"nonn", "easy", "nice"})},
		{Id: mustUID("A000007"), Name: "The characteristic function of {0}: a(n) = 0^n.", Keywords: mustKeywords([]string{"core", "nonn", "mult", "cons", "easy"})},
	}
	return &DataIndex{Programs: programs}
}

func mustUID(s string) util.UID {
	uid, err := util.NewUIDFromString(s)
	if err != nil {
		panic(err)
	}
	return uid
}

func mustKeywords(kw []string) uint64 {
	k, err := EncodeKeywords(kw)
	if err != nil {
		panic(err)
	}
	return k
}

func TestFindProgramById(t *testing.T) {
	programs := makeTestData().Programs
	// Test existing
	p := FindProgramById(programs, mustUID("A000004"))
	if p == nil || p.Name != "The zero sequence." {
		t.Errorf("FindById failed for A000004")
	}
	// Test non-existing
	p = FindProgramById(programs, mustUID("A999999"))
	if p != nil {
		t.Errorf("FindById should return nil for non-existent ID")
	}
	// Test first and last
	if FindProgramById(programs, programs[0].Id) == nil {
		t.Errorf("FindById failed for first program")
	}
	if FindProgramById(programs, programs[len(programs)-1].Id) == nil {
		t.Errorf("FindById failed for last program")
	}
}

func TestSearchPrograms(t *testing.T) {
	idx := makeTestData()
	// Search by name substring
	results, total := SearchPrograms(idx, "Kolakoski", 0, 0, false)
	if total != 1 || len(results) != 1 || results[0].Name != "Kolakoski sequence" {
		t.Errorf("Search by name failed: got total=%d, len=%d", total, len(results))
	}
	// Search by included keyword
	results, total = SearchPrograms(idx, "+core", 0, 0, false)
	if total != 3 {
		t.Errorf("Search +core: got total=%d, want 3", total)
	}
	for _, p := range results {
		if !HasKeyword(p.Keywords, "core") {
			t.Errorf("Search +core: missing keyword")
		}
	}
	// Search by excluded keyword
	results, total = SearchPrograms(idx, "-mult", 0, 0, false)
	if total != 2 {
		t.Errorf("Search -mult: got total=%d, want 2", total)
	}
	for _, p := range results {
		if HasKeyword(p.Keywords, "mult") {
			t.Errorf("Search -mult: should not contain 'mult'")
		}
	}
	// Search with multiple tokens
	// Search by multiple tokens (all must match)
	results, total = SearchPrograms(idx, "zero sequence", 0, 0, false)
	if total != 1 || len(results) != 1 || results[0].Name != "The zero sequence." {
		t.Errorf("Search with multiple tokens failed: got total=%d, len=%d", total, len(results))
	}
	// Pagination
	all, allTotal := SearchPrograms(idx, "", 0, 0, false)
	if allTotal != 4 {
		t.Errorf("All: got total=%d, want 4", allTotal)
	}
	paged, _ := SearchPrograms(idx, "", 2, 1, false)
	if len(paged) != 2 || paged[0].Id != all[1].Id || paged[1].Id != all[2].Id {
		t.Errorf("Pagination failed")
	}
}

func TestSearchProgramsShuffle(t *testing.T) {
	programs := makeTestData()
	// Get results without shuffle
	results1, total1 := SearchPrograms(programs, "+core", 0, 0, false)
	results2, total2 := SearchPrograms(programs, "+core", 0, 0, false)
	// Results should be the same when not shuffled
	if total1 != total2 {
		t.Errorf("Non-shuffled searches have different totals: %d vs %d", total1, total2)
	}
	if len(results1) != len(results2) {
		t.Errorf("Non-shuffled searches have different result lengths: %d vs %d", len(results1), len(results2))
	}
	for i := range results1 {
		if results1[i].Id != results2[i].Id {
			t.Errorf("Non-shuffled results differ at position %d: %s vs %s", i, results1[i].Id, results2[i].Id)
		}
	}
	// Test with shuffle enabled - we can't easily test randomness, but we can test that it doesn't break anything
	shuffled, totalShuffled := SearchPrograms(programs, "+core", 0, 0, true)
	if totalShuffled != total1 {
		t.Errorf("Shuffled search has different total: %d vs %d", totalShuffled, total1)
	}
	if len(shuffled) != len(results1) {
		t.Errorf("Shuffled search has different result length: %d vs %d", len(shuffled), len(results1))
	}
}
