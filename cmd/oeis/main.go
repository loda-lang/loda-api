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
	oeisDir               string
	bfileUpdateInterval   time.Duration
	summaryUpdateInterval time.Duration
	crawlerFetchInterval  time.Duration
	crawlerFlushInterval  time.Duration
	crawler               *Crawler
	httpClient            *http.Client
	lists                 []*List
}

const (
	OeisWebsite string = "https://oeis.org/"
)

var (
	ListNames = map[string]string{
		"A": "authors",
		"C": "comments",
		"F": "formulas",
		"K": "keywords",
		"O": "offsets",
		"o": "programs",
	}
)

func NewOeisServer(oeisDir string, updateInterval time.Duration) *OeisServer {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}
	i := 0
	lists := make([]*List, len(ListNames))
	for key, name := range ListNames {
		lists[i] = NewList(key, name, oeisDir)
		i++
	}
	return &OeisServer{
		oeisDir:               oeisDir,
		bfileUpdateInterval:   180 * 24 * time.Hour, // 6 months
		summaryUpdateInterval: updateInterval,
		crawlerFetchInterval:  30 * time.Second,
		crawlerFlushInterval:  30 * time.Minute,
		crawler:               NewCrawler(httpClient),
		httpClient:            httpClient,
		lists:                 lists,
	}
}

func newSummaryHandler(s *OeisServer, filename string) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		path := filepath.Join(s.oeisDir, filename)
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
		dir := filepath.Join(s.oeisDir, "b", id[0:3])
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

func newListHandler(l *List) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		l.ServeGzip(w, req)
	}
	return http.HandlerFunc(f)
}

func (s *OeisServer) Run(port int) {
	router := mux.NewRouter()
	router.Handle("/v1/oeis/names.gz", newSummaryHandler(s, "names.gz"))
	router.Handle("/v1/oeis/stripped.gz", newSummaryHandler(s, "stripped.gz"))
	router.Handle("/v1/oeis/b{id:[0-9]+}.txt.gz", newBFileHandler(s))
	for _, l := range s.lists {
		router.Handle(fmt.Sprintf("/v1/oeis/%s.gz", l.name), newListHandler(l))
	}
	router.NotFoundHandler = http.HandlerFunc(util.HandleNotFound)
	log.Printf("Using data dir %s", s.oeisDir)
	log.Printf("Listening on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), router)
}

func (s *OeisServer) StartCrawler() {
	err := s.crawler.Init()
	if err != nil {
		log.Fatal(err)
	}
	fetchTicker := time.NewTicker(s.crawlerFetchInterval)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-fetchTicker.C:
				// Reinitialize the crawler every 1000 fetched sequences
				if s.crawler.numFetched > 0 && s.crawler.numFetched%1000 == 0 {
					err = s.crawler.Init()
					if err != nil {
						log.Print("Stopping crawler")
						done <- true
					}
				}
				// Fetch the next sequence
				fields, status, err := s.crawler.FetchNext()
				if err != nil {
					log.Printf("Error fetching fields: %v", err)
					if status <= 500 || status >= 600 {
						log.Print("Stopping crawler")
						done <- true
					}
				} else {
					// Update the lists with the new fields
					for _, l := range s.lists {
						l.Update(fields)
					}
				}
			}
		}
	}()
	flushTicker := time.NewTicker(s.crawlerFlushInterval)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-flushTicker.C:
				for _, l := range s.lists {
					err := l.Flush()
					if err != nil {
						log.Printf("Error flushing list %s: %v", l.name, err)
						log.Print("Stopping crawler")
						done <- true
					}
				}
			}
		}
	}()
}

func main() {
	setup := cmd.GetSetup("oeis")
	util.MustDirExist(setup.DataDir)
	oeisDir := filepath.Join(setup.DataDir, "oeis")
	os.MkdirAll(oeisDir, os.ModePerm)
	s := NewOeisServer(oeisDir, setup.UpdateInterval)
	s.StartCrawler()
	s.Run(8080)
}
