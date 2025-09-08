package shared

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/loda-lang/loda-api/util"
)

type Program struct {
	Id         util.UID
	Name       string
	Code       string
	Submitter  *Submitter
	Keywords   uint64 // Bitmask of keywords
	Operations []string
	Length     int
	Usages     int
	IncEval    bool
	LogEval    bool
}

// ProgramFromText creates a Program instance from LODA code in plain text format.
// It expects the code as input, and extracts metadata from comments as in .asm files.
func NewProgramFromCode(code string) (Program, error) {
	id, name := extractIdAndName(code)
	submitter := extractSubmitter(code)
	operations := extractOperations(code)
	return Program{
		Id:         id,
		Name:       name,
		Code:       code,
		Submitter:  submitter,
		Operations: operations,
	}, nil
}

var expectedHeader = []string{"id", "submitter", "length", "usages", "inc_eval", "log_eval"}

// LoadProgramsCSV parses the programs.csv file and returns a slice of Program structs.
// It also takes a slice of sequences, and for each program, if a matching sequence is found by ID, sets the program's name and keywords accordingly.
func LoadProgramsCSV(path string, submitters []*Submitter, index *Index) ([]Program, error) {
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
			continue
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

		// Find matching sequence by ID
		var name string
		var keywords uint64
		seq := index.FindById(uid)
		if seq != nil {
			name = seq.Name
			keywords = seq.Keywords
		}

		p := Program{
			Id:        uid,
			Name:      name,
			Keywords:  keywords,
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
		Keywords:   DecodeKeywords(p.Keywords),
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
	keywords, err := EncodeKeywords(aux.Keywords)
	if err != nil {
		return err
	}
	submitter := &Submitter{Name: aux.Submitter}
	p.Id = uid
	p.Name = aux.Name
	p.Code = aux.Code
	p.Submitter = submitter
	p.Keywords = keywords
	p.Operations = aux.Operations
	return nil
}

func (p *Program) SetOffset(offset int) {
	lines := []string{}
	found := false
	for _, line := range strings.Split(p.Code, "\n") {
		if len(line) > 8 && line[:8] == "#offset " {
			lines = append(lines, "#offset "+strconv.Itoa(offset))
			found = true
		} else {
			lines = append(lines, line)
		}
	}
	if !found {
		lines = append([]string{"#offset " + strconv.Itoa(offset)}, lines...)
	}
	p.Code = strings.Join(lines, "\n")
}

func (p *Program) SetCode(code string) error {
	id, name := extractIdAndName(code)
	submitter := extractSubmitter(code)
	if !id.IsZero() {
		p.Id = id
	}
	if name != "" {
		p.Name = name
	}
	if submitter != nil {
		p.Submitter = submitter
	}
	p.Code = code
	p.Operations = extractOperations(code)
	return nil
}

func (p *Program) GetPath(programsDir string) (string, error) {
	if p.Id.Domain() == 'A' {
		idStr := p.Id.String()
		return filepath.Join(programsDir, idStr[1:3], idStr+".asm"), nil
	}
	return "", fmt.Errorf("unsupport domain: %c", p.Id.Domain())
}
