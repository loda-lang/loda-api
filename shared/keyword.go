package shared

import (
	"fmt"
)

// List of all keywords in the order from the OpenAPI spec
var KeywordList = []string{
	"base", "bref", "cofr", "conjecture", "cons", "core", "dead",
	"decimal-expansion", "dumb", "easy", "egf-expansion", "eigen",
	"fini", "formula", "frac", "full", "gf-expansion", "hard", "less",
	"loda", "loda-formula", "loda-inceval", "loda-logeval", "loda-loop",
	"look", "more", "mult", "nice", "nonn", "obsc", "pari", "sign",
	"tabf", "tabl", "unkn", "walk", "word",
}

// Map for fast lookup
var keywordToBit = func() map[string]uint {
	m := make(map[string]uint)
	for i, k := range KeywordList {
		m[k] = uint(i)
	}
	return m
}()

// EncodeKeywords encodes a list of keywords into a uint64 bitmask
func EncodeKeywords(keywords []string) (uint64, error) {
	var bits uint64
	for _, k := range keywords {
		bit, ok := keywordToBit[k]
		if !ok {
			return 0, fmt.Errorf("unknown keyword: %s", k)
		}
		bits |= 1 << bit
	}
	return bits, nil
}

// DecodeKeywords decodes a uint64 bitmask into a list of keywords
func DecodeKeywords(bits uint64) []string {
	var result []string
	for i, k := range KeywordList {
		if bits&(1<<uint(i)) != 0 {
			result = append(result, k)
		}
	}
	return result
}

// ContainsAllKeywords returns true if all keywords in bits1 are present in bits2
func ContainsAllKeywords(bits1, bits2 uint64) bool {
	return bits1&bits2 == bits1
}

// ContainsNoKeywords returns true if none of the keywords in bits1 are present in bits2
func ContainsNoKeywords(bits1, bits2 uint64) bool {
	return bits1&bits2 == 0
}
