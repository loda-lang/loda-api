package shared

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/loda-lang/loda-api/util"
)

type Program struct {
	Id         util.UID
	Name       string
	Code       string
	Submitter  *Submitter
	Keywords   uint64 // bitmask of keywords
	OpsMask    uint64 // bitmask of operation types
	Operations []string
	Formula    string
	Length     int
	Usages     string // space-separated program IDs
}

// ProgramFromText creates a Program instance from LODA code in plain text format.
// It expects the code as input, and extracts metadata from comments as in .asm files.
func NewProgramFromCode(code string) (Program, error) {
	id, name := extractIdAndName(code)
	submitter := extractSubmitter(code)
	operations := extractOperations(code)
	formula := extractFormula(code)
	return Program{
		Id:         id,
		Name:       name,
		Code:       code,
		Submitter:  submitter,
		OpsMask:    0, // OpsMask is computed separately when OpTypeIndex is available
		Operations: operations,
		Formula:    formula,
		Length:     len(operations),
	}, nil
}

// MarshalJSON implements custom JSON serialization for Program according to the OpenAPI spec.
func (p Program) MarshalJSON() ([]byte, error) {
	submitter := ""
	if p.Submitter != nil {
		submitter = p.Submitter.Name
	}
	usages := []string{}
	if strings.TrimSpace(p.Usages) != "" {
		usages = strings.Fields(p.Usages)
	}
	return json.Marshal(struct {
		Id         string   `json:"id"`
		Name       string   `json:"name"`
		Code       string   `json:"code"`
		Submitter  string   `json:"submitter,omitempty"`
		Keywords   []string `json:"keywords"`
		Operations []string `json:"operations"`
		Formula    string   `json:"formula,omitempty"`
		Usages     []string `json:"usages"`
	}{
		Id:         p.Id.String(),
		Name:       p.Name,
		Code:       p.Code,
		Submitter:  submitter,
		Keywords:   DecodeKeywords(p.Keywords),
		Operations: p.Operations,
		Formula:    p.Formula,
		Usages:     usages,
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
		Usages     []string `json:"usages"`
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
	p.OpsMask = 0 // OpsMask is not persisted in JSON, only used internally
	p.Usages = strings.Join(aux.Usages, " ")
	return nil
}

func (p *Program) SetOffset(offset int) {
	header := []string{}
	rest := []string{}
	found := false
	isHeader := true
	for _, line := range strings.Split(p.Code, "\n") {
		line = strings.TrimSpace(line)
		if len(line) > 0 && line[0] != ';' {
			isHeader = false
		}
		if strings.HasPrefix(line, "#offset ") {
			line = "#offset " + strconv.Itoa(offset)
			found = true
		}
		if isHeader {
			header = append(header, line)
		} else {
			rest = append(rest, line)
		}
	}
	if !found {
		header = append(header, "#offset "+strconv.Itoa(offset))
	}
	lines := append(header, rest...)
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
	p.OpsMask = 0 // OpsMask is computed separately when OpTypeIndex is available
	p.Operations = extractOperations(code)
	p.Formula = extractFormula(code)
	p.Length = len(p.Operations)
	return nil
}

func (p *Program) SetIdAndName(id util.UID, name string) {
	p.Code = updateIdAndName(p.Code, id, name)
	p.Id = id
	p.Name = name
}

func (p *Program) SetSubmitter(submitter *Submitter) {
	p.Code = updateSubmitter(p.Code, submitter)
	p.Submitter = submitter
}

func (p *Program) GetMinerProfile() string {
	return extractMinerProfile(p.Code)
}

func (p *Program) GetPath(programsDir string) (string, error) {
	if p.Id.Domain() == 'A' {
		idStr := p.Id.String()
		return filepath.Join(programsDir, idStr[1:4], idStr+".asm"), nil
	}
	return "", fmt.Errorf("unsupport domain: %c", p.Id.Domain())
}
