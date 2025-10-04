package shared

import (
	"strings"

	"github.com/loda-lang/loda-api/util"
)

func FindProgramById(programs []Program, id util.UID) *Program {
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

// SearchPrograms returns paginated results and total count of all matches
func SearchPrograms(programs []Program, query string, limit, skip int) ([]Program, int) {
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
	var results []Program
	var total int
	for _, prog := range programs {
		// Check included and excluded keywords
		if !HasAllKeywords(prog.Keywords, included) {
			continue
		}
		if !HasNoKeywords(prog.Keywords, excluded) {
			continue
		}
		match := true
		// Query string filtering (case-insensitive, all tokens must be present in name or submitter)
		if len(filteredTokens) > 0 {
			nameLower := strings.ToLower(prog.Name)
			submitterLower := ""
			if prog.Submitter != nil {
				submitterLower = strings.ToLower(prog.Submitter.Name)
			}
			for _, t := range filteredTokens {
				if !strings.Contains(nameLower, t) && (submitterLower == "" || !strings.Contains(submitterLower, t)) {
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
		results = append(results, prog)
	}
	return results, total
}
