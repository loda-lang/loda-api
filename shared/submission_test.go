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
		Id:             id,
		Submitter:      "alice",
		Content:        "mov $0,1",
		Mode: ModeAdd,
		Type:     TypeProgram,
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

func TestSubmissionsResult_JSON(t *testing.T) {
	id1, _ := util.NewUIDFromString("A000045")
	id2, _ := util.NewUIDFromString("A000142")

	result := SubmissionsResult{
		Total: 2,
		Results: []Submission{
			{
				Id:             id1,
				Submitter:      "alice",
				Content:        "mov $0,1",
				Mode: ModeAdd,
				Type:     TypeProgram,
			},
			{
				Id:             id2,
				Submitter:      "bob",
				Content:        "mul $0,2",
				Mode: ModeUpdate,
				Type:     TypeProgram,
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
