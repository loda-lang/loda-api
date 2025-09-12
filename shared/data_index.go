package shared

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/loda-lang/loda-api/util"
)

type DataIndex struct {
	DataDir    string
	OeisDir    string
	StatsDir   string
	Programs   []Program
	Sequences  []Sequence
	Submitters []*Submitter
}

func NewDataIndex(dataDir string) *DataIndex {
	oeisDir := filepath.Join(dataDir, "seqs", "oeis")
	statsDir := filepath.Join(dataDir, "stats")
	return &DataIndex{
		DataDir:  dataDir,
		StatsDir: statsDir,
		OeisDir:  oeisDir,
	}
}

// Load reads and parses the data files to populate the index.
func (idx *DataIndex) Load() error {
	namesPath := filepath.Join(idx.OeisDir, "names")
	nameMap, err := LoadNamesFile(namesPath)
	if err != nil {
		return err
	}
	keywordsPath := filepath.Join(idx.OeisDir, "keywords")
	keywordsMap, err := LoadKeywordsFile(keywordsPath)
	if err != nil {
		return err
	}
	strippedPath := filepath.Join(idx.OeisDir, "stripped")
	sequences, err := LoadStrippedFile(strippedPath, nameMap)
	if err != nil {
		return err
	}
	commentsPath := filepath.Join(idx.OeisDir, "comments")
	comments, err := LoadOeisTextFile(commentsPath)
	if err != nil {
		return err
	}
	formulasPath := filepath.Join(idx.OeisDir, "formulas")
	formulas, err := LoadOeisTextFile(formulasPath)
	if err != nil {
		return err
	}
	submittersPath := filepath.Join(idx.StatsDir, "submitters.csv")
	submitters, err := LoadSubmittersCSV(submittersPath)
	if err != nil {
		return err
	}
	programsPath := filepath.Join(idx.StatsDir, "programs.csv")
	programs, err := LoadProgramsCSV(programsPath, submitters)
	if err != nil {
		return err
	}

	// Sort sequences and programs by ID
	sort.Slice(sequences, func(i, j int) bool {
		return sequences[i].Id.IsLessThan(sequences[j].Id)
	})
	sort.Slice(programs, func(i, j int) bool {
		return programs[i].Id.IsLessThan(programs[j].Id)
	})

	// Update sequences and programs with keywords and names, including extra keywords from comments/formulas
	// Both lists are sorted by ID, so we can do a linear scan
	si, pi := 0, 0
	for si < len(sequences) {
		id := sequences[si].Id
		idStr := id.String()
		var keywords uint64
		if keywordsStr, ok := keywordsMap[idStr]; ok {
			k, err := EncodeKeywords(keywordsStr)
			if err != nil {
				return err
			}
			keywords = k
		}

		// --- Begin: extra keyword extraction logic ---
		var extraKeywords []string
		// Check for formulas
		if f, ok := formulas[idStr]; ok && f != "" {
			extraKeywords = append(extraKeywords, "formula")
		}
		// Combine name and comments for keyword extraction
		desc := strings.ToLower(sequences[si].Name)
		if c, ok := comments[idStr]; ok && c != "" {
			desc += " " + strings.ToLower(c)
		}
		// Keyword heuristics
		if strings.Contains(desc, "conjecture") || strings.Contains(desc, "it appears") || strings.Contains(desc, "empirical") {
			extraKeywords = append(extraKeywords, "conjecture")
		}
		if strings.Contains(desc, "decimal expansion") {
			extraKeywords = append(extraKeywords, "decimal-expansion")
		}
		if strings.Contains(desc, " e.g.f.") {
			extraKeywords = append(extraKeywords, "egf-expansion")
		}
		if strings.Contains(desc, " g.f.") {
			extraKeywords = append(extraKeywords, "gf-expansion")
		}
		if len(extraKeywords) > 0 {
			bits, _ := EncodeKeywords(extraKeywords)
			keywords |= bits
		}
		// --- End: extra keyword extraction logic ---

		// If a program with the same ID exists, update it as well
		for pi < len(programs) && programs[pi].Id.IsLessThan(id) {
			pi++
		}
		if pi < len(programs) && programs[pi].Id == id {
			keywords |= programs[pi].Keywords
			programs[pi].Keywords = keywords
			programs[pi].Name = sequences[si].Name
			pi++
		}
		// Update sequence keywords
		sequences[si].Keywords = keywords
		si++
	}

	idx.Submitters = submitters
	idx.Programs = programs
	idx.Sequences = sequences
	log.Printf("Loaded %d sequences, %d programs, %d submitters",
		len(sequences), len(programs), len(submitters))
	return nil
}

// LoadNamesFile reads the OEIS names file and returns a map from UID string to name.
func LoadNamesFile(path string) (map[string]string, error) {
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

// LoadKeywordsFile reads the OEIS keywords file and returns a map from UID string to list of keywords.
func LoadKeywordsFile(path string) (map[string][]string, error) {
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
				_, err := EncodeKeywords([]string{k})
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

// LoadStrippedFile reads the OEIS stripped file and returns a list of sequences.
// It uses the provided nameMap to set sequence names.
func LoadStrippedFile(path string, nameMap map[string]string) ([]Sequence, error) {
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

// LoadOeisTextFile reads an OEIS-style text file (e.g., comments, formulas) and returns a map from UID string to concatenated lines.
func LoadOeisTextFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open OEIS text file: %w", err)
	}
	defer file.Close()
	entryMap := make(map[string][]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			id := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			entryMap[id] = append(entryMap[id], value)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read OEIS text file: %w", err)
	}
	// Concatenate lines for each UID
	result := make(map[string]string, len(entryMap))
	for id, lines := range entryMap {
		result[id] = strings.Join(lines, "\n")
	}
	return result, nil
}

var expectedSubmitterHeader = []string{"submitter", "ref_id", "num_programs"}

func LoadSubmittersCSV(path string) ([]*Submitter, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	if !slices.Equal(header, expectedSubmitterHeader) {
		return nil, fmt.Errorf("unexpected header in submitters.csv: %v", header)
	}
	var records [][]string
	maxRefId := 0
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
		refId, err := strconv.Atoi(rec[1])
		if err != nil {
			return nil, err
		}
		if refId > maxRefId {
			maxRefId = refId
		}
	}
	submitters := make([]*Submitter, maxRefId+1)
	for _, rec := range records {
		name := rec[0]
		refId, err := strconv.Atoi(rec[1])
		if err != nil {
			return nil, err
		}
		numPrograms, err := strconv.Atoi(rec[2])
		if err != nil {
			return nil, err
		}
		submitters[refId] = &Submitter{
			Name:        name,
			RefId:       refId,
			NumPrograms: numPrograms,
		}
	}
	return submitters, nil
}

var expectedProgramsHeader = []string{"id", "submitter", "length", "usages", "inc_eval", "log_eval"}

func LoadProgramsCSV(path string, submitters []*Submitter) ([]Program, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	if !slices.Equal(header, expectedProgramsHeader) {
		return nil, err
	}
	var programs []Program
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(rec) != 6 {
			return nil, fmt.Errorf("unexpected number of fields: %v", rec)
		}
		uid, err := util.NewUIDFromString(rec[0])
		if err != nil {
			return nil, err
		}
		var submitter *Submitter = nil
		if refId, err := strconv.Atoi(rec[1]); err == nil {
			if refId >= 0 && refId < len(submitters) {
				submitter = submitters[refId]
			}
		}
		length, err := strconv.Atoi(rec[2])
		if err != nil {
			return nil, err
		}
		usages, err := strconv.Atoi(rec[3])
		if err != nil {
			return nil, err
		}
		incEval := rec[4] == "1"
		logEval := rec[5] == "1"

		// Add loda-specific keywords
		bit, _ := EncodeKeywords([]string{"loda"})
		keywords := bit
		if incEval {
			bit, _ = EncodeKeywords([]string{"loda-inceval"})
			keywords |= bit
		}
		if logEval {
			bit, _ = EncodeKeywords([]string{"loda-logeval"})
			keywords |= bit
		}
		p := Program{
			Id:        uid,
			Name:      "", // Will be filled in later from sequence name
			Keywords:  keywords,
			Submitter: submitter,
			Length:    length,
			Usages:    usages,
		}
		programs = append(programs, p)
	}
	return programs, nil
}
