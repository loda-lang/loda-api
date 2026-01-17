package shared

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/loda-lang/loda-api/util"
)

const RefreshQueueFile = "refresh_queue.txt"

// RefreshQueue manages a file-based queue of sequence IDs to refresh
type RefreshQueue struct {
	dataDir string
	mutex   sync.Mutex
}

// NewRefreshQueue creates a new RefreshQueue
func NewRefreshQueue(dataDir string) *RefreshQueue {
	return &RefreshQueue{
		dataDir: dataDir,
	}
}

// getQueuePath returns the path to the refresh queue file
func (rq *RefreshQueue) getQueuePath() string {
	return filepath.Join(rq.dataDir, RefreshQueueFile)
}

// Enqueue adds a sequence ID to the refresh queue
func (rq *RefreshQueue) Enqueue(id util.UID) error {
	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	queuePath := rq.getQueuePath()
	file, err := os.OpenFile(queuePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open refresh queue: %v", err)
	}
	defer file.Close()

	// Write the numeric ID (without the 'A' prefix)
	_, err = fmt.Fprintf(file, "%d\n", id.Number())
	if err != nil {
		return fmt.Errorf("failed to write to refresh queue: %v", err)
	}

	return nil
}

// DequeueAll reads all IDs from the queue and clears the file
func (rq *RefreshQueue) DequeueAll() ([]int, error) {
	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	queuePath := rq.getQueuePath()
	
	// Check if file exists
	if !util.FileExists(queuePath) {
		return []int{}, nil
	}

	file, err := os.Open(queuePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open refresh queue: %v", err)
	}
	defer file.Close()

	var ids []int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		id, err := strconv.Atoi(line)
		if err != nil {
			// Skip invalid lines
			continue
		}
		ids = append(ids, id)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading refresh queue: %v", err)
	}

	// Clear the file by truncating it
	if err := os.Truncate(queuePath, 0); err != nil {
		return nil, fmt.Errorf("failed to clear refresh queue: %v", err)
	}

	return ids, nil
}
