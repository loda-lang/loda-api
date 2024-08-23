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
	dataDir               string
	bfileUpdateInterval   time.Duration
	summaryUpdateInterval time.Duration
	httpClient            *http.Client
}

const (
	OeisWebsite string = "https://oeis.org/"
)

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
		dataDir:               dataDir,
		bfileUpdateInterval:   90 * 24 * time.Hour, // hard-coded to 3 months
		summaryUpdateInterval: updateInterval,
		httpClient:            httpClient,
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
		if !util.IsFileRecent(path, s.summaryUpdateInterval) {
			err := util.FetchFile(s.httpClient, OeisWebsite+filename, path)
			if err != nil {
				util.WriteHttpInternalServerError(w)
				log.Fatal(err)
			}
		}
		util.ServeBinary(w, req, path)
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
		if !util.IsFileRecent(path, s.bfileUpdateInterval) {
			url := fmt.Sprintf("%sA%s/b%s.txt", OeisWebsite, id, id)
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
		util.ServeBinary(w, req, path)
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
