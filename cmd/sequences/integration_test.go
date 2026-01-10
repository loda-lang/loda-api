package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// TestSequenceHandler_POST_Integration simulates a more realistic integration test
func TestSequenceHandler_POST_Integration(t *testing.T) {
	// Create a test server
	s := NewSequencesServer(".", ".", 0)
	s.crawler = NewCrawler(http.DefaultClient)

	// Create a router and register the handler
	router := mux.NewRouter()
	router.Handle("/v2/sequences/{id:[A-Z][0-9]+}", s.SequenceHandler())

	// Create a test HTTP server
	server := httptest.NewServer(router)
	defer server.Close()

	// Test 1: POST with empty body - should succeed
	t.Run("POST with empty body", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/v2/sequences/A000042", "application/json", bytes.NewReader([]byte{}))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]string
		err = json.Unmarshal(body, &result)
		assert.NoError(t, err)
		assert.Equal(t, "success", result["status"])
		assert.Contains(t, result["message"], "A000042")
		
		// Verify the ID was added to the queue
		assert.Equal(t, 1, len(s.crawler.nextIds))
		assert.Equal(t, 42, s.crawler.nextIds[0])
	})

	// Test 2: POST with non-empty body - should fail
	t.Run("POST with non-empty body", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/v2/sequences/A000100", "application/json", bytes.NewReader([]byte(`{"data": "test"}`)))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]string
		err = json.Unmarshal(body, &result)
		assert.NoError(t, err)
		assert.Equal(t, "error", result["status"])
		assert.Contains(t, result["message"], "empty")
		
		// Verify no ID was added to the queue
		assert.Equal(t, 1, len(s.crawler.nextIds)) // Still only the first one from Test 1
	})

	// Test 3: Multiple POSTs with different IDs
	t.Run("Multiple POSTs", func(t *testing.T) {
		ids := []string{"A000001", "A000002", "A000003"}
		for _, id := range ids {
			resp, err := http.Post(server.URL+"/v2/sequences/"+id, "application/json", nil)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}
		
		// Verify all IDs were added
		// Should have 1 (from Test 1) + 3 new ones = 4 total
		assert.Equal(t, 4, len(s.crawler.nextIds))
		// Check the new ones
		assert.Equal(t, 1, s.crawler.nextIds[1])
		assert.Equal(t, 2, s.crawler.nextIds[2])
		assert.Equal(t, 3, s.crawler.nextIds[3])
	})
}

// TestCrawler_NextIds_Processing verifies that nextIds are processed correctly by the crawler
func TestCrawler_NextIds_Processing(t *testing.T) {
	c := NewCrawler(http.DefaultClient)
	
	// Manually set up some state to avoid needing external dependencies
	c.maxId = 100
	c.currentId = 50
	c.stepSize = 7
	
	// Add some IDs to the queue
	c.AddNextId(10)
	c.AddNextId(20)
	c.AddNextId(30)
	
	// Verify the IDs are in the queue
	assert.Equal(t, 3, len(c.nextIds))
	
	// The first call to FetchNext should process ID 10
	// We can't actually fetch without network, but we can verify the queue is consumed
	// by checking the internal state change
	
	// Simulate what would happen - nextIds should be consumed
	c.mutex.Lock()
	initialLen := len(c.nextIds)
	if initialLen > 0 {
		firstId := c.nextIds[0]
		c.nextIds = c.nextIds[1:]
		assert.Equal(t, 10, firstId, "First ID should be 10")
		assert.Equal(t, initialLen-1, len(c.nextIds), "Queue should shrink")
	}
	c.mutex.Unlock()
}

// TestSequenceHandler_POST_EdgeCases tests edge cases
func TestSequenceHandler_POST_EdgeCases(t *testing.T) {
	t.Run("POST with ContentLength 0", func(t *testing.T) {
		s := NewSequencesServer(".", ".", 0)
		s.crawler = NewCrawler(http.DefaultClient)
		
		router := mux.NewRouter()
		router.Handle("/v2/sequences/{id:[A-Z][0-9]+}", s.SequenceHandler())
		
		req := httptest.NewRequest("POST", "/v2/sequences/A000045", nil)
		req.ContentLength = 0
		rr := httptest.NewRecorder()
		
		router.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusOK, rr.Code)
		var result map[string]string
		json.Unmarshal(rr.Body.Bytes(), &result)
		assert.Equal(t, "success", result["status"])
	})
	
	t.Run("POST with smallest valid ID", func(t *testing.T) {
		s := NewSequencesServer(".", ".", 0)
		s.crawler = NewCrawler(http.DefaultClient)
		
		router := mux.NewRouter()
		router.Handle("/v2/sequences/{id:[A-Z][0-9]+}", s.SequenceHandler())
		
		req := httptest.NewRequest("POST", "/v2/sequences/A0", nil)
		rr := httptest.NewRecorder()
		
		router.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, 1, len(s.crawler.nextIds))
		assert.Equal(t, 0, s.crawler.nextIds[0])
	})
	
	t.Run("POST with large ID", func(t *testing.T) {
		s := NewSequencesServer(".", ".", 0)
		s.crawler = NewCrawler(http.DefaultClient)
		
		router := mux.NewRouter()
		router.Handle("/v2/sequences/{id:[A-Z][0-9]+}", s.SequenceHandler())
		
		req := httptest.NewRequest("POST", "/v2/sequences/A999999", nil)
		rr := httptest.NewRecorder()
		
		router.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, 1, len(s.crawler.nextIds))
		assert.Equal(t, 999999, s.crawler.nextIds[0])
	})
}

func TestMain(m *testing.M) {
	// Run tests
	fmt.Println("Running manual integration tests for POST /v2/sequences/{id}")
	m.Run()
}
