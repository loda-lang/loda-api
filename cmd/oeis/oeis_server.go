package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/loda-lang/loda-api/cmd"
	"github.com/loda-lang/loda-api/util"
)

type OeisServer struct {
	dataDir        string
	updateInterval time.Duration
	httpClient     *http.Client
}

const OEIS_WEBSITE string = "https://oeis.org/"

func NewOeisServer(dataDir string, updateInterval time.Duration) *OeisServer {
	util.MustDirExist(dataDir)
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}
	return &OeisServer{
		dataDir:        dataDir,
		updateInterval: updateInterval,
		httpClient:     httpClient,
	}
}

func newSummaryHandler(s *OeisServer, filename string) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		dir := filepath.Join(s.dataDir, "oeis")
		os.MkdirAll(dir, os.ModePerm)
		path := filepath.Join(dir, filename)
		if !util.IsFileRecent(path, s.updateInterval) {
			err := util.FetchFile(s.httpClient, OEIS_WEBSITE+filename, path)
			if err != nil {
				util.WriteHttpInternalServerError(w)
				log.Fatal(err)
			}
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		log.Printf("Serving %s to %s", filename, req.UserAgent())
		http.ServeFile(w, req, path)
	}
	return http.HandlerFunc(f)
}

func newBFileHandler(s *OeisServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		params := mux.Vars(req)
		id := params["id"]
		if len(id) != 6 {
			util.WriteHttpBadRequest(w)
			return
		}
		dir := filepath.Join(s.dataDir, "oeis", "b", id[0:3])
		os.MkdirAll(dir, os.ModePerm)
		filename := fmt.Sprintf("b%s.txt.gz", id)
		path := filepath.Join(dir, filename)
		if !util.IsFileRecent(path, s.updateInterval) {
			url := fmt.Sprintf("%sA%s/b%s.txt", OEIS_WEBSITE, id, id)
			txtpath := filepath.Join(dir, fmt.Sprintf("b%s.txt", id))
			err := util.FetchFile(s.httpClient, url, txtpath)
			if err != nil {
				util.WriteHttpInternalServerError(w)
				log.Fatal(err)
			}
			err = exec.Command("gzip", "-f", txtpath).Run()
			if err != nil {
				util.WriteHttpInternalServerError(w)
				log.Fatalf("Error executing gzip: %v", err)
			}
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		log.Printf("Serving %s to %s", filename, req.UserAgent())
		http.ServeFile(w, req, path)
	}
	return http.HandlerFunc(f)
}

func (s *OeisServer) Run(port int) {
	router := mux.NewRouter()
	router.Handle("/v1/oeis/names.gz", newSummaryHandler(s, "names.gz"))
	router.Handle("/v1/oeis/stripped.gz", newSummaryHandler(s, "stripped.gz"))
	router.Handle("/v1/oeis/b{id:[0-9]+}.txt.gz", newBFileHandler(s))
	router.NotFoundHandler = http.HandlerFunc(util.HandleNotFound)
	log.Printf("Using data dir %s", s.dataDir)
	log.Printf("Listening on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), router)
}

func main() {
	setup := cmd.GetSetup("oeis")
	s := NewOeisServer(setup.DataDir, setup.UpdateInterval)
	s.Run(8080)
}
