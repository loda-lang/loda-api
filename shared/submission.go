package shared

import (
	"encoding/json"
	"fmt"

	"github.com/loda-lang/loda-api/util"
)

// SubmissionType represents the type of submission operation
type SubmissionType string

const (
	SubmissionTypeAdd    SubmissionType = "add"
	SubmissionTypeUpdate SubmissionType = "update"
	SubmissionTypeDelete SubmissionType = "delete"
)

// ObjectType represents the type of object being submitted
type ObjectType string

const (
	ObjectTypeProgram  ObjectType = "program"
	ObjectTypeSequence ObjectType = "sequence"
)

// Submission represents a submission of a program or sequence
type Submission struct {
	Id             util.UID
	Submitter      string
	Content        string
	SubmissionType SubmissionType
	ObjectType     ObjectType
	// Additional fields for internal use (not serialized in JSON)
	Operations   []string // extracted operations for duplicate detection
	MinerProfile string   // miner profile for metrics
}

// MarshalJSON implements custom JSON serialization for Submission
func (s Submission) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id             string `json:"id"`
		Submitter      string `json:"submitter"`
		Content        string `json:"content"`
		SubmissionType string `json:"submissionType"`
		ObjectType     string `json:"objectType"`
	}{
		Id:             s.Id.String(),
		Submitter:      s.Submitter,
		Content:        s.Content,
		SubmissionType: string(s.SubmissionType),
		ObjectType:     string(s.ObjectType),
	})
}

// UnmarshalJSON implements custom JSON deserialization for Submission
func (s *Submission) UnmarshalJSON(data []byte) error {
	var aux struct {
		Id             string `json:"id"`
		Submitter      string `json:"submitter"`
		Content        string `json:"content"`
		SubmissionType string `json:"submissionType"`
		ObjectType     string `json:"objectType"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	uid, err := util.NewUIDFromString(aux.Id)
	if err != nil {
		return err
	}
	// Validate submission type
	submissionType := SubmissionType(aux.SubmissionType)
	if submissionType != SubmissionTypeAdd && submissionType != SubmissionTypeUpdate && submissionType != SubmissionTypeDelete {
		return fmt.Errorf("invalid submission type: %s", aux.SubmissionType)
	}
	// Validate object type
	objectType := ObjectType(aux.ObjectType)
	if objectType != ObjectTypeProgram && objectType != ObjectTypeSequence {
		return fmt.Errorf("invalid object type: %s", aux.ObjectType)
	}
	s.Id = uid
	s.Submitter = aux.Submitter
	s.Content = aux.Content
	s.SubmissionType = submissionType
	s.ObjectType = objectType
	// Extract operations and miner profile for internal use
	s.Operations = extractOperations(aux.Content)
	s.MinerProfile = extractMinerProfile(aux.Content)
	return nil
}

// SubmissionsResult represents a paginated list of submissions
type SubmissionsResult struct {
	Total   int          `json:"total"`
	Results []Submission `json:"results"`
}

// NewSubmissionFromProgram creates a Submission from a Program (for v1 API compatibility)
func NewSubmissionFromProgram(program Program) Submission {
	submitter := ""
	if program.Submitter != nil {
		submitter = program.Submitter.Name
	}
	return Submission{
		Id:             program.Id,
		Submitter:      submitter,
		Content:        program.Code,
		SubmissionType: SubmissionTypeAdd, // v1 submissions are always "add"
		ObjectType:     ObjectTypeProgram,
		Operations:     program.Operations,
		MinerProfile:   program.GetMinerProfile(),
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
