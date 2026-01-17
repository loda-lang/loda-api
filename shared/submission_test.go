package shared

import (
	"encoding/json"
	"testing"

	"github.com/loda-lang/loda-api/util"
	"github.com/stretchr/testify/assert"
)

func TestSubmission_MarshalJSON(t *testing.T) {
	id, _ := util.NewUIDFromString("A000045")
	sub := Submission{
		Id:        id,
		Submitter: "alice",
		Content:   "mov $0,1",
		Mode:      ModeAdd,
		Type:      TypeProgram,
	}

	data, err := json.Marshal(sub)
	assert.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, "A000045", result["id"])
	assert.Equal(t, "alice", result["submitter"])
	assert.Equal(t, "mov $0,1", result["content"])
	assert.Equal(t, "add", result["mode"])
	assert.Equal(t, "program", result["type"])
}

func TestSubmission_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "A000045",
		"submitter": "bob",
		"content": "add $0,2",
		"mode": "update",
		"type": "program"
	}`

	var sub Submission
	err := json.Unmarshal([]byte(jsonData), &sub)
	assert.NoError(t, err)
	assert.Equal(t, "A000045", sub.Id.String())
	assert.Equal(t, "bob", sub.Submitter)
	assert.Equal(t, "add $0,2", sub.Content)
	assert.Equal(t, ModeUpdate, sub.Mode)
	assert.Equal(t, TypeProgram, sub.Type)
}

func TestSubmission_UnmarshalJSON_InvalidMode(t *testing.T) {
	jsonData := `{
		"id": "A000045",
		"submitter": "bob",
		"content": "add $0,2",
		"mode": "invalid",
		"type": "program"
	}`

	var sub Submission
	err := json.Unmarshal([]byte(jsonData), &sub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mode")
}

func TestSubmission_UnmarshalJSON_InvalidType(t *testing.T) {
	jsonData := `{
		"id": "A000045",
		"submitter": "bob",
		"content": "add $0,2",
		"mode": "add",
		"type": "invalid"
	}`

	var sub Submission
	err := json.Unmarshal([]byte(jsonData), &sub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

func TestSubmission_UnmarshalJSON_InvalidId(t *testing.T) {
	jsonData := `{
		"id": "invalid",
		"submitter": "bob",
		"content": "add $0,2",
		"mode": "add",
		"type": "program"
	}`

	var sub Submission
	err := json.Unmarshal([]byte(jsonData), &sub)
	assert.Error(t, err)
}

func TestSubmission_UnmarshalJSON_BFileRemove(t *testing.T) {
	jsonData := `{
		"id": "A000045",
		"submitter": "bob",
		"mode": "remove",
		"type": "bfile"
	}`

	var sub Submission
	err := json.Unmarshal([]byte(jsonData), &sub)
	assert.NoError(t, err)
	assert.Equal(t, "A000045", sub.Id.String())
	assert.Equal(t, "bob", sub.Submitter)
	assert.Equal(t, ModeRemove, sub.Mode)
	assert.Equal(t, TypeBFile, sub.Type)
}

func TestSubmission_UnmarshalJSON_BFileAddNotAllowed(t *testing.T) {
	jsonData := `{
		"id": "A000045",
		"submitter": "bob",
		"mode": "add",
		"type": "bfile"
	}`

	var sub Submission
	err := json.Unmarshal([]byte(jsonData), &sub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only remove mode is allowed for bfile type")
}

func TestSubmission_UnmarshalJSON_BFileUpdateNotAllowed(t *testing.T) {
	jsonData := `{
		"id": "A000045",
		"submitter": "bob",
		"mode": "update",
		"type": "bfile"
	}`

	var sub Submission
	err := json.Unmarshal([]byte(jsonData), &sub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only remove mode is allowed for bfile type")
}

func TestSubmission_UnmarshalJSON_SequenceRefresh(t *testing.T) {
	jsonData := `{
		"id": "A000045",
		"submitter": "charlie",
		"mode": "refresh",
		"type": "sequence"
	}`

	var sub Submission
	err := json.Unmarshal([]byte(jsonData), &sub)
	assert.NoError(t, err)
	assert.Equal(t, "A000045", sub.Id.String())
	assert.Equal(t, "charlie", sub.Submitter)
	assert.Equal(t, ModeRefresh, sub.Mode)
	assert.Equal(t, TypeSequence, sub.Type)
}

func TestSubmission_UnmarshalJSON_SequenceAddNotAllowed(t *testing.T) {
	jsonData := `{
		"id": "A000045",
		"submitter": "charlie",
		"mode": "add",
		"type": "sequence"
	}`

	var sub Submission
	err := json.Unmarshal([]byte(jsonData), &sub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only refresh mode is allowed for sequence type")
}

func TestSubmission_UnmarshalJSON_SequenceUpdateNotAllowed(t *testing.T) {
	jsonData := `{
		"id": "A000045",
		"submitter": "charlie",
		"mode": "update",
		"type": "sequence"
	}`

	var sub Submission
	err := json.Unmarshal([]byte(jsonData), &sub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only refresh mode is allowed for sequence type")
}

func TestSubmission_UnmarshalJSON_SequenceRemoveNotAllowed(t *testing.T) {
	jsonData := `{
		"id": "A000045",
		"submitter": "charlie",
		"mode": "remove",
		"type": "sequence"
	}`

	var sub Submission
	err := json.Unmarshal([]byte(jsonData), &sub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only refresh mode is allowed for sequence type")
}

func TestSubmissionsResult_JSON(t *testing.T) {
	id1, _ := util.NewUIDFromString("A000045")
	id2, _ := util.NewUIDFromString("A000142")

	result := SubmissionsResult{
		Total: 2,
		Results: []Submission{
			{
				Id:        id1,
				Submitter: "alice",
				Content:   "mov $0,1",
				Mode:      ModeAdd,
				Type:      TypeProgram,
			},
			{
				Id:        id2,
				Submitter: "bob",
				Content:   "mul $0,2",
				Mode:      ModeUpdate,
				Type:      TypeProgram,
			},
		},
	}

	data, err := json.Marshal(result)
	assert.NoError(t, err)

	var unmarshaled SubmissionsResult
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, 2, unmarshaled.Total)
	assert.Equal(t, 2, len(unmarshaled.Results))
	assert.Equal(t, "A000045", unmarshaled.Results[0].Id.String())
	assert.Equal(t, "alice", unmarshaled.Results[0].Submitter)
}

func TestNewSubmissionFromProgram_ChangeTypeFound(t *testing.T) {
	code := `; A164177: Number of binary strings of length n.
; Submitted by bob
mov $2,1
lpb $0
  sub $0,1
lpe
mov $0,$2
; Miner Profile: batch-new
; Change Type: Found
`
	program, err := NewProgramFromCode(code)
	assert.NoError(t, err)

	sub := NewSubmissionFromProgram(program)
	assert.Equal(t, ModeAdd, sub.Mode)
	assert.Equal(t, "bob", sub.Submitter)
	assert.Equal(t, "batch-new", sub.MinerProfile)
}

func TestNewSubmissionFromProgram_ChangeTypeFaster(t *testing.T) {
	code := `; A000045: Fibonacci numbers
; Submitted by user123
mov $2,1
lpb $0
  sub $0,1
lpe
mov $0,$2
; Miner Profile: batch-optimize
; Change Type: Faster
`
	program, err := NewProgramFromCode(code)
	assert.NoError(t, err)

	sub := NewSubmissionFromProgram(program)
	assert.Equal(t, ModeUpdate, sub.Mode)
	assert.Equal(t, "user123", sub.Submitter)
	assert.Equal(t, "batch-optimize", sub.MinerProfile)
}

func TestNewSubmissionFromProgram_NoChangeType(t *testing.T) {
	code := `; A000045: Fibonacci numbers
; Submitted by user123
mov $2,1
lpb $0
  sub $0,1
lpe
mov $0,$2
`
	program, err := NewProgramFromCode(code)
	assert.NoError(t, err)

	sub := NewSubmissionFromProgram(program)
	assert.Equal(t, ModeAdd, sub.Mode) // defaults to add when no Change Type
	assert.Equal(t, "user123", sub.Submitter)
}
