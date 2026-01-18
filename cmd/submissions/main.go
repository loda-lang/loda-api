package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/loda-lang/loda-api/cmd"
	"github.com/loda-lang/loda-api/shared"
	"github.com/loda-lang/loda-api/util"
)

const (
	NumSubmissionsLow           = 1000
	NumSubmissionsHigh          = 2000
	NumSubmissionsMax           = 50000
	NumSubmissionsPerUser       = 100
	MaxProgramLength            = 100000
	CheckpointInterval          = 10 * time.Minute
	CheckSessionInterval        = 24 * time.Hour
	CheckpointFile              = "checkpoint.json"
	OeisWebsite                 = "https://oeis.org/"
	SequenceRefreshLimitPerHour = 200
)

type OperationResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type SubmissionsServer struct {
	dataDir               string
	oeisDir               string
	influxDbClient        *util.InfluxDbClient
	session               time.Time
	submissions           []shared.Submission // Unified submissions (v1 and v2)
	submissionsPerProfile map[string]int
	submissionsPerUser    map[string]int
	refreshSubmissions    []time.Time // Tracks sequence refresh submission timestamps for rate limiting
	httpClient            *http.Client
	crawler               *shared.Crawler
	lists                 []*shared.List
	crawlerFetchInterval  time.Duration
	crawlerRestartPause   time.Duration
	crawlerFlushInterval  int
	crawlerReinitInterval int
	crawlerIdsCacheSize   int
	crawlerIdsFetchRatio  float64
	crawlerMaxQueueSize   int
	crawlerStopped        chan bool
	submissionsMutex      sync.Mutex
}

func NewSubmissionsServer(dataDir string, oeisDir string, influxDbClient *util.InfluxDbClient) *SubmissionsServer {
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
	lists := make([]*shared.List, len(shared.ListNames))
	for key, name := range shared.ListNames {
		lists[i] = shared.NewList(key, name, oeisDir)
		i++
	}
	return &SubmissionsServer{
		dataDir:               dataDir,
		oeisDir:               oeisDir,
		influxDbClient:        influxDbClient,
		session:               time.Now(),
		submissions:           []shared.Submission{},
		submissionsPerProfile: make(map[string]int),
		submissionsPerUser:    make(map[string]int),
		refreshSubmissions:    []time.Time{},
		httpClient:            httpClient,
		crawler:               shared.NewCrawler(httpClient),
		lists:                 lists,
		crawlerFetchInterval:  1 * time.Minute,
		crawlerRestartPause:   24 * time.Hour,
		crawlerFlushInterval:  100,
		crawlerReinitInterval: 2000,
		crawlerIdsCacheSize:   1000,
		crawlerIdsFetchRatio:  0.5,
		crawlerMaxQueueSize:   10000,
		crawlerStopped:        make(chan bool),
	}
}

// Returns (ok, OperationResult)
func (s *SubmissionsServer) checkSubmit(submission shared.Submission) (bool, OperationResult) {
	s.submissionsMutex.Lock()
	defer s.submissionsMutex.Unlock()
	s.checkSession()
	if len(s.submissions) > NumSubmissionsMax {
		log.Print("Maximum number of submissions exceeded")
		return false, OperationResult{Status: "error", Message: "Too many total submissions"}
	}
	if s.submissionsPerUser[submission.Submitter] >= NumSubmissionsPerUser {
		log.Printf("Rejected submission from %s", submission.Submitter)
		return false, OperationResult{Status: "error", Message: "Too many user submissions"}
	}
	// Skip duplicate check for remove mode
	if submission.Mode != shared.ModeRemove {
		for _, p := range s.submissions {
			if slices.Equal(p.Operations, submission.Operations) {
				return false, OperationResult{Status: "error", Message: "Duplicate submission"}
			}
		}
	}
	return true, OperationResult{}
}

func (s *SubmissionsServer) doSubmit(submission shared.Submission) OperationResult {
	profile := submission.MinerProfile
	if len(profile) == 0 {
		profile = "unknown"
	}
	s.submissionsMutex.Lock()
	defer s.submissionsMutex.Unlock()
	s.submissions = append(s.submissions, submission)
	s.submissionsPerUser[submission.Submitter]++
	s.submissionsPerProfile[profile]++
	msg := fmt.Sprintf("Accepted submission from %s (%d/%d); profile %s (%d)",
		submission.Submitter, s.submissionsPerUser[submission.Submitter], NumSubmissionsPerUser,
		profile, s.submissionsPerProfile[profile])
	log.Print(msg)
	return OperationResult{Status: "success", Message: "Accepted submission"}
}

// getBFilePath returns the path to a b-file for the given sequence ID.
// The ID is validated using util.NewUIDFromString format (e.g., "A000045").
func (s *SubmissionsServer) getBFilePath(id util.UID) string {
	idStr := id.String()
	numericId := idStr[1:] // e.g., "000045"
	dir := filepath.Join(s.dataDir, "seqs", "oeis", "b", numericId[0:3])
	filename := fmt.Sprintf("b%s.txt.gz", numericId)
	return filepath.Join(dir, filename)
}

// refreshSequence adds a sequence ID to the crawler's next IDs queue
// and deletes the b-file if it exists
func (s *SubmissionsServer) refreshSequence(submission shared.Submission) OperationResult {
	idStr := submission.Id.String()

	// Check rate limit (200 per hour)
	s.submissionsMutex.Lock()
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	// Remove timestamps older than 1 hour
	validRefreshes := []time.Time{}
	for _, ts := range s.refreshSubmissions {
		if ts.After(oneHourAgo) {
			validRefreshes = append(validRefreshes, ts)
		}
	}
	s.refreshSubmissions = validRefreshes

	// Check if we've exceeded the limit
	if len(s.refreshSubmissions) >= SequenceRefreshLimitPerHour {
		s.submissionsMutex.Unlock()
		remaining := s.refreshSubmissions[0].Add(1 * time.Hour).Sub(now)
		remainingSeconds := int(remaining.Seconds())
		msg := fmt.Sprintf("Rate limit exceeded: maximum %d sequence refreshes per hour. Please try again in %d seconds.", SequenceRefreshLimitPerHour, remainingSeconds)
		log.Printf("%s: %s", msg, submission.Submitter)
		return OperationResult{Status: "error", Message: msg}
	}

	// Record the refresh submission
	s.refreshSubmissions = append(s.refreshSubmissions, now)
	s.submissionsMutex.Unlock()

	// Delete the b-file if it exists
	bfilePath := s.getBFilePath(submission.Id)
	if util.FileExists(bfilePath) {
		if err := os.Remove(bfilePath); err != nil {
			log.Printf("Warning: Failed to remove b-file %s during refresh: %v", bfilePath, err)
			// Continue with refresh even if b-file deletion fails
		} else {
			log.Printf("Deleted b-file for sequence %s during refresh", idStr)
		}
	}

	// Add to crawler queue
	success := s.crawler.AddNextId(int(submission.Id.Number()), s.crawlerMaxQueueSize)
	if !success {
		log.Printf("Failed to add sequence %s to crawler queue (queue full)", idStr)
		return OperationResult{Status: "error", Message: "Crawler queue is full, please retry later"}
	}

	log.Printf("Added sequence %s to crawler queue by %s", idStr, submission.Submitter)
	return OperationResult{Status: "success", Message: fmt.Sprintf("Sequence %s added to crawler queue", idStr)}
}

func (s *SubmissionsServer) writeCheckpoint() error {
	s.submissionsMutex.Lock()
	defer s.submissionsMutex.Unlock()
	f, err := os.Create(filepath.Join(s.dataDir, CheckpointFile))
	if err != nil {
		return fmt.Errorf("cannot open checkpoint file: %v", err)
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s.submissions); err != nil {
		return fmt.Errorf("cannot write to checkpoint file: %v", err)
	}
	return nil
}

func (s *SubmissionsServer) checkSession() {
	if len(s.submissions) < NumSubmissionsHigh {
		return
	}
	if time.Since(s.session).Minutes() < CheckSessionInterval.Minutes() {
		return
	}
	s.session = time.Now()
	log.Printf("Starting new session: %v", s.session)
	if len(s.submissions) > NumSubmissionsLow {
		end := len(s.submissions)
		start := end - NumSubmissionsLow
		s.submissions = s.submissions[start:end]
	}
}

func (s *SubmissionsServer) publishMetrics() {
	s.submissionsMutex.Lock()
	defer s.submissionsMutex.Unlock()
	totalCount := 0
	for profile, count := range s.submissionsPerProfile {
		labels := map[string]string{"kind": "submitted", "profile": profile}
		s.influxDbClient.Write("programs", labels, count)
		totalCount += count
	}
	s.submissionsPerProfile = make(map[string]int)
}

func (s *SubmissionsServer) clearUserStats() {
	s.submissionsMutex.Lock()
	defer s.submissionsMutex.Unlock()
	s.submissionsPerUser = make(map[string]int)
}

func (s *SubmissionsServer) loadCheckpoint() {
	checkpointPath := filepath.Join(s.dataDir, CheckpointFile)
	file, err := os.Open(checkpointPath)
	if err != nil {
		log.Printf("Cannot load checkpoint %s", checkpointPath)
		return
	}
	defer file.Close()
	log.Printf("Loading checkpoint %s", checkpointPath)
	s.submissions = []shared.Submission{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&s.submissions); err != nil {
		log.Printf("Cannot decode checkpoint JSON: %v", err)
		return
	}
	log.Printf("Loaded %v submissions from checkpoint", len(s.submissions))
}

// newV2SubmissionsGetHandler handles GET requests for v2/submissions
func newV2SubmissionsGetHandler(s *SubmissionsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		limit, skip, _ := util.ParseLimitSkipShuffle(req, 10, 100)

		// Get filter parameters
		modeFilter := req.URL.Query().Get("mode")
		typeFilter := req.URL.Query().Get("type")
		submitterFilter := req.URL.Query().Get("submitter")

		s.submissionsMutex.Lock()
		defer s.submissionsMutex.Unlock()

		// Apply filters
		filtered := []shared.Submission{}
		for _, sub := range s.submissions {
			// Filter by mode if specified
			if modeFilter != "" && string(sub.Mode) != modeFilter {
				continue
			}
			// Filter by type if specified
			if typeFilter != "" && string(sub.Type) != typeFilter {
				continue
			}
			// Filter by submitter if specified
			if submitterFilter != "" && sub.Submitter != submitterFilter {
				continue
			}
			filtered = append(filtered, sub)
		}

		total := len(filtered)
		results := []shared.Submission{}

		// Apply pagination
		start := skip
		if start > total {
			start = total
		}
		end := start + limit
		if end > total {
			end = total
		}

		if start < end {
			results = filtered[start:end]
		}

		resp := shared.SubmissionsResult{
			Session: s.session.Unix(),
			Total:   total,
			Results: results,
		}
		util.WriteJsonResponse(w, resp)
	}
	return http.HandlerFunc(f)
}

// newV2SubmissionsPostHandler handles POST requests for v2/submissions
func newV2SubmissionsPostHandler(s *SubmissionsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			util.WriteHttpMethodNotAllowed(w)
			return
		}

		// Read and parse submission from body
		defer req.Body.Close()
		var submission shared.Submission
		if err := json.NewDecoder(req.Body).Decode(&submission); err != nil {
			log.Printf("Invalid submission JSON: %v", err)
			util.WriteHttpBadRequest(w)
			return
		}

		// Validate submission
		if submission.Id.IsZero() {
			util.WriteJsonResponse(w, OperationResult{Status: "error", Message: "Invalid or missing ID"})
			return
		}

		// Handle different submission types
		switch submission.Type {
		case shared.TypeProgram:
			switch submission.Mode {
			case shared.ModeAdd, shared.ModeUpdate:
				if submission.Content == "" {
					util.WriteJsonResponse(w, OperationResult{Status: "error", Message: "Missing content"})
					return
				}
			case shared.ModeRemove:
				// Removal: content can be empty
			default:
				util.WriteJsonResponse(w, OperationResult{Status: "error", Message: "Unsupported submission mode for programs"})
				return
			}
			// Use unified check and submit functions
			if ok, res := s.checkSubmit(submission); !ok {
				util.WriteJsonResponse(w, res)
				return
			}
			res := s.doSubmit(submission)
			util.WriteJsonResponse(w, res)
		case shared.TypeSequence:
			// Only refresh mode is allowed for sequences (already validated in UnmarshalJSON)
			res := s.refreshSequence(submission)
			if res.Status == "success" {
				// Record submission if refresh was successful
				s.doSubmit(submission)
			}
			util.WriteJsonResponse(w, res)
		default:
			util.WriteJsonResponse(w, OperationResult{Status: "error", Message: "Unsupported submission type"})
			return
		}
	}
	return http.HandlerFunc(f)
}

func (s *SubmissionsServer) StopCrawler() {
	log.Print("Stopping crawler")
	s.crawlerStopped <- true
	restartTimer := time.NewTimer(s.crawlerRestartPause)
	go func() {
		<-restartTimer.C
		s.StartCrawler()
	}()
}

// filterValidKeywordsFields filters out unknown keywords from fields with key 'K'.
func filterValidKeywordsFields(fields []shared.Field) []shared.Field {
	filteredFields := make([]shared.Field, 0, len(fields))
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

func (s *SubmissionsServer) StartCrawler() {
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
func (s *SubmissionsServer) handleCrawlerTick() {
	if s.crawler.NumFetched() > 0 {
		// Regularly flush the lists
		if s.crawler.NumFetched()%s.crawlerFlushInterval == 0 {
			for _, l := range s.lists {
				err := l.Flush()
				if err != nil {
					log.Printf("Error flushing list %s: %v", l.Name(), err)
					s.StopCrawler()
					continue
				}
			}
		}
		// Regularly re-initialize the crawler
		if s.crawler.NumFetched()%s.crawlerReinitInterval == 0 {
			err := s.crawler.Init()
			if err != nil {
				log.Printf("Error re-initializing crawler: %v", err)
				s.StopCrawler()
				return
			}
		}
	}
	if s.crawler.NumFetched()%s.crawlerIdsCacheSize == 0 && rand.Float64() < s.crawlerIdsFetchRatio {
		// Find the missing ids
		for _, l := range s.lists {
			if l.Name() == "offsets" {
				ids, _, err := l.FindMissingIds(s.crawler.MaxId(), s.crawlerIdsCacheSize)
				if err != nil {
					s.StopCrawler()
					return
				}
				s.crawler.SetNextIds(ids)
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

// newV2SubmissionsCheckpointPostHandler handles POST requests for v2/submissions/checkpoint
func newV2SubmissionsCheckpointPostHandler(s *SubmissionsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		if err := s.writeCheckpoint(); err != nil {
			log.Printf("Checkpoint failed: %v", err)
			util.WriteJsonResponse(w, OperationResult{Status: "error", Message: "Checkpoint failed"})
			return
		}
		util.WriteJsonResponse(w, OperationResult{Status: "success", Message: "Checkpoint created"})
	}
	return http.HandlerFunc(f)
}

func (s *SubmissionsServer) Run(port int) {
	s.loadCheckpoint()

	// schedule background tasks
	checkpointTicker := time.NewTicker(CheckpointInterval)
	defer checkpointTicker.Stop()
	go func() {
		for range checkpointTicker.C {
			s.publishMetrics()
			s.clearUserStats()
			s.writeCheckpoint()
		}
	}()

	// start web server
	router := mux.NewRouter()
	router.Handle("/v2/submissions", newV2SubmissionsGetHandler(s)).Methods(http.MethodGet)
	router.Handle("/v2/submissions", newV2SubmissionsPostHandler(s)).Methods(http.MethodPost)
	router.Handle("/v2/submissions/checkpoint", newV2SubmissionsCheckpointPostHandler(s)).Methods(http.MethodPost)
	router.NotFoundHandler = http.HandlerFunc(util.HandleNotFound)
	log.Printf("Listening on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), util.CORSHandler(router))
}

func main() {
	setup := cmd.GetSetup("submissions")
	util.MustDirExist(setup.DataDir)
	oeisDir := filepath.Join(setup.DataDir, "seqs", "oeis")
	os.MkdirAll(oeisDir, os.ModePerm)
	u, p := util.ParseAuthInfo(setup.InfluxDbAuth)
	i := util.NewInfluxDbClient(setup.InfluxDbHost, u, p)
	s := NewSubmissionsServer(setup.DataDir, oeisDir, i)
	s.StartCrawler()
	s.Run(8084)
}
