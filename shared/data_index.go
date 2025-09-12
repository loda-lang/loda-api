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

	// Merge keywords and attach them to sequences and programs
	for i := range sequences {
		id := sequences[i].Id
		p := FindProgramById(programs, id)
		if keywordsStr, ok := keywordsMap[id.String()]; ok {
			keywords, err := EncodeKeywords(keywordsStr)
			if err != nil {
				return err
			}
			if p != nil {
				keywords = MergeKeywords(keywords, p.Keywords)
				p.Keywords = keywords
				p.Name = sequences[i].Name // Update program name from sequence
			}
			sequences[i].Keywords = keywords
		}
	}

	// Sort sequences and programs by ID
	sort.Slice(sequences, func(i, j int) bool {
		return sequences[i].Id.IsLessThan(sequences[j].Id)
	})
	sort.Slice(programs, func(i, j int) bool {
		return programs[i].Id.IsLessThan(programs[j].Id)
	})

	idx.Submitters = submitters
	idx.Programs = programs
	idx.Sequences = sequences
	log.Printf("Loaded %d sequences, %d programs, %d submitters",
		len(sequences), len(programs), len(submitters))
	return nil
}

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
