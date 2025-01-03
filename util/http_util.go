package util

import (
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

func HandleNotFound(w http.ResponseWriter, r *http.Request) {
	log.Printf("Not found: %s", r.URL.String())
	WriteHttpNotFound(w)
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
