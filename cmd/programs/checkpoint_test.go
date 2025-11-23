package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/loda-lang/loda-api/shared"
	"github.com/loda-lang/loda-api/util"
	"github.com/stretchr/testify/assert"
)

func TestCheckpoint_WriteAndLoad_JSON(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "checkpoint-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test server
	server := NewProgramsServer(tmpDir, nil, nil)

	// Add some test submissions
	id1, _ := util.NewUIDFromString("A000045")
	id2, _ := util.NewUIDFromString("A000142")

	server.submissions = []shared.Submission{
		{
			Id:        id1,
			Mode:      shared.ModeAdd,
			Type:      shared.TypeProgram,
			Content:   "mov $0,1\nadd $0,2\n",
			Submitter: "alice",
		},
		{
			Id:        id2,
			Mode:      shared.ModeUpdate,
			Type:      shared.TypeProgram,
			Content:   "mul $0,3\nsub $0,1\n",
			Submitter: "bob",
		},
	}

	// Write checkpoint
	err = server.writeCheckpoint()
	assert.NoError(t, err)

	// Verify the checkpoint file exists
	checkpointPath := filepath.Join(tmpDir, CheckpointFile)
	_, err = os.Stat(checkpointPath)
	assert.NoError(t, err)

	// Create a new server and load the checkpoint
	server2 := NewProgramsServer(tmpDir, nil, nil)
	server2.loadCheckpoint()

	// Verify the loaded submissions match
	assert.Equal(t, len(server.submissions), len(server2.submissions))
	assert.Equal(t, "A000045", server2.submissions[0].Id.String())
	assert.Equal(t, shared.ModeAdd, server2.submissions[0].Mode)
	assert.Equal(t, shared.TypeProgram, server2.submissions[0].Type)
	assert.Equal(t, "alice", server2.submissions[0].Submitter)
	assert.Equal(t, "mov $0,1\nadd $0,2\n", server2.submissions[0].Content)

	assert.Equal(t, "A000142", server2.submissions[1].Id.String())
	assert.Equal(t, shared.ModeUpdate, server2.submissions[1].Mode)
	assert.Equal(t, shared.TypeProgram, server2.submissions[1].Type)
	assert.Equal(t, "bob", server2.submissions[1].Submitter)
	assert.Equal(t, "mul $0,3\nsub $0,1\n", server2.submissions[1].Content)
}

func TestCheckpoint_LoadLegacyFormat(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "checkpoint-legacy-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a legacy checkpoint file
	legacyCheckpointPath := filepath.Join(tmpDir, CheckpointFileLegacy)
	legacyContent := `; A000045: Fibonacci numbers
; Submitted by alice
mov $0,1
add $0,2
==============================
; A000142: Factorial
; Submitted by bob
mul $0,3
sub $0,1
==============================
`
	err = os.WriteFile(legacyCheckpointPath, []byte(legacyContent), 0644)
	assert.NoError(t, err)

	// Create a server and load the legacy checkpoint
	server := NewProgramsServer(tmpDir, nil, nil)
	server.loadCheckpoint()

	// Verify that submissions were loaded
	assert.Equal(t, 2, len(server.submissions))
	assert.Equal(t, "alice", server.submissions[0].Submitter)
	assert.Equal(t, "bob", server.submissions[1].Submitter)
}

func TestCheckpoint_MissingFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "checkpoint-missing-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a server and try to load a non-existent checkpoint
	server := NewProgramsServer(tmpDir, nil, nil)
	server.loadCheckpoint()

	// Should not crash, just have empty submissions
	assert.Equal(t, 0, len(server.submissions))
}

func TestCheckSubmit_DuplicateAdd(t *testing.T) {
	// Create a test server
	server := NewProgramsServer("", nil, nil)

	// Create a submission with mode "add"
	id1, _ := util.NewUIDFromString("A000045")
	submission1 := shared.Submission{
		Id:         id1,
		Mode:       shared.ModeAdd,
		Type:       shared.TypeProgram,
		Content:    "mov $0,1\nadd $0,2\n",
		Submitter:  "alice",
		Operations: []string{"mov", "add"},
	}

	// First submission should succeed
	ok, _ := server.checkSubmit(submission1)
	assert.True(t, ok, "First submission should be accepted")
	server.doSubmit(submission1)

	// Duplicate submission with same operations should fail
	submission2 := shared.Submission{
		Id:         id1,
		Mode:       shared.ModeAdd,
		Type:       shared.TypeProgram,
		Content:    "mov $0,1\nadd $0,2\n",
		Submitter:  "alice",
		Operations: []string{"mov", "add"},
	}
	ok, result := server.checkSubmit(submission2)
	assert.False(t, ok, "Duplicate add submission should be rejected")
	assert.Equal(t, "Duplicate submission", result.Message)
}

func TestCheckSubmit_DuplicateRemove(t *testing.T) {
	// Create a test server
	server := NewProgramsServer("", nil, nil)

	// Create a submission with mode "remove"
	id1, _ := util.NewUIDFromString("A000045")
	submission1 := shared.Submission{
		Id:         id1,
		Mode:       shared.ModeRemove,
		Type:       shared.TypeProgram,
		Content:    "",
		Submitter:  "alice",
		Operations: []string{"mov", "add"},
	}

	// First remove submission should succeed
	ok, _ := server.checkSubmit(submission1)
	assert.True(t, ok, "First remove submission should be accepted")
	server.doSubmit(submission1)

	// Duplicate remove submission with same operations should also succeed
	// because duplicate check is skipped for remove mode
	submission2 := shared.Submission{
		Id:         id1,
		Mode:       shared.ModeRemove,
		Type:       shared.TypeProgram,
		Content:    "",
		Submitter:  "alice",
		Operations: []string{"mov", "add"},
	}
	ok, _ = server.checkSubmit(submission2)
	assert.True(t, ok, "Duplicate remove submission should be accepted (duplicate check skipped)")
}

func TestCheckSubmit_DuplicateUpdate(t *testing.T) {
	// Create a test server
	server := NewProgramsServer("", nil, nil)

	// Create a submission with mode "update"
	id1, _ := util.NewUIDFromString("A000045")
	submission1 := shared.Submission{
		Id:         id1,
		Mode:       shared.ModeUpdate,
		Type:       shared.TypeProgram,
		Content:    "mov $0,1\nadd $0,2\n",
		Submitter:  "alice",
		Operations: []string{"mov", "add"},
	}

	// First submission should succeed
	ok, _ := server.checkSubmit(submission1)
	assert.True(t, ok, "First update submission should be accepted")
	server.doSubmit(submission1)

	// Duplicate submission with same operations should fail
	submission2 := shared.Submission{
		Id:         id1,
		Mode:       shared.ModeUpdate,
		Type:       shared.TypeProgram,
		Content:    "mov $0,1\nadd $0,2\n",
		Submitter:  "alice",
		Operations: []string{"mov", "add"},
	}
	ok, result := server.checkSubmit(submission2)
	assert.False(t, ok, "Duplicate update submission should be rejected")
	assert.Equal(t, "Duplicate submission", result.Message)
}
