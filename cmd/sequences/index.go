package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/loda-lang/loda-api/shared"
	"github.com/loda-lang/loda-api/util"
)

type Index struct {
	Sequences []Sequence
}

func NewIndex() *Index {
	return &Index{}
}

// Load reads and parses the "names", "keywords" and "stripped" files to populate the Sequences index.
func (idx *Index) Load(dataDir string) error {
	namesPath := filepath.Join(dataDir, "names")
	nameMap, err := loadNamesFile(namesPath)
	if err != nil {
		return err
	}
	keywordsPath := filepath.Join(dataDir, "keywords")
	keywordsMap, err := loadKeywordsFile(keywordsPath)
	if err != nil {
		return err
	}
	strippedPath := filepath.Join(dataDir, "stripped")
	sequences, err := loadStrippedFile(strippedPath, nameMap)
	if err != nil {
		return err
	}
	// Attach keywords to sequences
	for i := range sequences {
		id := sequences[i].Id.String()
		if keywords, ok := keywordsMap[id]; ok {
			encoded, err := shared.EncodeKeywords(keywords)
			if err != nil {
				return err
			}
			sequences[i].Keywords = encoded
		}
	}
	// Sort sequences by ID
	sort.Slice(sequences, func(i, j int) bool {
		return sequences[i].Id.IsLessThan(sequences[j].Id)
	})
	idx.Sequences = sequences
	return nil
}

func loadNamesFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open names file: %w", err)
	}
	defer file.Close()
	nameMap := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			id := strings.TrimSpace(parts[0])
			name := strings.TrimSpace(parts[1])
			nameMap[id] = name
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read names file: %w", err)
	}
	return nameMap, nil
}

func loadKeywordsFile(path string) (map[string][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open keywords file: %w", err)
	}
	defer file.Close()
	keywordsMap := make(map[string][]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			id := strings.TrimSpace(parts[0])
			keywords := strings.Split(parts[1], ",")
			var trimmed []string
			for _, k := range keywords {
				k = strings.TrimSpace(k)
				if k == "" {
					continue
				}
				_, err := shared.EncodeKeywords([]string{k})
				if err != nil {
					continue
				}
				trimmed = append(trimmed, k)
			}
			sort.Strings(trimmed)
			keywordsMap[id] = trimmed
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read keywords file: %w", err)
	}
	return keywordsMap, nil
}

func loadStrippedFile(path string, nameMap map[string]string) ([]Sequence, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open stripped file: %w", err)
	}
	defer file.Close()
	var sequences []Sequence
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			id := strings.TrimSpace(parts[0])
			uid, err := util.NewUIDFromString(id)
			if err != nil {
				return nil, fmt.Errorf("invalid UID %q in stripped file: %w", id, err)
			}
			terms := strings.TrimSpace(parts[1])
			name := nameMap[id]
			sequences = append(sequences, Sequence{
				Id:    uid,
				Name:  name,
				Terms: terms,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read stripped file: %w", err)
	}
	return sequences, nil
}

func (idx *Index) FindById(id util.UID) *Sequence {
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

// Search finds sequences matching the query and applies pagination.
func (idx *Index) Search(query string, limit, skip int) []Sequence {
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
		if shared.IsKeyword(t) {
			inc = append(inc, t)
		} else if len(t) > 1 && t[0] == '+' && shared.IsKeyword(t[1:]) {
			inc = append(inc, t[1:])
		} else if len(t) > 1 && (t[0] == '-' || t[0] == '!') && shared.IsKeyword(t[1:]) {
			exc = append(exc, t[1:])
		} else {
			filteredTokens = append(filteredTokens, t)
		}
	}
	included, err := shared.EncodeKeywords(inc)
	if err != nil {
		return nil
	}
	excluded, err := shared.EncodeKeywords(exc)
	if err != nil {
		return nil
	}

	count := 0
	var results []Sequence
	for _, seq := range idx.Sequences {
		// Check included and excluded keywords
		if !shared.ContainsAllKeywords(seq.Keywords, included) {
			continue
		}
		if !shared.ContainsNoKeywords(seq.Keywords, excluded) {
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
