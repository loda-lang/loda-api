package util

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func WriteHttpStatus(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	if len(message) > 0 {
		w.Header().Add("Content-type", "text/plain")
		if message[len(message)-1:] != "\n" {
			message = message + "\n"
		}
		fmt.Fprint(w, message)
	}
}

func WriteHttpOK(w http.ResponseWriter, message string) {
	WriteHttpStatus(w, http.StatusOK, message)
}

func WriteHttpCreated(w http.ResponseWriter, message string) {
	WriteHttpStatus(w, http.StatusCreated, message)
}

func WriteHttpBadRequest(w http.ResponseWriter) {
	WriteHttpStatus(w, http.StatusBadRequest, "Bad Request")
}

func WriteHttpNotFound(w http.ResponseWriter) {
	WriteHttpStatus(w, http.StatusNotFound, "Not Found")
}

func WriteHttpMethodNotAllowed(w http.ResponseWriter) {
	WriteHttpStatus(w, http.StatusMethodNotAllowed, "Method Not Allowed")
}

func WriteHttpTooManyRequests(w http.ResponseWriter) {
	WriteHttpStatus(w, http.StatusTooManyRequests, "Too Many Requests")
}

func WriteHttpInternalServerError(w http.ResponseWriter) {
	WriteHttpStatus(w, http.StatusInternalServerError, "Internal Server Error")
}

// WriteJsonResponse writes the given value as JSON to the response writer, sets content-type, and handles errors
func WriteJsonResponse(w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		WriteHttpInternalServerError(w)
	}
}

func HandleNotFound(w http.ResponseWriter, r *http.Request) {
	log.Printf("Not found: %s", r.URL.String())
	WriteHttpNotFound(w)
}

func CORSHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "https://loda-lang.org/" {
			w.Header().Set("Access-Control-Allow-Origin", "https://loda-lang.org/")
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func FetchFile(httpClient *http.Client, url string, localFile string) error {
	os.Remove(localFile)
	file, err := os.Create(localFile)
	if err != nil {
		return err
	}
	log.Print("Fetching " + url)
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}
	defer resp.Body.Close()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func ParseAuthInfo(auth string) (string, string) {
	a := strings.Split(auth, ":")
	if len(a) != 2 {
		log.Fatalf("Invalid auth info: %s", auth)
	}
	return a[0], a[1]
}

func ServeBinary(w http.ResponseWriter, req *http.Request, path string) {
	log.Printf("Serving %s to %s", filepath.Base(path), req.UserAgent())
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, req, path)
}

// ParseLimitSkip extracts 'limit' and 'skip' query params, applies bounds, and returns (limit, skip)
func ParseLimitSkip(req *http.Request, defaultLimit, maxLimit int) (limit, skip int) {
	limit = defaultLimit
	skip = 0
	if l := req.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
		if limit < 0 {
			limit = 0
		} else if limit > maxLimit {
			limit = maxLimit
		}
	}
	if s := req.URL.Query().Get("skip"); s != "" {
		fmt.Sscanf(s, "%d", &skip)
		if skip < 0 {
			skip = 0
		}
	}
	return
}
