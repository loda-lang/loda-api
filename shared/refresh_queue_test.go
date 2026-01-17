package shared

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/loda-lang/loda-api/util"
	"github.com/stretchr/testify/assert"
)

func TestRefreshQueue_EnqueueAndDequeue(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "refresh-queue-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a refresh queue
	rq := NewRefreshQueue(tempDir)

	// Enqueue some IDs
	id1, _ := util.NewUIDFromString("A000045")
	id2, _ := util.NewUIDFromString("A000142")
	id3, _ := util.NewUIDFromString("A000001")

	err = rq.Enqueue(id1)
	assert.NoError(t, err)
	err = rq.Enqueue(id2)
	assert.NoError(t, err)
	err = rq.Enqueue(id3)
	assert.NoError(t, err)

	// Dequeue all IDs
	ids, err := rq.DequeueAll()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(ids))
	assert.Contains(t, ids, 45)
	assert.Contains(t, ids, 142)
	assert.Contains(t, ids, 1)

	// Verify queue is empty after dequeue
	ids, err = rq.DequeueAll()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(ids))
}

func TestRefreshQueue_DequeueEmptyQueue(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "refresh-queue-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a refresh queue
	rq := NewRefreshQueue(tempDir)

	// Dequeue from empty queue should return empty list
	ids, err := rq.DequeueAll()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(ids))
}

func TestRefreshQueue_MultipleEnqueueDequeue(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "refresh-queue-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a refresh queue
	rq := NewRefreshQueue(tempDir)

	// First batch
	id1, _ := util.NewUIDFromString("A000045")
	err = rq.Enqueue(id1)
	assert.NoError(t, err)

	ids, err := rq.DequeueAll()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ids))
	assert.Contains(t, ids, 45)

	// Second batch
	id2, _ := util.NewUIDFromString("A000142")
	id3, _ := util.NewUIDFromString("A000001")
	err = rq.Enqueue(id2)
	assert.NoError(t, err)
	err = rq.Enqueue(id3)
	assert.NoError(t, err)

	ids, err = rq.DequeueAll()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(ids))
	assert.Contains(t, ids, 142)
	assert.Contains(t, ids, 1)
}

func TestRefreshQueue_ThreadSafety(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "refresh-queue-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a refresh queue
	rq := NewRefreshQueue(tempDir)

	// Enqueue from multiple goroutines
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			id, _ := util.NewUIDFromString("A000045")
			rq.Enqueue(id)
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	// Dequeue and verify we got all entries
	ids, err := rq.DequeueAll()
	assert.NoError(t, err)
	assert.Equal(t, 10, len(ids))
}

func TestRefreshQueue_PersistenceAcrossInstances(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "refresh-queue-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create first refresh queue instance and enqueue
	rq1 := NewRefreshQueue(tempDir)
	id1, _ := util.NewUIDFromString("A000045")
	err = rq1.Enqueue(id1)
	assert.NoError(t, err)

	// Create second refresh queue instance and dequeue
	rq2 := NewRefreshQueue(tempDir)
	ids, err := rq2.DequeueAll()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ids))
	assert.Contains(t, ids, 45)
}

func TestRefreshQueue_InvalidLines(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "refresh-queue-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a queue file with some invalid lines
	queuePath := filepath.Join(tempDir, RefreshQueueFile)
	content := "123\ninvalid\n456\n\n789\n"
	err = os.WriteFile(queuePath, []byte(content), 0644)
	assert.NoError(t, err)

	// Create refresh queue and dequeue
	rq := NewRefreshQueue(tempDir)
	ids, err := rq.DequeueAll()
	assert.NoError(t, err)
	// Should skip invalid lines and empty lines
	assert.Equal(t, 3, len(ids))
	assert.Contains(t, ids, 123)
	assert.Contains(t, ids, 456)
	assert.Contains(t, ids, 789)
}
