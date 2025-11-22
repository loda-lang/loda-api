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
	ModeDelete Mode = "delete"
)

// Type represents the type of object being submitted
type Type string

const (
	TypeProgram  Type = "program"
	TypeSequence Type = "sequence"
)

// Submission represents a submission of a program or sequence
type Submission struct {
	Id        util.UID
	Submitter string
	Content   string
	Mode      Mode
	Type      Type
	// Additional fields for internal use (not serialized in JSON)
	Operations   []string // extracted operations for duplicate detection
	MinerProfile string   // miner profile for metrics
}

// MarshalJSON implements custom JSON serialization for Submission
func (s Submission) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id        string `json:"id"`
		Submitter string `json:"submitter"`
		Content   string `json:"content"`
		Mode      string `json:"mode"`
		Type      string `json:"type"`
	}{
		Id:        s.Id.String(),
		Submitter: s.Submitter,
		Content:   s.Content,
		Mode:      string(s.Mode),
		Type:      string(s.Type),
	})
}

// UnmarshalJSON implements custom JSON deserialization for Submission
func (s *Submission) UnmarshalJSON(data []byte) error {
	var aux struct {
		Id        string `json:"id"`
		Submitter string `json:"submitter"`
		Content   string `json:"content"`
		Mode      string `json:"mode"`
		Type      string `json:"type"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	uid, err := util.NewUIDFromString(aux.Id)
	if err != nil {
		return err
	}
	// Validate mode
	mode := Mode(aux.Mode)
	if mode != ModeAdd && mode != ModeUpdate && mode != ModeDelete {
		return fmt.Errorf("invalid mode: %s", aux.Mode)
	}
	// Validate type
	objType := Type(aux.Type)
	if objType != TypeProgram && objType != TypeSequence {
		return fmt.Errorf("invalid type: %s", aux.Type)
	}
	s.Id = uid
	s.Submitter = aux.Submitter
	s.Content = aux.Content
	s.Mode = mode
	s.Type = objType
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
