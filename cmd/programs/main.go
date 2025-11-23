package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/loda-lang/loda-api/cmd"
	"github.com/loda-lang/loda-api/shared"
	"github.com/loda-lang/loda-api/util"
)

const (
	NumSubmissionsLow     = 1000
	NumSubmissionsHigh    = 2000
	NumSubmissionsMax     = 50000
	NumSubmissionsPerUser = 100
	MaxProgramLength      = 100000
	MaxNumParallelEval    = 10
	NumTermsCheck         = 8
	CheckpointInterval    = 10 * time.Minute
	UpdateInterval        = 24 * time.Hour
	CheckSessionInterval  = 24 * time.Hour
	CheckpointFile        = "checkpoint.json"
	CheckpointFileLegacy  = "checkpoint.txt"
	ProgramSeparator      = "=============================="
)

type ProgramsServer struct {
	dataDir               string
	influxDbClient        *util.InfluxDbClient
	lodaTool              *LODATool
	session               time.Time
	dataIndex             *shared.DataIndex
	submissions           []shared.Submission // Unified submissions (v1 and v2)
	submissionsPerProfile map[string]int
	submissionsPerUser    map[string]int
	dataIndexMutex        sync.Mutex
	submissionsMutex      sync.Mutex
	updateMutex           sync.Mutex
}

func NewProgramsServer(dataDir string, influxDbClient *util.InfluxDbClient, lodaTool *LODATool) *ProgramsServer {
	return &ProgramsServer{
		dataDir:               dataDir,
		influxDbClient:        influxDbClient,
		lodaTool:              lodaTool,
		session:               time.Now(),
		submissions:           []shared.Submission{},
		submissionsPerProfile: make(map[string]int),
		submissionsPerUser:    make(map[string]int),
	}
}

func newCountHandler(s *ProgramsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		s.submissionsMutex.Lock()
		defer s.submissionsMutex.Unlock()
		util.WriteHttpOK(w, fmt.Sprint(len(s.submissions)))
	}
	return http.HandlerFunc(f)
}

func newSessionHandler(s *ProgramsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		s.submissionsMutex.Lock()
		defer s.submissionsMutex.Unlock()
		s.checkSession()
		util.WriteHttpOK(w, fmt.Sprint(s.session.Unix()))
	}
	return http.HandlerFunc(f)
}

// Returns (ok, OperationResult)
func (s *ProgramsServer) checkSubmit(submission shared.Submission) (bool, OperationResult) {
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
	for _, p := range s.submissions {
		if slices.Equal(p.Operations, submission.Operations) {
			return false, OperationResult{Status: "error", Message: "Duplicate submission"}
		}
	}
	return true, OperationResult{}
}

func (s *ProgramsServer) doSubmit(submission shared.Submission) OperationResult {
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

func newPostHandler(s *ProgramsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		program, ok := readProgramFromBody(w, req)
		if !ok {
			return
		}
		// Convert Program to Submission
		submission := shared.NewSubmissionFromProgram(program)
		if ok, res := s.checkSubmit(submission); !ok {
			// Convert OperationResult to EvalResult for v1 API
			util.WriteJsonResponse(w, EvalResult{Status: res.Status, Message: res.Message, Terms: nil})
			return
		}
		res := s.doSubmit(submission)
		// Convert OperationResult to EvalResult for v1 API
		util.WriteJsonResponse(w, EvalResult{Status: res.Status, Message: res.Message, Terms: nil})
	}
	return http.HandlerFunc(f)
}

func newGetHandler(s *ProgramsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		// check request
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		params := mux.Vars(req)
		index, _ := strconv.Atoi(params["index"])

		// main work
		s.submissionsMutex.Lock()
		defer s.submissionsMutex.Unlock()
		s.checkSession()
		if index < 0 || index >= len(s.submissions) {
			util.WriteHttpNotFound(w)
			return
		}
		util.WriteHttpOK(w, s.submissions[index].Content)
	}
	return http.HandlerFunc(f)
}

func newCheckpointHandler(s *ProgramsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		// check request
		if req.Method != http.MethodPost {
			util.WriteHttpMethodNotAllowed(w)
			return
		}

		// main work
		err := s.writeCheckpoint()
		if err != nil {
			log.Print(err)
			util.WriteHttpInternalServerError(w)
		} else {
			msg := "Checkpoint created"
			util.WriteHttpCreated(w, msg)
			log.Print(msg)
		}
	}
	return http.HandlerFunc(f)
}

func newProgramByIdHandler(s *ProgramsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
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
		idx := s.getDataIndex()
		p := shared.FindProgramById(idx.Programs, uid)
		if p == nil {
			log.Printf("Program ID not found: %v", uid.String())
			w.WriteHeader(http.StatusNotFound)
			return
		}
		path, err := p.GetPath(filepath.Join(s.dataDir, "programs", "oeis"))
		if err != nil {
			util.WriteHttpInternalServerError(w)
			return
		}
		code, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Program file not found: %v", path)
			util.WriteHttpNotFound(w)
			return
		}
		p.SetCode(string(code))
		util.WriteJsonResponse(w, p)
	}
	return http.HandlerFunc(f)
}

func newProgramSearchHandler(s *ProgramsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		q := req.URL.Query().Get("q")
		limit, skip, shuffle := util.ParseLimitSkipShuffle(req, 10, 100)
		idx := s.getDataIndex()
		results, total := shared.SearchPrograms(idx, q, limit, skip, shuffle)
		resp := shared.SearchResult{
			Total: total,
		}
		for _, prog := range results {
			resp.Results = append(resp.Results, shared.SearchItem{
				Id:       prog.Id.String(),
				Name:     prog.Name,
				Keywords: shared.DecodeKeywords(prog.Keywords),
			})
		}
		util.WriteJsonResponse(w, resp)
	}
	return http.HandlerFunc(f)
}

func readProgramFromBody(w http.ResponseWriter, req *http.Request) (shared.Program, bool) {
	var p shared.Program
	if req.Method != http.MethodPost {
		util.WriteHttpMethodNotAllowed(w)
		return p, false
	}
	if req.ContentLength <= 0 || req.ContentLength > MaxProgramLength {
		util.WriteHttpBadRequest(w)
		return p, false
	}
	// Read program code from body
	defer req.Body.Close()
	content, err := io.ReadAll(req.Body)
	if err != nil || len(content) == 0 {
		util.WriteHttpBadRequest(w)
		return p, false
	}
	code := strings.TrimSpace(string(content))
	if len(code) == 0 {
		util.WriteHttpBadRequest(w)
		return p, false
	}
	code = strings.ReplaceAll(code, "\r\n", "\n") + "\n"
	p, err = shared.NewProgramFromCode(code)
	if err != nil {
		log.Printf("Invalid program: %v", err)
		util.WriteHttpBadRequest(w)
		return p, false
	}
	if len(p.Operations) == 0 {
		log.Printf("Invalid program (no operations): %s", code)
		util.WriteHttpBadRequest(w)
		return p, false
	}
	return p, true
}

func logProgramAction(action string, p *shared.Program) {
	msg := action + " program "
	if !p.Id.IsZero() {
		msg += p.Id.String()
	} else {
		msg += fmt.Sprintf("with %d operations", len(p.Operations))
	}
	log.Print(msg)
}

func newProgramEvalHandler(s *ProgramsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		p, ok := readProgramFromBody(w, req)
		if !ok {
			return
		}
		// Parse query params
		numTerms := 8
		if t := req.URL.Query().Get("t"); t != "" {
			if v, err := strconv.Atoi(t); err == nil && v > 0 && v <= 10000 {
				numTerms = v
			} else {
				util.WriteHttpBadRequest(w)
				return
			}
		}
		if o := req.URL.Query().Get("o"); o != "" {
			if v, err := strconv.Atoi(o); err == nil {
				p.SetOffset(v)
			} else {
				util.WriteHttpBadRequest(w)
				return
			}
		}
		logProgramAction("Evaluating", &p)

		// Call LODA tool and get result object
		result := s.lodaTool.Eval(p, numTerms)
		if result.Status == "error" {
			log.Printf("Evaluation failed: %v", result.Message)
		}
		util.WriteJsonResponse(w, result)
	}
	return http.HandlerFunc(f)
}

func newProgramExportHandler(s *ProgramsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		p, ok := readProgramFromBody(w, req)
		if !ok {
			return
		}
		// Parse format query param
		format := req.URL.Query().Get("format")
		if format == "" {
			format = "loda"
		}
		logProgramAction("Exporting", &p)

		// Call LODA tool and get result object
		result := s.lodaTool.Export(p, format)
		if result.Status == "error" {
			log.Printf("Export failed: %v", result.Message)
		}
		util.WriteJsonResponse(w, result)
	}
	return http.HandlerFunc(f)
}

func (s *ProgramsServer) writeCheckpoint() error {
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

func (s *ProgramsServer) checkSession() {
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

func (s *ProgramsServer) publishMetrics() {
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

func (s *ProgramsServer) clearUserStats() {
	s.submissionsMutex.Lock()
	defer s.submissionsMutex.Unlock()
	s.submissionsPerUser = make(map[string]int)
}

func (s *ProgramsServer) update() {
	// Reset data index to free memory
	s.resetDataIndex()

	// Check available system memory, skip update if less than 500 MB (Linux only)
	const minMemKB = 500 * 1024 // 500 MB
	freeMemKB := util.GetFreeMemoryKB()
	if freeMemKB > 0 && freeMemKB < minMemKB {
		log.Printf("Skipping update: only %d MB memory available", freeMemKB/1024)
		return
	}
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	if err := s.lodaTool.Install(); err != nil {
		log.Fatalf("LODA tool installation failed: %v", err)
	}
	if _, err := s.lodaTool.Exec(0, "update"); err != nil {
		log.Printf("LODA tool update failed: %v", err)
	}

	// Reset data index again to force reload using the new data
	s.resetDataIndex()
}

func (s *ProgramsServer) resetDataIndex() {
	s.dataIndexMutex.Lock()
	s.dataIndex = nil
	s.dataIndexMutex.Unlock()
	runtime.GC()
}

// getDataIndex loads the dataIndex on demand, thread-safe
func (s *ProgramsServer) getDataIndex() *shared.DataIndex {
	s.dataIndexMutex.Lock()
	defer s.dataIndexMutex.Unlock()
	if s.dataIndex == nil {
		idx := shared.NewDataIndex(s.dataDir)
		err := idx.Load()
		if err != nil {
			log.Fatalf("Failed to load data index: %v", err)
		}
		s.dataIndex = idx
		runtime.GC()
	}
	return s.dataIndex
}

func (s *ProgramsServer) loadCheckpoint() {
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

func (s *ProgramsServer) loadCheckpointLegacy() {
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
func newV2SubmissionsGetHandler(s *ProgramsServer) http.Handler {
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
func newV2SubmissionsPostHandler(s *ProgramsServer) http.Handler {
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

		// For now, only support programs
		if submission.Type == shared.TypeProgram {
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
		} else {
			util.WriteJsonResponse(w, OperationResult{Status: "error", Message: "Unsupported submission type"})
			return
		}

		// Use unified check and submit functions
		if ok, res := s.checkSubmit(submission); !ok {
			util.WriteJsonResponse(w, res)
			return
		}

		res := s.doSubmit(submission)
		util.WriteJsonResponse(w, res)
	}
	return http.HandlerFunc(f)
}

func (s *ProgramsServer) Run(port int) {
	// ensure that loda is installed
	if err := s.lodaTool.Install(); err != nil {
		log.Fatalf("LODA tool installation failed: %v", err)
	}

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
	updateTicker := time.NewTicker(UpdateInterval)
	defer updateTicker.Stop()
	go func() {
		for range updateTicker.C {
			s.update()
		}
	}()

	// start web server
	router := mux.NewRouter()
	router.Handle("/v1/count", newCountHandler(s))
	router.Handle("/v1/session", newSessionHandler(s))
	postHandler := newPostHandler(s)
	router.Handle("/v1/programs", postHandler)
	router.Handle("/v1/programs/", postHandler)
	router.Handle("/v1/programs/{index:[0-9]+}", newGetHandler(s))
	router.Handle("/v1/checkpoint", newCheckpointHandler(s))
	router.Handle("/v2/programs/{id:[A-Z][0-9]+}", newProgramByIdHandler(s))
	router.Handle("/v2/programs/search", newProgramSearchHandler(s))
	router.Handle("/v2/programs/eval", newProgramEvalHandler(s))
	router.Handle("/v2/programs/export", newProgramExportHandler(s))
	router.Handle("/v2/submissions", newV2SubmissionsGetHandler(s)).Methods(http.MethodGet)
	router.Handle("/v2/submissions", newV2SubmissionsPostHandler(s)).Methods(http.MethodPost)
	router.NotFoundHandler = http.HandlerFunc(util.HandleNotFound)
	log.Printf("Listening on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), util.CORSHandler(router))
}

func main() {
	setup := cmd.GetSetup("programs")
	u, p := util.ParseAuthInfo(setup.InfluxDbAuth)
	i := util.NewInfluxDbClient(setup.InfluxDbHost, u, p)
	t := NewLODATool(setup.DataDir, MaxNumParallelEval)
	s := NewProgramsServer(setup.DataDir, i, t)
	s.Run(8081)
}
