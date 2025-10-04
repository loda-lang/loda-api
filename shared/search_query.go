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
}

func ParseSearchQuery(query string) SearchQuery {
	var rawTokens, tokens []string
	if query != "" {
		rawTokens = strings.Fields(query)
		tokens = make([]string, len(rawTokens))
		for i, t := range rawTokens {
			tokens[i] = strings.ToLower(t)
		}
	}
	var inc, exc []string
	filteredTokens := tokens[:0] // reuse underlying array
	var uidTokens []util.UID
	for i, t := range tokens {
		raw := t
		if len(rawTokens) > i {
			raw = rawTokens[i]
		}
		if IsKeyword(t) {
			inc = append(inc, t)
		} else if len(t) > 1 && t[0] == '+' && IsKeyword(t[1:]) {
			inc = append(inc, t[1:])
		} else if len(t) > 1 && (t[0] == '-' || t[0] == '!') && IsKeyword(t[1:]) {
			exc = append(exc, t[1:])
		} else {
			if uid, err := util.NewUIDFromString(raw); err == nil {
				uidTokens = append(uidTokens, uid)
			} else {
				filteredTokens = append(filteredTokens, t)
			}
		}
	}
	included, _ := EncodeKeywords(inc)
	excluded, _ := EncodeKeywords(exc)
	return SearchQuery{
		RawTokens:        rawTokens,
		Tokens:           tokens,
		UIDTokens:        uidTokens,
		FilteredTokens:   filteredTokens,
		IncludedKeywords: included,
		ExcludedKeywords: excluded,
	}
}
