package shared

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Crawler struct {
	maxId      int
	currentId  int
	stepSize   int
	numFetched int
	nextIds    []int
	rand       *rand.Rand
	httpClient *http.Client
	mutex      sync.Mutex
}

func NewCrawler(httpClient *http.Client) *Crawler {
	return &Crawler{
		httpClient: httpClient,
		rand:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (c *Crawler) Init() error {
	log.Print("Initializing crawler")
	maxId, err := c.findMaxId()
	if err != nil {
		return err
	}
	if maxId == 0 {
		return fmt.Errorf("no sequences found")
	}
	c.maxId = maxId
	c.currentId = c.rand.Intn(maxId) + 1
	for i := 0; i < maxId; i++ {
		c.stepSize = c.rand.Intn(maxId) + 1
		if gcd(c.stepSize, maxId) == 1 {
			break
		}
	}
	log.Printf("Found %d sequences", c.maxId)
	return nil
}

func (c *Crawler) FetchSeq(id int, silent bool) ([]Field, int, error) {
	if !silent {
		log.Printf("Fetching A%06d", id)
	}
	url := fmt.Sprintf("https://oeis.org/search?q=id:A%06d&fmt=text", id)

	const maxRetries = 3
	const baseDelay = 2 * time.Second

	var lastErr error
	var lastStatus int

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<uint(attempt-1)) // Exponential backoff: 2s, 4s, 8s
			log.Printf("Retrying A%06d (attempt %d/%d) after %v", id, attempt+1, maxRetries+1, delay)
			time.Sleep(delay)
		}

		resp, err := c.httpClient.Get(url)
		if err != nil {
			lastErr = err
			lastStatus = 0
			continue
		}

		status := resp.StatusCode
		lastStatus = status

		// For server errors (5xx), retry
		if status >= 500 && status < 600 && attempt < maxRetries {
			lastErr = fmt.Errorf("HTTP error: %s", resp.Status)
			resp.Body.Close()
			continue
		}

		// For client errors (4xx) or success, don't retry
		if status >= 400 {
			resp.Body.Close()
			return nil, status, fmt.Errorf("HTTP error: %s", resp.Status)
		}

		scanner := bufio.NewScanner(resp.Body)
		var fields []Field
		for scanner.Scan() {
			line := scanner.Text()
			field, err := ParseField(line)
			if err == nil {
				fields = append(fields, field)
			}
		}
		resp.Body.Close()

		if err := scanner.Err(); err != nil {
			return nil, 0, err
		}
		if len(fields) == 0 {
			return nil, status, fmt.Errorf("no fields found")
		}
		return fields, status, nil
	}

	// All retries exhausted
	return nil, lastStatus, fmt.Errorf("failed after %d retries: %v", maxRetries+1, lastErr)
}

func (c *Crawler) FetchNext() ([]Field, int, error) {
	c.mutex.Lock()
	// Fetch next sequences first
	if len(c.nextIds) > 0 {
		id := c.nextIds[0]
		c.nextIds = c.nextIds[1:]
		c.mutex.Unlock()
		c.numFetched++
		return c.FetchSeq(id, false)
	}
	c.mutex.Unlock()
	// Fetch the next sequence
	if c.maxId == 0 || c.numFetched == c.maxId {
		err := c.Init()
		if err != nil {
			return nil, 0, err
		}
	} else {
		c.currentId = ((c.currentId + c.stepSize) % c.maxId) + 1
	}
	c.numFetched++
	return c.FetchSeq(c.currentId, false)
}

// AddNextId adds an ID to the crawler's next IDs queue in a thread-safe manner.
// Returns false if the queue has reached the maximum size, true otherwise.
func (c *Crawler) AddNextId(id int, maxQueueSize int) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if len(c.nextIds) >= maxQueueSize {
		return false
	}
	c.nextIds = append(c.nextIds, id)
	return true
}

// NumFetched returns the number of sequences fetched
func (c *Crawler) NumFetched() int {
	return c.numFetched
}

// MaxId returns the maximum sequence ID
func (c *Crawler) MaxId() int {
	return c.maxId
}

// SetNextIds sets the next IDs to fetch
func (c *Crawler) SetNextIds(ids []int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.nextIds = ids
}

func (c *Crawler) findMaxId() (int, error) {
	l := 0
	h := 1000000
	var lastError error
	for l < h {
		m := (l + h) / 2
		_, _, lastError := c.FetchSeq(m, true)
		if lastError != nil {
			h = m
		} else {
			l = m + 1
		}
	}
	return h, lastError
}

func gcd(a, b int) int {
	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}
