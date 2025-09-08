package shared

import (
	"strings"

	"github.com/loda-lang/loda-api/util"
)

func FindById(programs []Program, id util.UID) *Program {
	d := id.Domain()
	n := int64(id.Number())
	if n >= 0 && n < int64(len(programs)) && programs[n].Id.Domain() == d {
		k := programs[n].Id.Number()
		if k == n {
			return &programs[n]
		} else if k < n {
			// Search forward
			for i := n + 1; i < int64(len(programs)); i++ {
				if programs[i].Id.Domain() != d {
					break
				}
				if programs[i].Id.Equals(id) {
					return &programs[i]
				}
			}
		} else {
			// Search backward
			for i := n - 1; i >= 0; i-- {
				if programs[i].Id.Domain() != d {
					break
				}
				if programs[i].Id.Equals(id) {
					return &programs[i]
				}
			}
		}
	} else {
		// Full search
		for _, s := range programs {
			if s.Id.Equals(id) {
				return &s
			}
		}
	}
	return nil
}

// Search finds programs matching the query and applies pagination.
func Search(programs []Program, query string, limit, skip int) []Program {
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
		return nil
	}
	excluded, err := EncodeKeywords(exc)
	if err != nil {
		return nil
	}

	count := 0
	var results []Program
	for _, seq := range programs {
		// Check included and excluded keywords
		if !ContainsAllKeywords(seq.Keywords, included) {
			continue
		}
		if !ContainsNoKeywords(seq.Keywords, excluded) {
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
		// Pagination: skip first 'skip' matches, then collect up to 'limit'
		if count < skip {
			count++
			continue
		}
		if limit > 0 && len(results) >= limit {
			break
		}
		results = append(results, seq)
	}
	return results
}
