package shared

import (
	"fmt"
)

// Encoded keyword constants for efficient bitwise operations
var (
	KeywordBaseBits         uint64
	KeywordBrefBits         uint64
	KeywordCofrBits         uint64
	KeywordConjectureBits   uint64
	KeywordConsBits         uint64
	KeywordCoreBits         uint64
	KeywordDeadBits         uint64
	KeywordDecimalExpBits   uint64
	KeywordDumbBits         uint64
	KeywordEasyBits         uint64
	KeywordEGFExpBits       uint64
	KeywordEigenBits        uint64
	KeywordFiniBits         uint64
	KeywordFormulaBits      uint64
	KeywordFracBits         uint64
	KeywordFullBits         uint64
	KeywordGFExpBits        uint64
	KeywordHardBits         uint64
	KeywordLessBits         uint64
	KeywordLodaBits         uint64
	KeywordLodaFormulaBits  uint64
	KeywordLodaIncevalBits  uint64
	KeywordLodaIndirectBits uint64
	KeywordLodaLeanBits     uint64
	KeywordLodaLogevalBits  uint64
	KeywordLodaLoopBits     uint64
	KeywordLodaPariBits     uint64
	KeywordLodaVirevalBits  uint64
	KeywordLookBits         uint64
	KeywordMoreBits         uint64
	KeywordMultBits         uint64
	KeywordNiceBits         uint64
	KeywordNonnBits         uint64
	KeywordObscBits         uint64
	KeywordPariBits         uint64
	KeywordSignBits         uint64
	KeywordTabfBits         uint64
	KeywordTablBits         uint64
	KeywordUnknBits         uint64
	KeywordWalkBits         uint64
	KeywordWordBits         uint64
)

func init() {
	KeywordBaseBits = MustEncodeKeyword("base")
	KeywordBrefBits = MustEncodeKeyword("bref")
	KeywordCofrBits = MustEncodeKeyword("cofr")
	KeywordConjectureBits = MustEncodeKeyword("conjecture")
	KeywordConsBits = MustEncodeKeyword("cons")
	KeywordCoreBits = MustEncodeKeyword("core")
	KeywordDeadBits = MustEncodeKeyword("dead")
	KeywordDecimalExpBits = MustEncodeKeyword("decimal-expansion")
	KeywordDumbBits = MustEncodeKeyword("dumb")
	KeywordEasyBits = MustEncodeKeyword("easy")
	KeywordEGFExpBits = MustEncodeKeyword("egf-expansion")
	KeywordEigenBits = MustEncodeKeyword("eigen")
	KeywordFiniBits = MustEncodeKeyword("fini")
	KeywordFormulaBits = MustEncodeKeyword("formula")
	KeywordFracBits = MustEncodeKeyword("frac")
	KeywordFullBits = MustEncodeKeyword("full")
	KeywordGFExpBits = MustEncodeKeyword("gf-expansion")
	KeywordHardBits = MustEncodeKeyword("hard")
	KeywordLessBits = MustEncodeKeyword("less")
	KeywordLodaBits = MustEncodeKeyword("loda")
	KeywordLodaFormulaBits = MustEncodeKeyword("loda-formula")
	KeywordLodaIncevalBits = MustEncodeKeyword("loda-inceval")
	KeywordLodaIndirectBits = MustEncodeKeyword("loda-indirect")
	KeywordLodaLeanBits = MustEncodeKeyword("loda-lean")
	KeywordLodaLogevalBits = MustEncodeKeyword("loda-logeval")
	KeywordLodaLoopBits = MustEncodeKeyword("loda-loop")
	KeywordLodaPariBits = MustEncodeKeyword("loda-pari")
	KeywordLodaVirevalBits = MustEncodeKeyword("loda-vireval")
	KeywordLookBits = MustEncodeKeyword("look")
	KeywordMoreBits = MustEncodeKeyword("more")
	KeywordMultBits = MustEncodeKeyword("mult")
	KeywordNiceBits = MustEncodeKeyword("nice")
	KeywordNonnBits = MustEncodeKeyword("nonn")
	KeywordObscBits = MustEncodeKeyword("obsc")
	KeywordPariBits = MustEncodeKeyword("pari")
	KeywordSignBits = MustEncodeKeyword("sign")
	KeywordTabfBits = MustEncodeKeyword("tabf")
	KeywordTablBits = MustEncodeKeyword("tabl")
	KeywordUnknBits = MustEncodeKeyword("unkn")
	KeywordWalkBits = MustEncodeKeyword("walk")
	KeywordWordBits = MustEncodeKeyword("word")
}

// List of all keywords
var KeywordList = []string{
	"base", "bref", "cofr", "conjecture", "cons", "core", "dead",
	"decimal-expansion", "dumb", "easy", "egf-expansion", "eigen",
	"fini", "formula", "frac", "full", "gf-expansion", "hard", "less",
	"loda", "loda-formula", "loda-inceval", "loda-indirect", "loda-lean", "loda-logeval", "loda-loop", "loda-pari", "loda-vireval",
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
	"loda-indirect":     "LODA programs that uses indirect operands exist for these sequences",
	"loda-lean":         "LODA programs with Lean proofs exist for these sequences",
	"loda-logeval":      "LODA programs with logarithmic complexity exist for these sequences",
	"loda-loop":         "LODA programs with loop exist for these sequences",
	"loda-pari":         "LODA programs with PARI/GP implementations exist for these sequences",
	"loda-vireval":      "LODA programs that support virtual evaluation exist for these sequences",
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

// MustEncodeKeyword encodes a single keyword and panics if it is unknown.
func MustEncodeKeyword(keyword string) uint64 {
	bits, err := EncodeKeywords([]string{keyword})
	if err != nil {
		panic(err)
	}
	return bits
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

// HasKeyword returns true if the given keyword is present in the bits
func HasKeyword(bits1 uint64, keyword string) bool {
	bits2, err := EncodeKeywords([]string{keyword})
	return err == nil && HasAllKeywords(bits1, bits2)
}

// HasAllKeywords returns true if all keywords in bits2 are present in bits1
func HasAllKeywords(bits1, bits2 uint64) bool {
	return bits1&bits2 == bits2
}

// HasNoKeywords returns true if none of the keywords in bits2 are present in bits1
func HasNoKeywords(bits1, bits2 uint64) bool {
	return bits1&bits2 == 0
}

// MergeKeywords merges two keyword bitmasks into one
func MergeKeywords(bits1, bits2 uint64) uint64 {
	return bits1 | bits2
}

// CountKeywordsInBits increments the count for each keyword present in bits.
// The map should have keys of type uint64 (bitmask for each keyword) and int values.
func CountKeywordsInBits(bits uint64, counts *map[uint64]int) {
	for i := range KeywordList {
		mask := uint64(1) << uint(i)
		if bits&mask != 0 {
			(*counts)[mask]++
		}
	}
}
