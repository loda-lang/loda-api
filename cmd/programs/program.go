package main

import (
	"encoding/json"
	"strings"

	"github.com/loda-lang/loda-api/util"
)

// Program represents a LODA program according to the OpenAPI spec.
type Program struct {
	Id         util.UID // ID of the LODA program (e.g. A000045)
	Name       string   // Name of the integer sequence or program
	Code       string   // LODA program code in plain text format
	Submitter  string   // Name of the submitter (nullable)
	Keywords   []string // Keywords for this program
	Operations []string // Operations of the LODA program as an array of strings
}

// NewProgramFromText creates a Program instance from LODA code in plain text format.
// It expects the code as input, and extracts metadata from comments as in .asm files.
func NewProgramFromText(code string) Program {
	var id util.UID
	var name string
	var submitter string
	var operations []string
	lines := strings.Split(code, "\n")
	numComments := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, ";") {
			if numComments == 0 {
				parts := strings.SplitN(line[1:], ":", 2)
				if len(parts) == 2 {
					idStr := strings.TrimSpace(parts[0])
					uid, err := util.NewUIDFromString(idStr)
					if err == nil {
						id = uid
					}
					name = strings.TrimSpace(parts[1])
				}
				numComments++
			} else if strings.HasPrefix(line, "; Submitted by ") {
				submitter = strings.TrimSpace(strings.TrimPrefix(line, "; Submitted by "))
			} else {
				numComments++
			}
		} else if len(line) > 0 {
			// Skip directives like #offset
			if strings.HasPrefix(line, "#") {
				continue
			}
			operations = append(operations, line)
		}
	}
	return Program{
		Id:         id,
		Name:       name,
		Code:       code,
		Submitter:  submitter,
		Operations: operations,
	}
}

// MarshalJSON implements custom JSON serialization for Program according to the OpenAPI spec.
func (p Program) MarshalJSON() ([]byte, error) {
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
		Submitter:  p.Submitter,
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
	p.Id = uid
	p.Name = aux.Name
	p.Code = aux.Code
	p.Submitter = aux.Submitter
	p.Keywords = aux.Keywords
	p.Operations = aux.Operations
	return nil
}
