package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	httpClient            *http.Client
	dataIndex             *shared.DataIndex
	dataIndexMutex        sync.Mutex
}

const (
	OeisWebsite string = "https://oeis.org/"
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
	return &SequencesServer{
		dataDir:               dataDir,
		oeisDir:               oeisDir,
		bfileUpdateInterval:   180 * 24 * time.Hour, // 6 months
		summaryUpdateInterval: updateInterval,
		httpClient:            httpClient,
		dataIndex:             nil,
		dataIndexMutex:        sync.Mutex{},
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

func (s *SequencesServer) SequenceHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		idStr := params["id"]
		uid, err := util.NewUIDFromString(idStr)
		if err != nil {
			util.WriteHttpBadRequest(w)
			return
		}

		switch req.Method {
		case http.MethodGet:
			seq := shared.FindSequenceById(GetIndex(s), uid)
			if seq == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			util.WriteJsonResponse(w, seq)

		default:
			util.WriteHttpMethodNotAllowed(w)
		}
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
	router.Handle("/v2/sequences/search", s.SequenceSearchHandler())
	router.Handle("/v2/sequences/{id:[A-Z][0-9]+}", s.SequenceHandler())
	router.Handle("/v2/sequences/data/oeis/names.gz", newSummaryHandler(s, "names.gz"))
	router.Handle("/v2/sequences/data/oeis/stripped.gz", newSummaryHandler(s, "stripped.gz"))
	router.Handle("/v2/sequences/data/oeis/b{id:[0-9]+}.txt.gz", newBFileHandler(s))
	// List handlers for OEIS data files
	for key, name := range shared.ListNames {
		router.Handle(fmt.Sprintf("/v2/sequences/data/oeis/%s.gz", name), newOeisListHandler(s, key, name))
	}
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

// newOeisListHandler creates a handler that serves pre-generated OEIS list files
func newOeisListHandler(s *SequencesServer, key, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		path := filepath.Join(s.oeisDir, fmt.Sprintf("%s.gz", name))
		util.ServeBinary(w, req, path)
	})
}

func main() {
	setup := cmd.GetSetup("sequences")
	util.MustDirExist(setup.DataDir)
	oeisDir := filepath.Join(setup.DataDir, "seqs", "oeis")
	os.MkdirAll(oeisDir, os.ModePerm)
	s := NewSequencesServer(setup.DataDir, oeisDir, setup.UpdateInterval)
	s.Run(8080)
}
