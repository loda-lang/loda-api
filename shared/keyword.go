package shared

import (
	"fmt"
)

// List of all keywords
var KeywordList = []string{
	"base", "bref", "cofr", "conjecture", "cons", "core", "dead",
	"decimal-expansion", "dumb", "easy", "egf-expansion", "eigen",
	"fini", "formula", "frac", "full", "gf-expansion", "hard", "less",
	"loda", "loda-formula", "loda-inceval", "loda-logeval", "loda-loop",
	"look", "more", "mult", "nice", "nonn", "obsc", "pari", "sign",
	"tabf", "tabl", "unkn", "walk", "word",
}

// KeywordDescriptions provides a description for each keyword
var KeywordDescriptions = map[string]string{
	"base":              "Sequence terms are defined based on a number representation in a particular base format, e.g. decimal or binary",
	"bref":              "Sequences with very few terms",
	"cofr":              "Continued fractions for (usually irrational) constants",
	"conjecture":        "Sequences that include conjectures in their OEIS description",
	"cons":              "Sequences that give terms for decimal expansions",
	"core":              "Core sequences of the OEIS database",
	"dead":              "Errornous or duplicate sequences",
	"decimal-expansion": "Decimal expansions of constants",
	"dumb":              "Unimportant sequences from non-mathematical contexts",
	"easy":              "Sequences that are easy to comute and understand",
	"egf-expansion":     "Expansions of exponential generating functions",
	"eigen":             "Eigensequences",
	"fini":              "Finite sequences",
	"formula":           "Formulas exist in OEIS entries for these sequences",
	"frac":              "Numerators or denominators of sequence of rationals",
	"full":              "Finite sequence with all terms available",
	"gf-expansion":      "Expansions of generating functions",
	"hard":              "Sequences that are hard to compute",
	"less":              "Less interesting sequences",
	"loda":              "LODA programs exist for these sequence",
	"loda-formula":      "Formulas generated from a LODA programs exist for these sequences",
	"loda-inceval":      "LODA programs that can be computed incrementally exist for these sequences",
	"loda-logeval":      "LODA programs with logarithmic complexity exist for these sequences",
	"loda-loop":         "LODA programs with loop exist for these sequences",
	"look":              "Pin or scatter plots reveal interesting information",
	"more":              "Sequences that need more terms",
	"mult":              "Multiplicative functions",
	"nice":              "Exceptionally \"nice\" sequences",
	"nonn":              "Sequences with only non-negative terms",
	"obsc":              "Obscure sequences: descriptions are known, but difficult to understand",
	"pari":              "PARI/GP programs exist for these sequence",
	"sign":              "Sequences with negative terms",
	"tabf":              "Tables with irregular row lengths",
	"tabl":              "Regular tables: fixed row length",
	"unkn":              "Sequences whose definition is unknown",
	"walk":              "Sequences that contain walks through a lattice",
	"word":              "Numbers related to a given natural language",
}

// GetKeywordDescription returns the description for a given keyword, or an empty string if not found
func GetKeywordDescription(keyword string) string {
	return KeywordDescriptions[keyword]
}

// Map for fast lookup
var keywordToBit = func() map[string]uint {
	m := make(map[string]uint)
	for i, k := range KeywordList {
		m[k] = uint(i)
	}
	return m
}()

func IsKeyword(s string) bool {
	_, ok := keywordToBit[s]
	return ok
}

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

// ContainsAllKeywords returns true if all keywords in bits2 are present in bits1
func ContainsAllKeywords(bits1, bits2 uint64) bool {
	return bits1&bits2 == bits2
}

// ContainsNoKeywords returns true if none of the keywords in bits2 are present in bits1
func ContainsNoKeywords(bits1, bits2 uint64) bool {
	return bits1&bits2 == 0
}
