package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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
	MaxProgramLength   = 100000
	MaxNumParallelEval = 10
	NumTermsCheck      = 8
	UpdateInterval     = 24 * time.Hour
)

type ProgramsServer struct {
	dataDir        string
	influxDbClient *util.InfluxDbClient
	lodaTool       *LODATool
	dataIndex      *shared.DataIndex
	dataIndexMutex sync.Mutex
	updateMutex    sync.Mutex
}

func NewProgramsServer(dataDir string, influxDbClient *util.InfluxDbClient, lodaTool *LODATool) *ProgramsServer {
	return &ProgramsServer{
		dataDir:        dataDir,
		influxDbClient: influxDbClient,
		lodaTool:       lodaTool,
	}
}

<<<<<<< HEAD
=======
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

// getBFilePath returns the path to a b-file for the given sequence ID.
// The ID is validated using util.NewUIDFromString format (e.g., "A000045").
func (s *ProgramsServer) getBFilePath(id util.UID) string {
	idStr := id.String()
	numericId := idStr[1:] // e.g., "000045"
	dir := filepath.Join(s.dataDir, "seqs", "oeis", "b", numericId[0:3])
	filename := fmt.Sprintf("b%s.txt.gz", numericId)
	return filepath.Join(dir, filename)
}

// removeBFile removes a b-file and returns an OperationResult.
// B-files are protected for 24 hours after removal.
func (s *ProgramsServer) removeBFile(submission shared.Submission) OperationResult {
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

>>>>>>> main
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

func (s *ProgramsServer) Run(port int) {
	// ensure that loda is installed
	if err := s.lodaTool.Install(); err != nil {
		log.Fatalf("LODA tool installation failed: %v", err)
	}

	// schedule background tasks
	updateTicker := time.NewTicker(UpdateInterval)
	defer updateTicker.Stop()
	go func() {
		for range updateTicker.C {
			s.update()
		}
	}()

	// start web server
	router := mux.NewRouter()
	router.Handle("/v2/programs/{id:[A-Z][0-9]+}", newProgramByIdHandler(s))
	router.Handle("/v2/programs/search", newProgramSearchHandler(s))
	router.Handle("/v2/programs/eval", newProgramEvalHandler(s))
	router.Handle("/v2/programs/export", newProgramExportHandler(s))
<<<<<<< HEAD
=======
	router.Handle("/v2/submissions", newV2SubmissionsGetHandler(s)).Methods(http.MethodGet)
	router.Handle("/v2/submissions", newV2SubmissionsPostHandler(s)).Methods(http.MethodPost)
	router.Handle("/v2/submissions/checkpoint", newCheckpointHandler(s)).Methods(http.MethodPost)
>>>>>>> main
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
