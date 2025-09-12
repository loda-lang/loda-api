package shared

import (
	"testing"

	"github.com/loda-lang/loda-api/util"
)

func makeTestPrograms() []Program {
	return []Program{
		{Id: mustUID("A000002"), Name: "Kolakoski sequence", Keywords: mustKeywords([]string{"nonn", "core", "easy", "nice"})},
		{Id: mustUID("A000004"), Name: "The zero sequence.", Keywords: mustKeywords([]string{"core", "easy", "nonn", "mult"})},
		{Id: mustUID("A000006"), Name: "Integer part of square root of n-th prime.", Keywords: mustKeywords([]string{"nonn", "easy", "nice"})},
		{Id: mustUID("A000007"), Name: "The characteristic function of {0}: a(n) = 0^n.", Keywords: mustKeywords([]string{"core", "nonn", "mult", "cons", "easy"})},
	}
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
	programs := makeTestPrograms()
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
	programs := makeTestPrograms()
	// Search by name substring
	results, total := SearchPrograms(programs, "Kolakoski", 0, 0)
	if total != 1 || len(results) != 1 || results[0].Name != "Kolakoski sequence" {
		t.Errorf("Search by name failed: got total=%d, len=%d", total, len(results))
	}
	// Search by included keyword
	results, total = SearchPrograms(programs, "+core", 0, 0)
	if total != 3 {
		t.Errorf("Search +core: got total=%d, want 3", total)
	}
	for _, p := range results {
		if !HasKeyword(p.Keywords, "core") {
			t.Errorf("Search +core: missing keyword")
		}
	}
	// Search by excluded keyword
	results, total = SearchPrograms(programs, "-mult", 0, 0)
	if total != 2 {
		t.Errorf("Search -mult: got total=%d, want 2", total)
	}
	for _, p := range results {
		if HasKeyword(p.Keywords, "mult") {
			t.Errorf("Search -mult: should not contain 'mult'")
		}
	}
	// Search with multiple tokens
	results, total = SearchPrograms(programs, "zero sequence", 0, 0)
	if total != 1 || len(results) != 1 || results[0].Name != "The zero sequence." {
		t.Errorf("Search with multiple tokens failed: got total=%d, len=%d", total, len(results))
	}
	// Pagination
	all, allTotal := SearchPrograms(programs, "", 0, 0)
	if allTotal != 4 {
		t.Errorf("All: got total=%d, want 4", allTotal)
	}
	paged, _ := SearchPrograms(programs, "", 2, 1)
	if len(paged) != 2 || paged[0].Id != all[1].Id || paged[1].Id != all[2].Id {
		t.Errorf("Pagination failed")
	}
}
