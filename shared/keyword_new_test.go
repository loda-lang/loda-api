package shared

import (
	"testing"
)

func TestNewKeywordsLodaPariLean(t *testing.T) {
	// Test encoding and decoding
	keywords, err := EncodeKeywords([]string{"loda-pari", "loda-lean"})
	if err != nil {
		t.Fatalf("Failed to encode new keywords: %v", err)
	}

	decoded := DecodeKeywords(keywords)
	if len(decoded) != 2 {
		t.Fatalf("Expected 2 keywords, got %d: %v", len(decoded), decoded)
	}

	// Test individual keyword detection
	if !HasKeyword(keywords, "loda-pari") {
		t.Errorf("HasKeyword failed for loda-pari")
	}
	if !HasKeyword(keywords, "loda-lean") {
		t.Errorf("HasKeyword failed for loda-lean")
	}

	// Test that descriptions exist
	pariDesc := GetKeywordDescription("loda-pari")
	if pariDesc == "" {
		t.Errorf("No description for loda-pari")
	}
	leanDesc := GetKeywordDescription("loda-lean")
	if leanDesc == "" {
		t.Errorf("No description for loda-lean")
	}

	// Test that keyword constants are set properly
	if KeywordLodaPariBits == 0 {
		t.Errorf("KeywordLodaPariBits not initialized")
	}
	if KeywordLodaLeanBits == 0 {
		t.Errorf("KeywordLodaLeanBits not initialized")
	}
}
