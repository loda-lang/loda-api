package shared

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
)

type Submitter struct {
	Name        string
	RefId       int
	NumPrograms int
}

var expectedSubmitterHeader = []string{"submitter", "ref_id", "num_programs"}

// Loads submitters.csv file and returns a slice of Submitter structs indexed by RefId.
func LoadSubmitters(path string) ([]*Submitter, error) {
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
