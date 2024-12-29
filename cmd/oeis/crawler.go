package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type Crawler struct {
	maxId      int
	currentId  int
	stepSize   int
	numFetched int
	missingIds []int
	rand       *rand.Rand
	httpClient *http.Client
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
	c.maxId = maxId
	c.currentId = c.rand.Intn(maxId) + 1
	for i := 0; i < maxId; i++ {
		c.stepSize = c.rand.Intn(maxId) + 1
		if gcd(c.stepSize, maxId) == 1 {
			break
		}
	}
	log.Printf("Set max ID: %d, current ID: %d, step size: %d", c.maxId, c.currentId, c.stepSize)
	return nil
}

func (c *Crawler) FetchSeq(id int, silent bool) ([]Field, int, error) {
	if !silent {
		log.Printf("Fetching A%06d", id)
	}
	url := fmt.Sprintf("https://oeis.org/search?q=id:A%06d&fmt=text", id)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, 0, err
	}
	status := resp.StatusCode
	if status >= 400 {
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
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}
	if len(fields) == 0 {
		return nil, status, fmt.Errorf("no fields found")
	}
	return fields, status, nil
}

func (c *Crawler) FetchNext() ([]Field, int, error) {
	// Fetch missing sequences first
	if len(c.missingIds) > 0 {
		id := c.missingIds[0]
		c.missingIds = c.missingIds[1:]
		c.numFetched++
		return c.FetchSeq(id, false)
	}
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
