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

// Create OEIS directory
oeisDir := filepath.Join(tmpDir, "seqs", "oeis")
os.MkdirAll(oeisDir, os.ModePerm)

// Create a test server
server := NewSubmissionsServer(tmpDir, oeisDir, nil)

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
server2 := NewSubmissionsServer(tmpDir, oeisDir, nil)
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

func TestCheckpoint_MissingFile(t *testing.T) {
// Create a temporary directory for testing
tmpDir, err := os.MkdirTemp("", "checkpoint-missing-test-*")
assert.NoError(t, err)
defer os.RemoveAll(tmpDir)

// Create OEIS directory
oeisDir := filepath.Join(tmpDir, "seqs", "oeis")
os.MkdirAll(oeisDir, os.ModePerm)

// Create a server and try to load a non-existent checkpoint
server := NewSubmissionsServer(tmpDir, oeisDir, nil)
server.loadCheckpoint()

// Should not crash, just have empty submissions
assert.Equal(t, 0, len(server.submissions))
}

func TestCheckSubmit_DuplicateAdd(t *testing.T) {
// Create a test server
server := NewSubmissionsServer("", "", nil)

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
