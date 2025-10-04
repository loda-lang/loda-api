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
	NumUsages  map[string]int
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
	// Extract extra keywords from names
	nameKeywords, err := ExtractKeywordsFromFile(namesPath, " ")
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

	// Efficiently extract extra keywords from comments, formulas, and names
	commentKeywords, err := ExtractKeywordsFromFile(commentsPath, ":")
	if err != nil {
		return err
	}
	formulasPath := filepath.Join(idx.OeisDir, "formulas")
	formulaKeywords, err := ExtractKeywordsFromFormulas(formulasPath)
	if err != nil {
		return err
	}
	oeisProgramsPath := filepath.Join(idx.OeisDir, "programs")
	idsWithPari, err := ExtractPariSeqs(oeisProgramsPath)
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
	callGraphPath := filepath.Join(idx.StatsDir, "call_graph.csv")
	programUsages, err := extractProgramUsages(callGraphPath)
	if err != nil {
		return err
	}

	// Compute program usage counts
	numUsages := make(map[string]int)
	for callee, callers := range programUsages {
		if callers == "" {
			numUsages[callee] = 0
		} else {
			numUsages[callee] = len(strings.Fields(callers))
		}
	}

	// Sort sequences and programs by ID
	sort.Slice(sequences, func(i, j int) bool {
		return sequences[i].Id.IsLessThan(sequences[j].Id)
	})
	sort.Slice(programs, func(i, j int) bool {
		return programs[i].Id.IsLessThan(programs[j].Id)
	})

	// Update sequences and programs with keywords, names, used program IDs, and submitter
	si, pi := 0, 0
	for si < len(sequences) {
		id := sequences[si].Id
		idStr := id.String()
		var keywords uint64
		var submitter *Submitter = nil
		if keywordsStr, ok := keywordsMap[idStr]; ok {
			k, err := EncodeKeywords(keywordsStr)
			if err != nil {
				return err
			}
			keywords = k
		}
		// Add extra keywords from formulas, comments, and names
		if bits, ok := formulaKeywords[idStr]; ok {
			keywords |= bits
		}
		if bits, ok := commentKeywords[idStr]; ok {
			keywords |= bits
		}
		if bits, ok := nameKeywords[idStr]; ok {
			keywords |= bits
		}
		if _, ok := idsWithPari[idStr]; ok {
			keywords |= KeywordPariBits
		}
		// If a program with the same ID exists, update it as well
		for pi < len(programs) && programs[pi].Id.IsLessThan(id) {
			pi++
		}
		if pi < len(programs) && programs[pi].Id == id {
			keywords |= programs[pi].Keywords
			programs[pi].Keywords = keywords
			programs[pi].Name = sequences[si].Name
			if usages, ok := programUsages[idStr]; ok {
				programs[pi].Usages = usages
			} else {
				programs[pi].Usages = ""
			}
			submitter = programs[pi].Submitter
			pi++
		}
		// Update sequence keywords and submitter
		sequences[si].Keywords = keywords
		sequences[si].Submitter = submitter
		si++
	}

	idx.Submitters = submitters
	idx.Programs = programs
	idx.Sequences = sequences
	idx.NumUsages = numUsages

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

// ExtractPariSeqs parses the OEIS programs file and returns a set of IDs with (PARI)
func ExtractPariSeqs(path string) (map[string]struct{}, error) {
	idsWithPari := make(map[string]struct{})
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		if strings.Contains(line, "(PARI)") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				id := strings.TrimSpace(parts[0])
				idsWithPari[id] = struct{}{}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return idsWithPari, nil
}

// extractKeywordBitsFromComment returns the encoded keyword bits for a single comment string.
func extractKeywordBitsFromComment(comment string) uint64 {
	var bits uint64
	if strings.Contains(comment, "conjecture") || strings.Contains(comment, "it appears") || strings.Contains(comment, "empirical") {
		bits |= KeywordConjectureBits
	}
	if strings.Contains(comment, "decimal expansion") {
		bits |= KeywordDecimalExpBits
	}
	if strings.Contains(comment, " e.g.f.") {
		bits |= KeywordEGFExpBits
	}
	if strings.Contains(comment, " g.f.") {
		bits |= KeywordGFExpBits
	}
	return bits
}

// ExtractKeywordsFromFile parses a file (comments or names) and returns a map from UID to encoded extra keywords.
// The separator argument should be ":" for comments and " " for names.
func ExtractKeywordsFromFile(path string, separator string) (map[string]uint64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	encoded := make(map[string]uint64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, separator, 2)
		if len(parts) == 2 {
			id := strings.TrimSpace(parts[0])
			text := strings.ToLower(strings.TrimSpace(parts[1]))
			bits := extractKeywordBitsFromComment(text)
			if bits != 0 {
				encoded[id] |= bits
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return encoded, nil
}

// ExtractKeywordsFromFormulas parses the formulas file once and returns a map from UID to encoded "formula" keyword.
func ExtractKeywordsFromFormulas(path string) (map[string]uint64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open formulas file: %w", err)
	}
	defer file.Close()
	result := make(map[string]uint64)
	formulaBits, _ := EncodeKeywords([]string{"formula"})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			id := strings.TrimSpace(parts[0])
			// If any entry exists, mark as having formula
			result[id] = result[id] | formulaBits
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read formulas file: %w", err)
	}
	return result, nil
}

// extractNumUsages parses a call_graph.csv file and returns a map from callee-to-caller IDs.
func extractProgramUsages(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open call_graph.csv: %w", err)
	}
	defer file.Close()
	usages := make(map[string][]string)
	r := csv.NewReader(file)
	// Read and check header
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}
	expectedHeader := []string{"caller", "callee"}
	if !slices.Equal(header, expectedHeader) {
		return nil, fmt.Errorf("unexpected header in call_graph file: %v", header)
	}
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read record: %w", err)
		}
		if len(rec) != 2 {
			continue
		}
		caller := strings.TrimSpace(rec[0])
		callee := strings.TrimSpace(rec[1])
		if callee != "" && caller != "" {
			usages[callee] = append(usages[callee], caller)
		}
	}
	// Convert to space-separated string
	result := make(map[string]string)
	for callee, callers := range usages {
		result[callee] = strings.Join(callers, " ")
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

var expectedProgramsHeader = []string{"id", "submitter", "length", "usages", "inc_eval", "log_eval", "vir_eval", "loop", "formula", "indirect"}

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
		return nil, fmt.Errorf("unexpected header in programs.csv: %v", header)
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
		if len(rec) != 10 {
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
		// usages, err := strconv.Atoi(rec[3])
		// if err != nil {
		//	return nil, err
		// }
		incEval := rec[4] == "1"
		logEval := rec[5] == "1"
		virevalFlag := rec[6] == "1"
		loopFlag := rec[7] == "1"
		formulaFlag := rec[8] == "1"
		indirectFlag := rec[9] == "1"

		// Add loda-specific keywords using constants
		keywords := KeywordLodaBits
		if incEval {
			keywords |= KeywordLodaIncevalBits
		}
		if logEval {
			keywords |= KeywordLodaLogevalBits
		}
		if virevalFlag {
			keywords |= KeywordLodaVirevalBits
		}
		if loopFlag {
			keywords |= KeywordLodaLoopBits
		}
		if formulaFlag {
			keywords |= KeywordLodaFormulaBits
		}
		if indirectFlag {
			keywords |= KeywordLodaIndirectBits
		}
		p := Program{
			Id:        uid,
			Name:      "", // Will be filled in later from sequence name
			Keywords:  keywords,
			Submitter: submitter,
			Length:    length,
		}
		programs = append(programs, p)
	}
	return programs, nil
}
