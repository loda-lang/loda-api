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
		sort.Strings(decoded)
		sort.Strings(input)
		if !reflect.DeepEqual(decoded, input) {
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
