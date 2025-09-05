package main

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"slices"
	"strconv"

	"github.com/loda-lang/loda-api/shared"
	"github.com/loda-lang/loda-api/util"
)

type Program struct {
	Id         util.UID
	Name       string
	Code       string
	Submitter  *shared.Submitter
	Keywords   []string
	Operations []string
	Length     int
	Usages     int
	IncEval    bool
	LogEval    bool
}

// ProgramFromText creates a Program instance from LODA code in plain text format.
// It expects the code as input, and extracts metadata from comments as in .asm files.
func NewProgramFromText(code string) Program {
	id, name := extractIdAndName(code)
	submitter := extractSubmitter(code)
	operations := extractOperations(code)
	return Program{
		Id:         id,
		Name:       name,
		Code:       code,
		Submitter:  submitter,
		Operations: operations,
	}
}

var expectedHeader = []string{"id", "submitter", "length", "usages", "inc_eval", "log_eval"}

// LoadProgramsCSV parses the programs.csv file and returns a slice of Program structs.
func LoadProgramsCSV(path string, submitters []*shared.Submitter) ([]*Program, error) {
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
	if !slices.Equal(header, expectedHeader) {
		return nil, err
	}
	var programs []*Program
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(rec) != 6 {
			continue
		}
		uid, err := util.NewUIDFromString(rec[0])
		if err != nil {
			return nil, err
		}
		var submitter *shared.Submitter = nil
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
		p := &Program{
			Id:        uid,
			Submitter: submitter,
			Length:    length,
			Usages:    usages,
			IncEval:   incEval,
			LogEval:   logEval,
		}
		programs = append(programs, p)
	}
	return programs, nil
}

// MarshalJSON implements custom JSON serialization for Program according to the OpenAPI spec.
func (p Program) MarshalJSON() ([]byte, error) {
	submitter := ""
	if p.Submitter != nil {
		submitter = p.Submitter.Name
	}
	return json.Marshal(struct {
		Id         string   `json:"id"`
		Name       string   `json:"name"`
		Code       string   `json:"code"`
		Submitter  string   `json:"submitter,omitempty"`
		Keywords   []string `json:"keywords"`
		Operations []string `json:"operations"`
	}{
		Id:         p.Id.String(),
		Name:       p.Name,
		Code:       p.Code,
		Submitter:  submitter,
		Keywords:   p.Keywords,
		Operations: p.Operations,
	})
}

// UnmarshalJSON implements custom JSON deserialization for Program according to the OpenAPI spec.
func (p *Program) UnmarshalJSON(data []byte) error {
	var aux struct {
		Id         string   `json:"id"`
		Name       string   `json:"name"`
		Code       string   `json:"code"`
		Submitter  string   `json:"submitter"`
		Keywords   []string `json:"keywords"`
		Operations []string `json:"operations"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	uid, err := util.NewUIDFromString(aux.Id)
	if err != nil {
		return err
	}
	submitter := &shared.Submitter{Name: aux.Submitter}
	p.Id = uid
	p.Name = aux.Name
	p.Code = aux.Code
	p.Submitter = submitter
	p.Keywords = aux.Keywords
	p.Operations = aux.Operations
	return nil
}
