package shared

import (
	"encoding/json"
	"fmt"

	"github.com/loda-lang/loda-api/util"
)

// Mode represents the type of submission operation
type Mode string

const (
	ModeAdd    Mode = "add"
	ModeUpdate Mode = "update"
	ModeRemove Mode = "remove"
)

// Type represents the type of object being submitted
type Type string

const (
	TypeProgram  Type = "program"
	TypeSequence Type = "sequence"
	TypeBFile    Type = "bfile"
)

// Submission represents a submission of a program or sequence
type Submission struct {
	Id        util.UID
	Mode      Mode
	Type      Type
	Content   string
	Submitter string
	// Additional fields for internal use (not serialized in JSON)
	Operations   []string // extracted operations for duplicate detection
	MinerProfile string   // miner profile for metrics
}

// MarshalJSON implements custom JSON serialization for Submission
func (s Submission) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id        string `json:"id"`
		Mode      string `json:"mode"`
		Type      string `json:"type"`
		Content   string `json:"content,omitempty"`
		Submitter string `json:"submitter,omitempty"`
	}{
		Id:        s.Id.String(),
		Mode:      string(s.Mode),
		Type:      string(s.Type),
		Content:   s.Content,
		Submitter: s.Submitter,
	})
}

// UnmarshalJSON implements custom JSON deserialization for Submission
func (s *Submission) UnmarshalJSON(data []byte) error {
	var aux struct {
		Id        string `json:"id"`
		Mode      string `json:"mode"`
		Type      string `json:"type"`
		Content   string `json:"content,omitempty"`
		Submitter string `json:"submitter,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	uid, err := util.NewUIDFromString(aux.Id)
	if err != nil {
		return err
	}
	// Validate mode (map 'delete' to 'remove' for backward compatibility)
	mode := Mode(aux.Mode)
	if mode == "delete" {
		mode = ModeRemove
	}
	if mode != ModeAdd && mode != ModeUpdate && mode != ModeRemove {
		return fmt.Errorf("invalid mode: %s", aux.Mode)
	}
	// Validate type
	objType := Type(aux.Type)
	if objType != TypeProgram && objType != TypeSequence && objType != TypeBFile {
		return fmt.Errorf("invalid type: %s", aux.Type)
	}
	// Validate mode for bfile type (only remove allowed)
	if objType == TypeBFile && mode != ModeRemove {
		return fmt.Errorf("only remove mode is allowed for bfile type")
	}
	s.Id = uid
	s.Mode = mode
	s.Type = objType
	s.Content = aux.Content
	s.Submitter = aux.Submitter
	// Extract operations and miner profile from programs for internal use
	if s.Type == TypeProgram {
		s.Operations = extractOperations(aux.Content)
		s.MinerProfile = extractMinerProfile(aux.Content)
	} else {
		s.Operations = nil
		s.MinerProfile = ""
	}
	return nil
}

// SubmissionsResult represents a paginated list of submissions
type SubmissionsResult struct {
	Session int64        `json:"session"`
	Total   int          `json:"total"`
	Results []Submission `json:"results"`
}

// NewSubmissionFromProgram creates a Submission from a Program (for v1 API compatibility)
func NewSubmissionFromProgram(program Program) Submission {
	submitter := ""
	if program.Submitter != nil {
		submitter = program.Submitter.Name
	}
	// Extract change type to determine mode
	changeType := extractChangeType(program.Code)
	mode := ModeAdd // default to "add"
	if changeType == "Found" {
		mode = ModeAdd
	} else if changeType != "" {
		// Any other non-empty change type means "update"
		mode = ModeUpdate
	}
	return Submission{
		Id:           program.Id,
		Submitter:    submitter,
		Content:      program.Code,
		Mode:         mode,
		Type:         TypeProgram,
		Operations:   program.Operations,
		MinerProfile: program.GetMinerProfile(),
	}
}

// NewSubmissionFromCode creates a Submission from program code (for v1 API compatibility)
func NewSubmissionFromCode(code string) (Submission, error) {
	program, err := NewProgramFromCode(code)
	if err != nil {
		return Submission{}, err
	}
	return NewSubmissionFromProgram(program), nil
}
