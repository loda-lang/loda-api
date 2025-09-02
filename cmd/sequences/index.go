package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
		if kws, ok := keywordsMap[sequences[i].Id]; ok {
			sequences[i].Keywords = kws
		}
	}
	// Sort sequences by ID
	sort.Slice(sequences, func(i, j int) bool {
		return sequences[i].Id < sequences[j].Id
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
				if k != "" {
					trimmed = append(trimmed, k)
				}
			}
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
			terms := strings.TrimSpace(parts[1])
			name := nameMap[id]
			sequences = append(sequences, Sequence{
				Id:    id,
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

func (idx *Index) FindById(id string) *Sequence {
	if len(id) < 2 {
		return nil
	}
	test := Sequence{Id: id}
	d := test.IdDomain()
	n := int64(test.IdNumber())
	if n >= 0 && n < int64(len(idx.Sequences)) && idx.Sequences[n].IdDomain() == d {
		k := idx.Sequences[n].IdNumber()
		if k == n {
			return &idx.Sequences[n]
		} else if k < n {
			// Search forward
			for i := n + 1; i < int64(len(idx.Sequences)); i++ {
				if idx.Sequences[i].IdDomain() != d {
					break
				}
				if idx.Sequences[i].IdNumber() == n {
					return &idx.Sequences[i]
				}
			}
		} else {
			// Search backward
			for i := n - 1; i >= 0; i-- {
				if idx.Sequences[i].IdDomain() != d {
					break
				}
				if idx.Sequences[i].IdNumber() == n {
					return &idx.Sequences[i]
				}
			}
		}
	} else {
		// Full search
		for _, s := range idx.Sequences {
			if s.IdDomain() == d && s.IdNumber() == n {
				return &s
			}
		}
	}
	return nil
}
