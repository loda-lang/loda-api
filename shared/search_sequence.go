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
	sq := ParseSearchQuery(query)
	count := 0
	var results []Sequence
	var total int
	for _, seq := range idx.Sequences {
		// Check included and excluded keywords
		if !HasAllKeywords(seq.Keywords, sq.IncludedKeywords) {
			continue
		}
		if !HasNoKeywords(seq.Keywords, sq.ExcludedKeywords) {
			continue
		}
		match := true
		// Query string filtering (case-insensitive, all tokens must be present in name, submitter, or ID)
		if len(sq.FilteredTokens) > 0 || len(sq.UIDTokens) > 0 {
			nameLower := strings.ToLower(seq.Name)
			submitterLower := ""
			if seq.Submitter != nil {
				submitterLower = strings.ToLower(seq.Submitter.Name)
			}
			// Build author names lowercased
			var authorLowers []string
			for _, a := range seq.Authors {
				authorLowers = append(authorLowers, strings.ToLower(a.Name))
			}
			// Check UID tokens: match if the sequence ID equals the UID or the UID string is contained in the name
			for _, uid := range sq.UIDTokens {
				if !seq.Id.Equals(uid) && !strings.Contains(seq.Name, uid.String()) {
					match = false
					break
				}
			}
			// Check string tokens
			if match && len(sq.FilteredTokens) > 0 {
				for _, t := range sq.FilteredTokens {
					found := false
					if strings.Contains(nameLower, t) {
						found = true
					} else if submitterLower != "" && strings.Contains(submitterLower, t) {
						found = true
					} else {
						for _, author := range authorLowers {
							if strings.Contains(author, t) {
								found = true
								break
							}
						}
					}
					if !found {
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
		results = append(results, seq)
	}
	return results, total
}
