package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestRefreshSequence_DeletesBFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "refresh-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create OEIS directory structure
	oeisDir := filepath.Join(tmpDir, "seqs", "oeis")
	os.MkdirAll(oeisDir, os.ModePerm)

	// Create a test server
	server := NewSubmissionsServer(tmpDir, oeisDir, nil)

	// Create a test b-file
	id, _ := util.NewUIDFromString("A000045")
	bfilePath := server.getBFilePath(id)
	os.MkdirAll(filepath.Dir(bfilePath), os.ModePerm)
	err = os.WriteFile(bfilePath, []byte("test content"), 0644)
	assert.NoError(t, err)

	// Verify b-file exists
	_, err = os.Stat(bfilePath)
	assert.NoError(t, err, "B-file should exist before refresh")

	// Create a refresh submission
	submission := shared.Submission{
		Id:        id,
		Mode:      shared.ModeRefresh,
		Type:      shared.TypeSequence,
		Submitter: "tester",
	}

	// Execute refresh
	result := server.refreshSequence(submission)
	assert.Equal(t, "success", result.Status)

	// Verify b-file was deleted
	_, err = os.Stat(bfilePath)
	assert.True(t, os.IsNotExist(err), "B-file should be deleted after refresh")
}

func TestRefreshSequence_NoBFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "refresh-nobfile-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create OEIS directory structure
	oeisDir := filepath.Join(tmpDir, "seqs", "oeis")
	os.MkdirAll(oeisDir, os.ModePerm)

	// Create a test server
	server := NewSubmissionsServer(tmpDir, oeisDir, nil)

	// Create a refresh submission (no b-file exists)
	id, _ := util.NewUIDFromString("A000045")
	submission := shared.Submission{
		Id:        id,
		Mode:      shared.ModeRefresh,
		Type:      shared.TypeSequence,
		Submitter: "tester",
	}

	// Execute refresh - should succeed even without b-file
	result := server.refreshSequence(submission)
	assert.Equal(t, "success", result.Status)
}
func TestRefreshSequence_RateLimitPerHour(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "refresh-ratelimit-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create OEIS directory structure
	oeisDir := filepath.Join(tmpDir, "seqs", "oeis")
	os.MkdirAll(oeisDir, os.ModePerm)

	// Create a test server
	server := NewSubmissionsServer(tmpDir, oeisDir, nil)

	// Create a refresh submission
	id, _ := util.NewUIDFromString("A000045")
	submission := shared.Submission{
		Id:        id,
		Mode:      shared.ModeRefresh,
		Type:      shared.TypeSequence,
		Submitter: "tester",
	}

	// Fill up the rate limit (200 submissions)
	for i := 0; i < SequenceRefreshLimitPerHour; i++ {
		result := server.refreshSequence(submission)
		assert.Equal(t, "success", result.Status, "Submission %d should succeed", i+1)
	}

	// Next submission should be rejected due to rate limit
	result := server.refreshSequence(submission)
	assert.Equal(t, "error", result.Status, "Submission should be rejected due to rate limit")
	assert.Contains(t, result.Message, "Rate limit exceeded")
	assert.Contains(t, result.Message, "200")
}

func TestRefreshSequence_RateLimitExpiry(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "refresh-ratelimit-expiry-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create OEIS directory structure
	oeisDir := filepath.Join(tmpDir, "seqs", "oeis")
	os.MkdirAll(oeisDir, os.ModePerm)

	// Create a test server
	server := NewSubmissionsServer(tmpDir, oeisDir, nil)

	// Create a refresh submission
	id, _ := util.NewUIDFromString("A000045")
	submission := shared.Submission{
		Id:        id,
		Mode:      shared.ModeRefresh,
		Type:      shared.TypeSequence,
		Submitter: "tester",
	}

	// Add two submissions with timestamps 1 hour and 1 second ago
	// This simulates that one is outside the 1-hour window
	now := time.Now()
	server.submissionsMutex.Lock()
	server.refreshSubmissions = []time.Time{
		now.Add(-61 * time.Minute), // More than 1 hour ago, should be cleaned up
		now.Add(-59 * time.Minute), // Less than 1 hour ago, should stay
	}
	server.submissionsMutex.Unlock()

	// Refresh should only count the one submission within the hour
	result := server.refreshSequence(submission)
	assert.Equal(t, "success", result.Status, "Submission should succeed after old timestamp expires")

	// Verify the timestamp was cleaned up (should only have 2 submissions now: the old one and the new one)
	server.submissionsMutex.Lock()
	assert.Equal(t, 2, len(server.refreshSubmissions), "Old timestamp should have been cleaned up")
	server.submissionsMutex.Unlock()
}
