package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"slices"
	"strconv"
)

type Summary struct {
	NumSequences int "json:\"numSequences\""
	NumPrograms  int "json:\"numPrograms\""
	NumFormulas  int "json:\"numFormulas\""
}

var expectedHeader = []string{"num_sequences", "num_programs", "num_formulas"}

// LoadSummaryCSV loads a summary from a CSV file with a header and one record.
func LoadSummaryCSV(path string) (*Summary, error) {
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
	record, err := r.Read()
	if err != nil {
		return nil, err
	}
	if !slices.Equal(header, expectedHeader) || len(record) != 3 {
		return nil, fmt.Errorf("unexpected CSV header or record: got %v, want %v", header, expectedHeader)
	}
	numSequences, err := strconv.Atoi(record[0])
	if err != nil {
		return nil, err
	}
	numPrograms, err := strconv.Atoi(record[1])
	if err != nil {
		return nil, err
	}
	numFormulas, err := strconv.Atoi(record[2])
	if err != nil {
		return nil, err
	}
	return &Summary{
		NumSequences: numSequences,
		NumPrograms:  numPrograms,
		NumFormulas:  numFormulas,
	}, nil
}
