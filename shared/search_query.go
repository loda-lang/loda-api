package shared

import (
	"strings"

	"github.com/loda-lang/loda-api/util"
)

type SearchQuery struct {
	RawTokens        []string
	Tokens           []string
	UIDTokens        []util.UID
	FilteredTokens   []string
	IncludedKeywords uint64
	ExcludedKeywords uint64
	IncludedOps      uint64
	ExcludedOps      uint64
}

func ParseSearchQuery(query string, opTypeIndex *OpTypeIndex) SearchQuery {
	var rawTokens, tokens []string
	if query != "" {
		rawTokens = strings.Fields(query)
		tokens = make([]string, len(rawTokens))
		for i, t := range rawTokens {
			tokens[i] = strings.ToLower(t)
		}
	}
	var incKw, excKw, incOps, excOps []string
	filteredTokens := tokens[:0] // reuse underlying array
	var uidTokens []util.UID
	for i, t := range tokens {
		raw := t
		if len(rawTokens) > i {
			raw = rawTokens[i]
		}
		if IsKeyword(t) {
			incKw = append(incKw, t)
		} else if len(t) > 1 && t[0] == '+' && IsKeyword(t[1:]) {
			incKw = append(incKw, t[1:])
		} else if len(t) > 1 && (t[0] == '-' || t[0] == '!') && IsKeyword(t[1:]) {
			excKw = append(excKw, t[1:])
		} else if opTypeIndex != nil && opTypeIndex.IsOperationType(t) {
			incOps = append(incOps, t)
		} else if opTypeIndex != nil && len(t) > 1 && t[0] == '+' && opTypeIndex.IsOperationType(t[1:]) {
			incOps = append(incOps, t[1:])
		} else if opTypeIndex != nil && len(t) > 1 && (t[0] == '-' || t[0] == '!') && opTypeIndex.IsOperationType(t[1:]) {
			excOps = append(excOps, t[1:])
		} else {
			if uid, err := util.NewUIDFromString(raw); err == nil {
				uidTokens = append(uidTokens, uid)
			} else {
				filteredTokens = append(filteredTokens, t)
			}
		}
	}
	includedKw, _ := EncodeKeywords(incKw)
	excludedKw, _ := EncodeKeywords(excKw)
	var includedOps, excludedOps uint64
	if opTypeIndex != nil {
		includedOps, _ = opTypeIndex.EncodeOperationTypes(incOps)
		excludedOps, _ = opTypeIndex.EncodeOperationTypes(excOps)
	}
	return SearchQuery{
		RawTokens:        rawTokens,
		Tokens:           tokens,
		UIDTokens:        uidTokens,
		FilteredTokens:   filteredTokens,
		IncludedKeywords: includedKw,
		ExcludedKeywords: excludedKw,
		IncludedOps:      includedOps,
		ExcludedOps:      excludedOps,
	}
}
