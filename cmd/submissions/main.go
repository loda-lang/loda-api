package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/loda-lang/loda-api/cmd"
	"github.com/loda-lang/loda-api/shared"
	"github.com/loda-lang/loda-api/util"
)

const (
	NumSubmissionsLow       = 1000
	NumSubmissionsHigh      = 2000
	NumSubmissionsMax       = 50000
	NumSubmissionsPerUser   = 100
	MaxProgramLength        = 100000
	CheckpointInterval      = 10 * time.Minute
	CheckSessionInterval    = 24 * time.Hour
	BFileProtectionDuration = 24 * time.Hour
	CheckpointFile          = "checkpoint.json"
	CheckpointFileLegacy    = "checkpoint.txt"
	ProgramSeparator        = "=============================="
)

type OperationResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type SubmissionsServer struct {
	dataDir               string
	influxDbClient        *util.InfluxDbClient
	session               time.Time
	submissions           []shared.Submission // Unified submissions (v1 and v2)
	submissionsPerProfile map[string]int
	submissionsPerUser    map[string]int
	bfileRemovals         map[string]time.Time // Tracks b-file removal times for 24h protection
	submissionsMutex      sync.Mutex
	bfileRemovalsMutex    sync.Mutex
}

func NewSubmissionsServer(dataDir string, influxDbClient *util.InfluxDbClient) *SubmissionsServer {
	return &SubmissionsServer{
		dataDir:               dataDir,
		influxDbClient:        influxDbClient,
		session:               time.Now(),
		submissions:           []shared.Submission{},
		submissionsPerProfile: make(map[string]int),
		submissionsPerUser:    make(map[string]int),
		bfileRemovals:         make(map[string]time.Time),
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

// removeBFile removes a b-file and returns an OperationResult.
// B-files are protected for 24 hours after removal.
func (s *SubmissionsServer) removeBFile(submission shared.Submission) OperationResult {
	idStr := submission.Id.String()

	// Check 24h protection
	s.bfileRemovalsMutex.Lock()
	if lastRemoval, exists := s.bfileRemovals[idStr]; exists {
		if time.Since(lastRemoval) < BFileProtectionDuration {
			s.bfileRemovalsMutex.Unlock()
			remaining := BFileProtectionDuration - time.Since(lastRemoval)
			protectionMsg := fmt.Sprintf("B-file is protected for %.0f more hours", remaining.Hours())
			log.Printf("%s: %s", protectionMsg, idStr)
			return OperationResult{Status: "error", Message: protectionMsg}
		}
	}
	s.bfileRemovalsMutex.Unlock()

	// Get the b-file path (ID format already validated by NewUIDFromString in submission)
	bfilePath := s.getBFilePath(submission.Id)

	// Check if the file exists
	if !util.FileExists(bfilePath) {
		log.Printf("B-file does not exist: %s", bfilePath)
		return OperationResult{Status: "error", Message: "B-file does not exist"}
	}

	// Remove the file
	if err := os.Remove(bfilePath); err != nil {
		log.Printf("Failed to remove b-file %s: %v", bfilePath, err)
		return OperationResult{Status: "error", Message: "Failed to remove b-file"}
	}

	// Record the removal time for 24h protection
	s.bfileRemovalsMutex.Lock()
	s.bfileRemovals[idStr] = time.Now()
	s.bfileRemovalsMutex.Unlock()

	log.Printf("Removed b-file %s by %s", idStr, submission.Submitter)
	return OperationResult{Status: "success", Message: "B-file removed"}
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
		// Try loading legacy format
		log.Printf("Cannot load JSON checkpoint %s, attempting legacy format", checkpointPath)
		s.loadCheckpointLegacy()
		return
	}
	defer file.Close()
	log.Printf("Loading checkpoint %s", checkpointPath)
	s.submissions = []shared.Submission{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&s.submissions); err != nil {
		log.Printf("Cannot decode checkpoint JSON: %v, trying legacy format", err)
		s.loadCheckpointLegacy()
		return
	}
	log.Printf("Loaded %v submissions from checkpoint", len(s.submissions))
}

func (s *SubmissionsServer) loadCheckpointLegacy() {
	checkpointPath := filepath.Join(s.dataDir, CheckpointFileLegacy)
	file, err := os.Open(checkpointPath)
	if err != nil {
		log.Printf("Cannot load checkpoint %s", checkpointPath)
		return
	}
	defer file.Close()
	log.Printf("Loading legacy checkpoint %s", checkpointPath)
	s.submissions = []shared.Submission{}
	scanner := bufio.NewScanner(file)
	program := ""
	for scanner.Scan() {
		line := scanner.Text()
		if line == ProgramSeparator {
			if len(program) > 0 {
				sub, err := shared.NewSubmissionFromCode(program)
				if err == nil && len(sub.Operations) > 0 {
					s.submissions = append(s.submissions, sub)
				} else {
					log.Printf("Invalid program in checkpoint: %v", err)
				}
			}
			program = ""
		} else {
			program = program + line + "\n"
		}
	}
	log.Printf("Loaded %v submissions from legacy checkpoint", len(s.submissions))
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
		case shared.TypeBFile:
			// Only remove mode is allowed for b-files (already validated in UnmarshalJSON)
			if ok, res := s.checkSubmit(submission); !ok {
				util.WriteJsonResponse(w, res)
				return
			}
			res := s.removeBFile(submission)
			if res.Status == "success" {
				// Only record submission if b-file removal succeeded
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
	router.NotFoundHandler = http.HandlerFunc(util.HandleNotFound)
	log.Printf("Listening on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), util.CORSHandler(router))
}

func main() {
	setup := cmd.GetSetup("submissions")
	u, p := util.ParseAuthInfo(setup.InfluxDbAuth)
	i := util.NewInfluxDbClient(setup.InfluxDbHost, u, p)
	s := NewSubmissionsServer(setup.DataDir, i)
	s.Run(8084)
}
