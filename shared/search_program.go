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
	sq := ParseSearchQuery(query)
	count := 0
	var results []Program
	var total int
	for _, prog := range programs {
		// Check included and excluded keywords
		if !HasAllKeywords(prog.Keywords, sq.IncludedKeywords) {
			continue
		}
		if !HasNoKeywords(prog.Keywords, sq.ExcludedKeywords) {
			continue
		}
		match := true
		// Query string filtering (case-insensitive, all tokens must be present in name, submitter, or ID)
		if len(sq.FilteredTokens) > 0 || len(sq.UIDTokens) > 0 {
			nameLower := strings.ToLower(prog.Name)
			submitterLower := ""
			if prog.Submitter != nil {
				submitterLower = strings.ToLower(prog.Submitter.Name)
			}
			// Check UID tokens: match if the program ID equals the UID or the UID string is contained in the name
			for _, uid := range sq.UIDTokens {
				if !prog.Id.Equals(uid) && !strings.Contains(prog.Name, uid.String()) {
					match = false
					break
				}
			}
			// Check string tokens
			if match && len(sq.FilteredTokens) > 0 {
				for _, t := range sq.FilteredTokens {
					if !strings.Contains(nameLower, t) && (submitterLower == "" || !strings.Contains(submitterLower, t)) {
						match = false
						break
					}
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
