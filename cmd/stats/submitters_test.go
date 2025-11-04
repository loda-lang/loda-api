package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/loda-lang/loda-api/shared"
)

func TestSubmittersHandler(t *testing.T) {
	// Load test data
	path := filepath.Join("../..", "testdata", "stats", "submitters.csv")
	submitters, err := shared.LoadSubmittersCSV(path)
	if err != nil {
		t.Fatalf("failed to load submitters: %v", err)
	}

	// Create a test server
	s := &StatsServer{
		submitters: submitters,
	}
	handler := newSubmittersHandler(s)

	// Test 1: Get submitters with default limit (10)
	t.Run("default limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/stats/submitters", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var result []shared.Submitter
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should return 10 submitters (default limit is 10, and we have exactly 10 non-nil submitters)
		if len(result) != 10 {
			t.Errorf("expected 10 submitters, got %d", len(result))
		}
	})

	// Test 2: Get all submitters (limit=0 means no limit)
	t.Run("no limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/stats/submitters?limit=0", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var result []shared.Submitter
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should return all 10 non-nil submitters
		if len(result) != 10 {
			t.Errorf("expected 10 submitters, got %d", len(result))
		}
	})

	// Test 3: Pagination with limit
	t.Run("with limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/stats/submitters?limit=3", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var result []shared.Submitter
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(result) != 3 {
			t.Errorf("expected 3 submitters, got %d", len(result))
		}
	})

	// Test 4: Pagination with skip
	t.Run("with skip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/stats/submitters?skip=2", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var result []shared.Submitter
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should return 8 submitters (10 total - 2 skipped)
		if len(result) != 8 {
			t.Errorf("expected 8 submitters, got %d", len(result))
		}
	})

	// Test 5: Pagination with both limit and skip
	t.Run("with limit and skip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/stats/submitters?limit=2&skip=1", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var result []shared.Submitter
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 submitters, got %d", len(result))
		}

		// Get all submitters to verify correct items are returned
		reqAll := httptest.NewRequest(http.MethodGet, "/v2/stats/submitters?limit=0", nil)
		wAll := httptest.NewRecorder()
		handler.ServeHTTP(wAll, reqAll)
		var allResult []shared.Submitter
		json.NewDecoder(wAll.Body).Decode(&allResult)

		// Verify the skipped items match the expected slice
		if result[0].Name != allResult[1].Name {
			t.Errorf("expected first item to be %q, got %q", allResult[1].Name, result[0].Name)
		}
		if result[1].Name != allResult[2].Name {
			t.Errorf("expected second item to be %q, got %q", allResult[2].Name, result[1].Name)
		}
	})

	// Test 6: Skip beyond available items
	t.Run("skip beyond available", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/stats/submitters?skip=100", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var result []shared.Submitter
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should return empty array
		if len(result) != 0 {
			t.Errorf("expected 0 submitters, got %d", len(result))
		}
	})

	// Test 7: Limit larger than available items (capped at maxLimit=100)
	t.Run("limit larger than available", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/stats/submitters?limit=1000", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var result []shared.Submitter
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should return all 10 submitters
		if len(result) != 10 {
			t.Errorf("expected 10 submitters, got %d", len(result))
		}
	})

	// Test 8: Method not allowed
	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v2/stats/submitters", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", w.Code)
		}
	})
}
