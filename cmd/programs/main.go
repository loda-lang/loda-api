package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	CheckpointFile        = "checkpoint.txt"
	ProgramSeparator      = "=============================="
)

type ProgramsServer struct {
	dataDir               string
	influxDbClient        *util.InfluxDbClient
	lodaTool              *LODATool
	session               time.Time
	programs              []shared.Program
	submitters            []*shared.Submitter
	submissions           []shared.Program
	submissionsPerProfile map[string]int
	submissionsPerUser    map[string]int
	dataMutex             sync.Mutex
	updateMutex           sync.Mutex
}

func NewProgramsServer(dataDir string, influxDbClient *util.InfluxDbClient, lodaTool *LODATool) *ProgramsServer {
	return &ProgramsServer{
		dataDir:               dataDir,
		influxDbClient:        influxDbClient,
		lodaTool:              lodaTool,
		session:               time.Now(),
		submissions:           []shared.Program{},
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
		s.dataMutex.Lock()
		defer s.dataMutex.Unlock()
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
		s.dataMutex.Lock()
		defer s.dataMutex.Unlock()
		s.checkSession()
		util.WriteHttpOK(w, fmt.Sprint(s.session.Unix()))
	}
	return http.HandlerFunc(f)
}

func (s *ProgramsServer) checkSubmit(program shared.Program, w http.ResponseWriter) bool {
	submitter := ""
	if program.Submitter != nil {
		submitter = program.Submitter.Name
	}
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.checkSession()
	if len(s.submissions) > NumSubmissionsMax {
		log.Print("Maximum number of submissions exceeded")
		util.WriteHttpInternalServerError(w)
		return false
	}
	if s.submissionsPerUser[submitter] >= NumSubmissionsPerUser {
		log.Printf("Rejected program from %s", submitter)
		util.WriteHttpTooManyRequests(w)
		return false
	}
	for _, p := range s.submissions {
		if slices.Equal(p.Operations, program.Operations) {
			util.WriteHttpOK(w, "Duplicate submission")
			return false
		}
	}
	return true
}

func (s *ProgramsServer) doSubmit(program shared.Program, w http.ResponseWriter) {
	submitter := ""
	if program.Submitter != nil {
		submitter = program.Submitter.Name
	}
	profile := program.GetMinerProfile()
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.submissions = append(s.submissions, program)
	s.submissionsPerUser[submitter]++
	s.submissionsPerProfile[profile]++
	util.WriteHttpCreated(w, "Accepted submission")
	msg := fmt.Sprintf("Accepted submission from %s (%d/%d)",
		submitter, s.submissionsPerUser[submitter], NumSubmissionsPerUser)
	if len(profile) > 0 {
		msg += fmt.Sprintf("; profile %s (%d)", profile, s.submissionsPerProfile[profile])
	}
	log.Print(msg)
}

func newPostHandler(s *ProgramsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		program, ok := readProgramFromBody(w, req)
		if !ok {
			return
		}
		ok = s.checkSubmit(program, w)
		if ok {
			s.doSubmit(program, w)
		}
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
		s.dataMutex.Lock()
		defer s.dataMutex.Unlock()
		s.checkSession()
		if index < 0 || index >= len(s.submissions) {
			util.WriteHttpNotFound(w)
			return
		}
		util.WriteHttpOK(w, s.submissions[index].Code)
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
		s.dataMutex.Lock()
		defer s.dataMutex.Unlock()
		p := shared.FindById(s.programs, uid)
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
		limit, skip := util.ParseLimitSkip(req, 10, 100)
		s.dataMutex.Lock()
		defer s.dataMutex.Unlock()
		results, total := shared.Search(s.programs, q, limit, skip)
		type IDAndName struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		}
		var resp struct {
			Total   int         `json:"total"`
			Results []IDAndName `json:"results"`
		}
		resp.Total = total
		for _, prog := range results {
			resp.Results = append(resp.Results, IDAndName{Id: prog.Id.String(), Name: prog.Name})
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
	if err != nil || len(p.Operations) == 0 {
		log.Printf("Invalid program: %v", err)
		util.WriteHttpBadRequest(w)
		return p, false
	}
	return p, true
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
		msg := "Evaluating program "
		if !p.Id.IsZero() {
			msg += p.Id.String()
		} else {
			msg += fmt.Sprintf("with %d operations", len(p.Operations))
		}
		log.Print(msg)
		terms, err := s.lodaTool.Eval(p, numTerms)
		if err != nil {
			log.Printf("Evaluation failed: %v", err)
			util.WriteHttpBadRequest(w)
			return
		}
		util.WriteJsonResponse(w, map[string]interface{}{"terms": terms})
	}
	return http.HandlerFunc(f)
}

func newSubmitHandler(s *ProgramsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		idStr := params["id"]
		id, err := util.NewUIDFromString(idStr)
		if err != nil || id.IsZero() {
			util.WriteHttpBadRequest(w)
			return
		}
		program, ok := readProgramFromBody(w, req)
		if !ok {
			return
		}
		submitter := program.Submitter
		if s := req.URL.Query().Get("submitter"); s != "" {
			submitter = &shared.Submitter{Name: s}
		}
		ok = s.checkSubmit(program, w)
		if !ok {
			return
		}
		index, err := s.loadSequences()
		if err != nil {
			util.WriteHttpInternalServerError(w)
			return
		}
		seq := index.FindById(id)
		if seq == nil {
			util.WriteHttpNotFound(w)
			return
		}
		program.SetIdAndName(id, seq.Name)
		program.SetSubmitter(submitter)

		// Check that the program produces the expected terms
		expectedTerms := seq.TermsList()
		if len(expectedTerms) > NumTermsCheck {
			expectedTerms = expectedTerms[:NumTermsCheck]
		}
		log.Printf("Checking program %v", program.Id)
		evalTerms, err := s.lodaTool.Eval(program, NumTermsCheck)
		if err != nil {
			log.Printf("Evaluation failed: %v", err)
			util.WriteHttpBadRequest(w)
			return
		}
		if !slices.Equal(expectedTerms, evalTerms) {
			log.Printf("Program for %v produced incorrect terms; expected: %v, got: %v",
				id.String(), expectedTerms, evalTerms)
			util.WriteHttpBadRequest(w)
			return
		}
		s.doSubmit(program, w)
	}
	return http.HandlerFunc(f)
}

func (s *ProgramsServer) writeCheckpoint() error {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	f, err := os.Create(filepath.Join(s.dataDir, CheckpointFile))
	if err != nil {
		return fmt.Errorf("cannot opening checkpoint file: %v", err)
	}
	defer f.Close()
	for _, p := range s.submissions {
		_, err = f.WriteString(fmt.Sprintf("%s%s\n", p.Code, ProgramSeparator))
		if err != nil {
			return fmt.Errorf("cannot write to checkpoint file: %v", err)
		}
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
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	totalCount := 0
	for profile, count := range s.submissionsPerProfile {
		labels := map[string]string{"kind": "submitted", "profile": profile}
		s.influxDbClient.Write("programs", labels, count)
		totalCount += count
	}
	log.Printf("Published %d new submissions to InfluxDB", totalCount)
	s.submissionsPerProfile = make(map[string]int)
}

func (s *ProgramsServer) clearUserStats() {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.submissionsPerUser = make(map[string]int)
}

func (s *ProgramsServer) update() {
	{
		s.updateMutex.Lock()
		defer s.updateMutex.Unlock()
		if err := s.lodaTool.Install(); err != nil {
			log.Fatalf("LODA tool installation failed: %v", err)
		}
		if _, err := s.lodaTool.Exec(0, "update"); err != nil {
			log.Printf("LODA tool update failed: %v", err)
		}
	}
	s.loadPrograms()
}

func (s *ProgramsServer) loadSequences() (*shared.SequenceIndex, error) {
	oeisDir := filepath.Join(s.dataDir, "seqs", "oeis")
	index := shared.NewSequenceIndex()
	err := index.Load(oeisDir)
	if err != nil {
		log.Printf("Error loading sequences: %v", err)
	} else {
		log.Printf("Loaded %d sequences", len(index.Sequences))
	}
	return index, err
}

// loadPrograms loads submitters and programs from the stats directory
func (s *ProgramsServer) loadPrograms() {
	statsDir := filepath.Join(s.dataDir, "stats")
	submittersPath := filepath.Join(statsDir, "submitters.csv")
	programsPath := filepath.Join(statsDir, "programs.csv")

	submitters, err := shared.LoadSubmittersCSV(submittersPath)
	if err != nil {
		log.Printf("Failed to load submitters: %v", err)
		return
	}
	index, err := s.loadSequences()
	if err != nil {
		log.Printf("Failed to load sequences: %v", err)
		return
	}
	programs, err := shared.LoadProgramsCSV(programsPath, submitters, index)
	if err != nil {
		log.Printf("Failed to load programs: %v", err)
		return
	}

	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.submitters = submitters
	s.programs = programs
	log.Printf("Loaded %d submitters and %d programs", len(submitters), len(programs))
}

func (s *ProgramsServer) loadCheckpoint() {
	checkpointPath := filepath.Join(s.dataDir, CheckpointFile)
	file, err := os.Open(checkpointPath)
	if err != nil {
		log.Printf("Cannot load checkpoint %s", checkpointPath)
		return
	}
	log.Printf("Loading checkpoint %s", checkpointPath)
	s.submissions = []shared.Program{}
	scanner := bufio.NewScanner(file)
	program := ""
	for scanner.Scan() {
		line := scanner.Text()
		if line == ProgramSeparator {
			if len(program) > 0 {
				p, err := shared.NewProgramFromCode(program)
				if err == nil && len(p.Operations) > 0 {
					s.submissions = append(s.submissions, p)
				} else {
					log.Printf("Invalid program in checkpoint: %v", err)
				}
			}
			program = ""
		} else {
			program = program + line + "\n"
		}
	}
	log.Printf("Loaded %v submissions from checkpoint", len(s.submissions))
}

func (s *ProgramsServer) Run(port int) {
	// load programs and checkpoint
	s.loadPrograms()
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

	// perform update in background on startup
	go s.update()

	// start web server
	router := mux.NewRouter()
	router.Handle("/v1/count", newCountHandler(s))
	router.Handle("/v1/session", newSessionHandler(s))
	postHandler := newPostHandler(s)
	router.Handle("/v1/programs", postHandler)
	router.Handle("/v1/programs/", postHandler)
	router.Handle("/v1/programs/{index:[0-9]+}", newGetHandler(s))
	router.Handle("/v1/checkpoint", newCheckpointHandler(s))
	router.Handle("/v2/programs/{id:[A-Z][0-9]+}/submit", newSubmitHandler(s))
	router.Handle("/v2/programs/{id:[A-Z][0-9]+}", newProgramByIdHandler(s))
	router.Handle("/v2/programs/search", newProgramSearchHandler(s))
	router.Handle("/v2/programs/eval", newProgramEvalHandler(s))
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
