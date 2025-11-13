package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/loda-lang/loda-api/shared"
	"github.com/loda-lang/loda-api/util"
	"github.com/stretchr/testify/assert"
)

// TestV2SubmissionsPostHandler tests the POST /v2/submissions endpoint
func TestV2SubmissionsPostHandler(t *testing.T) {
	// Create a test server
	s := &ProgramsServer{
		submissions:           []shared.Submission{},
		submissionsPerProfile: make(map[string]int),
		submissionsPerUser:    make(map[string]int),
	}

	// Test valid submission
	submission := map[string]interface{}{
		"id":             "A000045",
		"submitter":      "alice",
		"content":        "mov $0,1\nadd $0,2",
		"mode": "add",
		"type":     "program",
	}
	body, _ := json.Marshal(submission)

	req := httptest.NewRequest(http.MethodPost, "/v2/submissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := newV2SubmissionsPostHandler(s)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "success", result["status"])

	// Verify submission was stored
	assert.Equal(t, 1, len(s.submissions))
	assert.Equal(t, "A000045", s.submissions[0].Id.String())
	assert.Equal(t, "alice", s.submissions[0].Submitter)
}

// TestV2SubmissionsPostHandler_InvalidType tests rejection of non-program submissions
func TestV2SubmissionsPostHandler_InvalidType(t *testing.T) {
	s := &ProgramsServer{
		submissions:           []shared.Submission{},
		submissionsPerProfile: make(map[string]int),
		submissionsPerUser:    make(map[string]int),
	}

	submission := map[string]interface{}{
		"id":             "A000045",
		"submitter":      "alice",
		"content":        "1,1,2,3,5,8",
		"mode": "add",
		"type":     "sequence",
	}
	body, _ := json.Marshal(submission)

	req := httptest.NewRequest(http.MethodPost, "/v2/submissions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := newV2SubmissionsPostHandler(s)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "error", result["status"])
	assert.Contains(t, result["message"], "Only program submissions")
}

// TestV2SubmissionsGetHandler tests the GET /v2/submissions endpoint
func TestV2SubmissionsGetHandler(t *testing.T) {
	// Create test server with some submissions
	id1, _ := util.NewUIDFromString("A000045")
	id2, _ := util.NewUIDFromString("A000142")

	s := &ProgramsServer{
		submissions: []shared.Submission{
			{
				Id:             id1,
				Submitter:      "alice",
				Content:        "mov $0,1",
				Mode: shared.ModeAdd,
				Type:     shared.TypeProgram,
			},
			{
				Id:             id2,
				Submitter:      "bob",
				Content:        "mul $0,2",
				Mode: shared.ModeUpdate,
				Type:     shared.TypeProgram,
			},
		},
		submissionsPerProfile: make(map[string]int),
		submissionsPerUser:    make(map[string]int),
	}

	req := httptest.NewRequest(http.MethodGet, "/v2/submissions", nil)
	w := httptest.NewRecorder()

	handler := newV2SubmissionsGetHandler(s)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result shared.SubmissionsResult
	err := json.NewDecoder(w.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 2, len(result.Results))
	assert.Equal(t, "A000045", result.Results[0].Id.String())
	assert.Equal(t, "alice", result.Results[0].Submitter)
}

// TestV2SubmissionsGetHandler_Pagination tests pagination
func TestV2SubmissionsGetHandler_Pagination(t *testing.T) {
	// Create test server with multiple submissions
	submissions := []shared.Submission{}
	for i := 0; i < 25; i++ {
		id, _ := util.NewUIDFromString("A000045")
		submissions = append(submissions, shared.Submission{
			Id:             id,
			Submitter:      "alice",
			Content:        "mov $0,1",
			Mode: shared.ModeAdd,
			Type:     shared.TypeProgram,
		})
	}

	s := &ProgramsServer{
		submissions:           submissions,
		submissionsPerProfile: make(map[string]int),
		submissionsPerUser:    make(map[string]int),
	}

	// Test first page
	req := httptest.NewRequest(http.MethodGet, "/v2/submissions?limit=10&skip=0", nil)
	w := httptest.NewRecorder()

	handler := newV2SubmissionsGetHandler(s)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result shared.SubmissionsResult
	err := json.NewDecoder(w.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, 25, result.Total)
	assert.Equal(t, 10, len(result.Results))

	// Test second page
	req = httptest.NewRequest(http.MethodGet, "/v2/submissions?limit=10&skip=10", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	err = json.NewDecoder(w.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, 25, result.Total)
	assert.Equal(t, 10, len(result.Results))

	// Test third page
	req = httptest.NewRequest(http.MethodGet, "/v2/submissions?limit=10&skip=20", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	err = json.NewDecoder(w.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, 25, result.Total)
	assert.Equal(t, 5, len(result.Results))
}

// TestV2SubmissionsPostHandler_MissingFields tests validation
func TestV2SubmissionsPostHandler_MissingFields(t *testing.T) {
	s := &ProgramsServer{
		submissions:           []shared.Submission{},
		submissionsPerProfile: make(map[string]int),
		submissionsPerUser:    make(map[string]int),
	}

	tests := []struct {
		name       string
		submission map[string]interface{}
		errMsg     string
	}{
		{
			name: "missing submitter",
			submission: map[string]interface{}{
				"id":             "A000045",
				"content":        "mov $0,1",
				"mode": "add",
				"type":     "program",
			},
			errMsg: "Missing submitter",
		},
		{
			name: "missing content",
			submission: map[string]interface{}{
				"id":             "A000045",
				"submitter":      "alice",
				"mode": "add",
				"type":     "program",
			},
			errMsg: "Missing content",
		},
		{
			name: "empty content",
			submission: map[string]interface{}{
				"id":             "A000045",
				"submitter":      "alice",
				"content":        "",
				"mode": "add",
				"type":     "program",
			},
			errMsg: "Missing content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.submission)
			req := httptest.NewRequest(http.MethodPost, "/v2/submissions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler := newV2SubmissionsPostHandler(s)
			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var result map[string]interface{}
			err := json.NewDecoder(w.Body).Decode(&result)
			assert.NoError(t, err)
			assert.Equal(t, "error", result["status"])
			assert.Contains(t, result["message"], tt.errMsg)
		})
	}
}

// TestV2SubmissionsEndToEnd tests both POST and GET together
func TestV2SubmissionsEndToEnd(t *testing.T) {
	s := &ProgramsServer{
		submissions:           []shared.Submission{},
		submissionsPerProfile: make(map[string]int),
		submissionsPerUser:    make(map[string]int),
	}

	// Submit first submission
	submission1 := map[string]interface{}{
		"id":             "A000045",
		"submitter":      "alice",
		"content":        "mov $0,1",
		"mode": "add",
		"type":     "program",
	}
	body1, _ := json.Marshal(submission1)
	req := httptest.NewRequest(http.MethodPost, "/v2/submissions", bytes.NewBuffer(body1))
	w := httptest.NewRecorder()
	newV2SubmissionsPostHandler(s).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Submit second submission
	submission2 := map[string]interface{}{
		"id":             "A000142",
		"submitter":      "bob",
		"content":        "mul $0,2",
		"mode": "update",
		"type":     "program",
	}
	body2, _ := json.Marshal(submission2)
	req = httptest.NewRequest(http.MethodPost, "/v2/submissions", bytes.NewBuffer(body2))
	w = httptest.NewRecorder()
	newV2SubmissionsPostHandler(s).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Get all submissions
	req = httptest.NewRequest(http.MethodGet, "/v2/submissions", nil)
	w = httptest.NewRecorder()
	newV2SubmissionsGetHandler(s).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result shared.SubmissionsResult
	err := json.NewDecoder(w.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 2, len(result.Results))
	assert.Equal(t, "A000045", result.Results[0].Id.String())
	assert.Equal(t, "alice", result.Results[0].Submitter)
	assert.Equal(t, "A000142", result.Results[1].Id.String())
	assert.Equal(t, "bob", result.Results[1].Submitter)
}

// TestV2SubmissionsRoutes tests that routes are properly configured
func TestV2SubmissionsRoutes(t *testing.T) {
	s := &ProgramsServer{
		submissions:           []shared.Submission{},
		submissionsPerProfile: make(map[string]int),
		submissionsPerUser:    make(map[string]int),
	}

	router := mux.NewRouter()
	router.Handle("/v2/submissions", newV2SubmissionsGetHandler(s)).Methods(http.MethodGet)
	router.Handle("/v2/submissions", newV2SubmissionsPostHandler(s)).Methods(http.MethodPost)

	// Test GET route
	req := httptest.NewRequest(http.MethodGet, "/v2/submissions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test POST route
	submission := map[string]interface{}{
		"id":             "A000045",
		"submitter":      "alice",
		"content":        "mov $0,1",
		"mode": "add",
		"type":     "program",
	}
	body, _ := json.Marshal(submission)
	req = httptest.NewRequest(http.MethodPost, "/v2/submissions", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
