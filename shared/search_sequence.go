package shared

import (
	"strings"

	"github.com/loda-lang/loda-api/util"
)

func FindSequenceById(idx *DataIndex, id util.UID) *Sequence {
	d := id.Domain()
	n := int64(id.Number())
	if n >= 0 && n < int64(len(idx.Sequences)) && idx.Sequences[n].Id.Domain() == d {
		k := idx.Sequences[n].Id.Number()
		if k == n {
			return &idx.Sequences[n]
		} else if k < n {
			// Search forward
			for i := n + 1; i < int64(len(idx.Sequences)); i++ {
				if idx.Sequences[i].Id.Domain() != d {
					break
				}
				if idx.Sequences[i].Id.Equals(id) {
					return &idx.Sequences[i]
				}
			}
		} else {
			// Search backward
			for i := n - 1; i >= 0; i-- {
				if idx.Sequences[i].Id.Domain() != d {
					break
				}
				if idx.Sequences[i].Id.Equals(id) {
					return &idx.Sequences[i]
				}
			}
		}
	} else {
		// Full search
		for _, s := range idx.Sequences {
			if s.Id.Equals(id) {
				return &s
			}
		}
	}
	return nil
}

// Search returns paginated results and total count of all matches
func SearchSequences(idx *DataIndex, query string, limit, skip int) ([]Sequence, int) {
	// Split the query into lower-case tokens
	var tokens []string
	if query != "" {
		tokens = strings.Fields(query)
		for i, t := range tokens {
			tokens[i] = strings.ToLower(t)
		}
	}

	// Extract included/excluded keywords and remove them from tokens
	var inc, exc []string
	filteredTokens := tokens[:0] // reuse underlying array
	for _, t := range tokens {
		if IsKeyword(t) {
			inc = append(inc, t)
		} else if len(t) > 1 && t[0] == '+' && IsKeyword(t[1:]) {
			inc = append(inc, t[1:])
		} else if len(t) > 1 && (t[0] == '-' || t[0] == '!') && IsKeyword(t[1:]) {
			exc = append(exc, t[1:])
		} else {
			filteredTokens = append(filteredTokens, t)
		}
	}
	included, err := EncodeKeywords(inc)
	if err != nil {
		return nil, 0
	}
	excluded, err := EncodeKeywords(exc)
	if err != nil {
		return nil, 0
	}

	count := 0
	var results []Sequence
	var total int
	for _, seq := range idx.Sequences {
		// Check included and excluded keywords
		if !HasAllKeywords(seq.Keywords, included) {
			continue
		}
		if !HasNoKeywords(seq.Keywords, excluded) {
			continue
		}
		match := true
		// Query string filtering (case-insensitive, all tokens must be present in name)
		if len(filteredTokens) > 0 {
			nameLower := strings.ToLower(seq.Name)
			for _, t := range filteredTokens {
				if !strings.Contains(nameLower, t) {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		total++
		if count < skip {
			count++
			continue
		}
		if limit > 0 && len(results) >= limit {
			continue
		}
		results = append(results, seq)
	}
	return results, total
}
