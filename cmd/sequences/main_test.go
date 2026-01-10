package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestSequenceHandler_POST_EmptyBody(t *testing.T) {
	// Create a test server
	s := NewSequencesServer(".", ".", 0)
	s.crawler = NewCrawler(http.DefaultClient)

	// Create a request with empty body
	req, err := http.NewRequest("POST", "/v2/sequences/A000045", nil)
	assert.NoError(t, err)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Create a router and register the handler
	router := mux.NewRouter()
	router.Handle("/v2/sequences/{id:[A-Z][0-9]+}", s.SequenceHandler())
	router.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status OK")

	// Check the response body
	var response map[string]string
	err = json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response["status"])
	assert.Contains(t, response["message"], "A000045")

	// Check that the ID was added to the crawler's nextIds
	assert.Equal(t, 1, len(s.crawler.nextIds), "Expected one ID in nextIds")
	assert.Equal(t, 45, s.crawler.nextIds[0], "Expected ID 45 in nextIds")
}

func TestSequenceHandler_POST_NonEmptyBody(t *testing.T) {
	// Create a test server
	s := NewSequencesServer(".", ".", 0)
	s.crawler = NewCrawler(http.DefaultClient)

	// Create a request with non-empty body
	body := bytes.NewBufferString("some content")
	req, err := http.NewRequest("POST", "/v2/sequences/A000045", body)
	assert.NoError(t, err)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Create a router and register the handler
	router := mux.NewRouter()
	router.Handle("/v2/sequences/{id:[A-Z][0-9]+}", s.SequenceHandler())
	router.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected status Bad Request")
	
	var response map[string]string
	err = json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "error", response["status"])
	assert.Contains(t, response["message"], "empty")

	// Check that no ID was added to the crawler's nextIds
	assert.Equal(t, 0, len(s.crawler.nextIds), "Expected no IDs in nextIds")
}

func TestSequenceHandler_POST_InvalidID(t *testing.T) {
	// Create a test server
	s := NewSequencesServer(".", ".", 0)
	s.crawler = NewCrawler(http.DefaultClient)

	// Create a request with invalid ID
	req, err := http.NewRequest("POST", "/v2/sequences/INVALID", nil)
	assert.NoError(t, err)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Create a router and register the handler
	router := mux.NewRouter()
	router.Handle("/v2/sequences/{id:[A-Z][0-9]+}", s.SequenceHandler())
	router.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusNotFound, rr.Code, "Expected status Not Found")
}

func TestCrawler_AddNextId(t *testing.T) {
	c := NewCrawler(http.DefaultClient)
	
	// Add IDs to the crawler
	c.AddNextId(1)
	c.AddNextId(2)
	c.AddNextId(3)

	// Check that the IDs were added
	assert.Equal(t, 3, len(c.nextIds), "Expected 3 IDs in nextIds")
	assert.Equal(t, 1, c.nextIds[0])
	assert.Equal(t, 2, c.nextIds[1])
	assert.Equal(t, 3, c.nextIds[2])
}

func TestCrawler_AddNextId_ThreadSafety(t *testing.T) {
	c := NewCrawler(http.DefaultClient)
	
	// Add IDs concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			c.AddNextId(id)
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check that all IDs were added
	assert.Equal(t, 10, len(c.nextIds), "Expected 10 IDs in nextIds")
}
