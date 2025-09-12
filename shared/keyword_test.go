package shared

import (
	"reflect"
	"sort"
	"testing"
)

func TestEncodeDecodeKeywords(t *testing.T) {
	cases := [][]string{
		{},
		{"base"},
		{"word"},
		{"base", "word"},
		{"loda", "nice", "hard"},
	}
	for _, input := range cases {
		bits, err := EncodeKeywords(input)
		if err != nil {
			t.Errorf("EncodeKeywords(%v) unexpected error: %v", input, err)
		}
		decoded := DecodeKeywords(bits)
		// Order is not guaranteed, so sort for comparison
		sortedDecoded := append([]string(nil), decoded...)
		sort.Strings(sortedDecoded)
		sortedInput := append([]string(nil), input...)
		sort.Strings(sortedInput)
		if !reflect.DeepEqual(sortedDecoded, sortedInput) {
			t.Errorf("round-trip Encode/Decode failed: got %v, want %v", decoded, input)
		}
	}
}

func TestEncodeKeywordsUnknown(t *testing.T) {
	_, err := EncodeKeywords([]string{"notakeyword"})
	if err == nil {
		t.Error("expected error for unknown keyword, got nil")
	}
}

func TestHasAllKeywords(t *testing.T) {
	a, _ := EncodeKeywords([]string{"base", "word"})
	b, _ := EncodeKeywords([]string{"base", "word", "nice"})
	c, _ := EncodeKeywords([]string{"base"})
	d, _ := EncodeKeywords([]string{"nice"})

	if !HasAllKeywords(b, a) {
		t.Error("expected a to be contained in b")
	}
	if !HasAllKeywords(a, c) {
		t.Error("expected c to be contained in a")
	}
	if HasAllKeywords(a, b) {
		t.Error("expected a not to be contained in b")
	}
	if HasAllKeywords(a, d) {
		t.Error("expected d not to be contained in a")
	}
}

func TestHasNoKeywords(t *testing.T) {
	a, _ := EncodeKeywords([]string{"base", "word"})
	b, _ := EncodeKeywords([]string{"nice", "hard"})
	c, _ := EncodeKeywords([]string{"word"})
	d, _ := EncodeKeywords([]string{})

	if !HasNoKeywords(a, b) {
		t.Error("expected a and b to have no keywords in common")
	}
	if HasNoKeywords(a, c) {
		t.Error("expected a and c to have at least one keyword in common")
	}
	if !HasNoKeywords(d, a) {
		t.Error("expected empty set to have no keywords in common with a")
	}
	if !HasNoKeywords(d, d) {
		t.Error("expected empty sets to have no keywords in common")
	}
}

func TestHasKeyword(t *testing.T) {
	bits, _ := EncodeKeywords([]string{"base", "word", "nice"})
	if !HasKeyword(bits, "base") {
		t.Error("expected HasKeyword to be true for 'base'")
	}
	if !HasKeyword(bits, "word") {
		t.Error("expected HasKeyword to be true for 'word'")
	}
	if !HasKeyword(bits, "nice") {
		t.Error("expected HasKeyword to be true for 'nice'")
	}
	if HasKeyword(bits, "hard") {
		t.Error("expected HasKeyword to be false for 'hard'")
	}
	if HasKeyword(bits, "notakeyword") {
		t.Error("expected HasKeyword to be false for unknown keyword")
	}
}
