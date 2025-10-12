package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/loda-lang/loda-api/cmd"
	"github.com/loda-lang/loda-api/shared"
	"github.com/loda-lang/loda-api/util"
)

type SequencesServer struct {
	dataDir               string
	oeisDir               string
	bfileUpdateInterval   time.Duration
	summaryUpdateInterval time.Duration
	crawlerFetchInterval  time.Duration
	crawlerRestartPause   time.Duration
	crawlerFlushInterval  int
	crawlerReinitInterval int
	crawlerIdsCacheSize   int
	crawlerIdsFetchRatio  float64
	crawlerStopped        chan bool
	crawler               *Crawler
	dataIndex             *shared.DataIndex
	httpClient            *http.Client
	lists                 []*List
	dataIndexMutex        sync.Mutex
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

func NewSequencesServer(dataDir string, oeisDir string, updateInterval time.Duration) *SequencesServer {
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
	return &SequencesServer{
		dataDir:               dataDir,
		oeisDir:               oeisDir,
		bfileUpdateInterval:   180 * 24 * time.Hour, // 6 months
		summaryUpdateInterval: updateInterval,
		crawlerFetchInterval:  1 * time.Minute,
		crawlerRestartPause:   24 * time.Hour,
		crawlerFlushInterval:  100,
		crawlerReinitInterval: 2000,
		crawlerIdsCacheSize:   1000,
		crawlerIdsFetchRatio:  0.5,
		crawlerStopped:        make(chan bool),
		crawler:               NewCrawler(httpClient),
		dataIndex:             nil,
		dataIndexMutex:        sync.Mutex{},
		httpClient:            httpClient,
		lists:                 lists,
	}
}

func GetIndex(s *SequencesServer) *shared.DataIndex {
	s.dataIndexMutex.Lock()
	defer s.dataIndexMutex.Unlock()
	if s.dataIndex == nil {
		idx := shared.NewDataIndex(s.dataDir)
		err := idx.Load()
		if err != nil {
			log.Fatalf("Failed to load data index: %v", err)
		}
		// We don't need the programs in memory for the sequences server
		// Also run garbage collection to free memory
		idx.Programs = nil
		runtime.GC()
		s.dataIndex = idx
	}
	return s.dataIndex
}

func newSummaryHandler(s *SequencesServer, filename string) http.Handler {
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
			cmd := exec.Command("gunzip", "-f", "-k", path)
			if err := cmd.Run(); err != nil {
				util.WriteHttpInternalServerError(w)
				log.Fatalf("Error executing gunzip: %v", err)
			}
		}
		util.ServeBinary(w, req, path)
	}
	return http.HandlerFunc(f)
}

func newBFileHandler(s *SequencesServer) http.Handler {
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

func (s *SequencesServer) SequenceHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		params := mux.Vars(req)
		idStr := params["id"]
		uid, err := util.NewUIDFromString(idStr)
		if err != nil {
			util.WriteHttpBadRequest(w)
			return
		}
		seq := shared.FindSequenceById(GetIndex(s), uid)
		if seq == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		util.WriteJsonResponse(w, seq)
	})
}

func (s *SequencesServer) SequenceSearchHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		q := req.URL.Query().Get("q")
		limit, skip, shuffle := util.ParseLimitSkipShuffle(req, 10, 100)
		results, total := shared.SearchSequences(GetIndex(s), q, limit, skip, shuffle)
		resp := shared.SearchResult{
			Total: total,
		}
		for _, seq := range results {
			resp.Results = append(resp.Results, shared.SearchItem{
				Id:       seq.Id.String(),
				Name:     seq.Name,
				Keywords: shared.DecodeKeywords(seq.Keywords),
			})
		}
		util.WriteJsonResponse(w, resp)
	})
}

func (s *SequencesServer) Run(port int) {
	router := mux.NewRouter()
	router.Handle("/v1/oeis/names.gz", newSummaryHandler(s, "names.gz"))
	router.Handle("/v1/oeis/stripped.gz", newSummaryHandler(s, "stripped.gz"))
	router.Handle("/v1/oeis/b{id:[0-9]+}.txt.gz", newBFileHandler(s))
	for _, l := range s.lists {
		router.Handle(fmt.Sprintf("/v1/oeis/%s.gz", l.name), newListHandler(l))
	}
	router.Handle("/v2/sequences/search", s.SequenceSearchHandler())
	router.Handle("/v2/sequences/{id:[A-Z][0-9]+}", s.SequenceHandler())
	router.NotFoundHandler = http.HandlerFunc(util.HandleNotFound)

	// Start goroutine to reset dataIndex to nil at summaryUpdateInterval
	go func() {
		resetTicker := time.NewTicker(s.summaryUpdateInterval)
		defer resetTicker.Stop()
		for {
			<-resetTicker.C
			s.dataIndexMutex.Lock()
			s.dataIndex = nil
			s.dataIndexMutex.Unlock()
			log.Printf("Reset data index")
		}
	}()

	log.Printf("Using data dir %s", s.oeisDir)
	log.Printf("Listening on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), util.CORSHandler(router))
}

func (s *SequencesServer) StopCrawler() {
	log.Print("Stopping crawler")
	s.crawlerStopped <- true
	restartTimer := time.NewTimer(s.crawlerRestartPause)
	go func() {
		<-restartTimer.C
		s.StartCrawler()
	}()
}

// filterValidKeywordsFields filters out unknown keywords from fields with key 'K'.
func filterValidKeywordsFields(fields []Field) []Field {
	filteredFields := make([]Field, 0, len(fields))
	for _, field := range fields {
		if field.Key == "K" {
			var validKeywords []string
			for _, kw := range strings.Split(field.Content, ",") {
				kw = strings.TrimSpace(kw)
				if shared.IsKeyword(kw) {
					validKeywords = append(validKeywords, kw)
				}
			}
			if len(validKeywords) > 0 {
				field.Content = strings.Join(validKeywords, ",")
				filteredFields = append(filteredFields, field)
			}
			// If no valid keywords, skip this field
		} else {
			filteredFields = append(filteredFields, field)
		}
	}
	return filteredFields
}

func (s *SequencesServer) StartCrawler() {
	err := s.crawler.Init()
	if err != nil {
		log.Printf("Error initializing crawler: %v", err)
		return
	}
	fetchTicker := time.NewTicker(s.crawlerFetchInterval)
	s.crawlerStopped = make(chan bool)
	go func() {
		for {
			select {
			case <-s.crawlerStopped:
				return
			case <-fetchTicker.C:
				s.handleCrawlerTick()
			}
		}
	}()
}

// handleCrawlerTick contains the logic for each fetchTicker tick in StartCrawler
func (s *SequencesServer) handleCrawlerTick() {
	if s.crawler.numFetched > 0 {
		// Regularly flush the lists
		if s.crawler.numFetched%s.crawlerFlushInterval == 0 {
			for _, l := range s.lists {
				deduplicate := l.name == "offsets"
				err := l.Flush(deduplicate)
				if err != nil {
					log.Printf("Error flushing list %s: %v", l.name, err)
					s.StopCrawler()
					continue
				}
			}
		}
		// Regularly re-initialize the crawler
		if s.crawler.numFetched%s.crawlerReinitInterval == 0 {
			err := s.crawler.Init()
			if err != nil {
				log.Printf("Error re-initializing crawler: %v", err)
				s.StopCrawler()
				return
			}
		}
	}
	if s.crawler.numFetched%s.crawlerIdsCacheSize == 0 && rand.Float64() < s.crawlerIdsFetchRatio {
		// Find the missing ids
		for _, l := range s.lists {
			if l.name == "offsets" {
				ids, _, err := l.FindMissingIds(s.crawler.maxId, s.crawlerIdsCacheSize)
				if err != nil {
					s.StopCrawler()
					return
				}
				s.crawler.missingIds = ids
				break
			}
		}
	}
	// Fetch the next sequence
	fields, _, err := s.crawler.FetchNext()
	if err != nil {
		log.Printf("Error fetching fields: %v", err)
		s.StopCrawler()
		return
	}
	// Update the lists with the new fields
	filteredFields := filterValidKeywordsFields(fields)
	for _, l := range s.lists {
		l.Update(filteredFields)
	}
}

func main() {
	setup := cmd.GetSetup("sequences")
	util.MustDirExist(setup.DataDir)
	oeisDir := filepath.Join(setup.DataDir, "seqs", "oeis")
	os.MkdirAll(oeisDir, os.ModePerm)
	s := NewSequencesServer(setup.DataDir, oeisDir, setup.UpdateInterval)
	s.StartCrawler()
	s.Run(8080)
}
