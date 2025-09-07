package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/loda-lang/loda-api/cmd"
	"github.com/loda-lang/loda-api/util"
)

const (
	NumProgramsLow       = 1000
	NumProgramsHigh      = 2000
	NumProgramsMax       = 50000
	NumProgramsPerUser   = 100
	MaxProgramLength     = 100000
	CheckpointInterval   = 10 * time.Minute
	UpdateInterval       = 24 * time.Hour
	CheckSessionInterval = 24 * time.Hour
	ProfilePrefix        = "; Miner Profile:"
	SubmittedByPrefix    = "; Submitted by "
	CheckpointFile       = "checkpoint.txt"
	ProgramSeparator     = "=============================="
)

type ProgramsServer struct {
	dataDir                string
	influxDbClient         *util.InfluxDbClient
	lodaTool               *LODATool
	session                time.Time
	programs               []string
	submisstionsPerProfile map[string]int
	submisstionsPerUser    map[string]int
	dataMutex              sync.Mutex
	updateMutex            sync.Mutex
}

func NewProgramsServer(dataDir string, influxDbClient *util.InfluxDbClient, lodaTool *LODATool) *ProgramsServer {
	return &ProgramsServer{
		dataDir:                dataDir,
		influxDbClient:         influxDbClient,
		lodaTool:               lodaTool,
		session:                time.Now(),
		programs:               []string{},
		submisstionsPerProfile: make(map[string]int),
		submisstionsPerUser:    make(map[string]int),
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
		util.WriteHttpOK(w, fmt.Sprint(len(s.programs)))
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

func newPostHandler(s *ProgramsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		// check request
		if req.Method != http.MethodPost {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		if req.ContentLength <= 0 || req.ContentLength > MaxProgramLength {
			util.WriteHttpBadRequest(w)
			return
		}
		defer req.Body.Close()
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			util.WriteHttpInternalServerError(w)
			return
		}
		program := strings.TrimSpace(string(body))
		if len(program) == 0 {
			util.WriteHttpBadRequest(w)
			return
		}
		program = strings.ReplaceAll(program, "\r\n", "\n") + "\n"
		profile := "unknown"
		user := "unknown"
		lines := strings.Split(program, "\n")
		for _, l := range lines {
			if strings.HasPrefix(l, ProfilePrefix) {
				profile = strings.TrimSpace(l[len(ProfilePrefix):])
			}
			if strings.HasPrefix(l, SubmittedByPrefix) {
				user = strings.TrimSpace(l[len(SubmittedByPrefix):])
			}
		}

		// main work
		s.dataMutex.Lock()
		defer s.dataMutex.Unlock()
		s.checkSession()
		if len(s.programs) > NumProgramsMax {
			log.Print("Maximum number of programs exceeded")
			util.WriteHttpInternalServerError(w)
			return
		}
		if s.submisstionsPerUser[user] >= NumProgramsPerUser {
			log.Printf("Rejected program from %s, profile %s", user, profile)
			util.WriteHttpTooManyRequests(w)
			return
		}
		s.submisstionsPerUser[user]++
		for _, p := range s.programs {
			if p == program {
				util.WriteHttpOK(w, "Duplicate program")
				return
			}
		}
		s.programs = append(s.programs, program)
		s.submisstionsPerProfile[profile]++
		msg := fmt.Sprintf("Accepted program from %s, profile %s", user, profile)
		util.WriteHttpCreated(w, msg)
		log.Print(msg)
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
		if index < 0 || index >= len(s.programs) {
			util.WriteHttpNotFound(w)
			return
		}
		util.WriteHttpOK(w, s.programs[index])
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
			util.WriteHttpCreated(w, "Checkpoint created")
		}
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
	for _, p := range s.programs {
		_, err = f.WriteString(fmt.Sprintf("%s%s\n", p, ProgramSeparator))
		if err != nil {
			return fmt.Errorf("cannot write to checkpoint file: %v", err)
		}
	}
	return nil
}

func (s *ProgramsServer) checkSession() {
	if len(s.programs) < NumProgramsHigh {
		return
	}
	if time.Since(s.session).Minutes() < CheckSessionInterval.Minutes() {
		return
	}
	s.session = time.Now()
	log.Printf("Starting new session: %v", s.session)
	if len(s.programs) > NumProgramsLow {
		end := len(s.programs)
		start := end - NumProgramsLow
		s.programs = s.programs[start:end]
	}
}

func (s *ProgramsServer) publishMetrics() {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	for profile, count := range s.submisstionsPerProfile {
		labels := map[string]string{"kind": "submitted", "profile": profile}
		s.influxDbClient.Write("programs", labels, count)
	}
	s.submisstionsPerProfile = make(map[string]int)
}

func (s *ProgramsServer) clearUserStats() {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.submisstionsPerUser = make(map[string]int)
}

func (s *ProgramsServer) update() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	if err := s.lodaTool.Install(); err != nil {
		log.Fatalf("LODA tool installation failed: %v", err)
	}
	if _, err := s.lodaTool.Exec("update"); err != nil {
		log.Printf("LODA tool update failed: %v", err)
	}
}

func (s *ProgramsServer) lodaCheckpoint() {
	checkpointPath := filepath.Join(s.dataDir, CheckpointFile)
	file, err := os.Open(checkpointPath)
	if err != nil {
		log.Printf("Cannot load checkpoint %s", checkpointPath)
		return
	}
	log.Printf("Loading checkpoint %s", checkpointPath)
	s.programs = []string{}
	scanner := bufio.NewScanner(file)
	program := ""
	for scanner.Scan() {
		line := scanner.Text()
		if line == ProgramSeparator {
			if len(program) > 0 {
				s.programs = append(s.programs, program)
			}
			program = ""
		} else {
			program = program + line + "\n"
		}
	}
	log.Printf("Loaded %v programs from checkpoint", len(s.programs))
}

func (s *ProgramsServer) Run(port int) {
	// load checkpoint
	s.lodaCheckpoint()

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
	router.NotFoundHandler = http.HandlerFunc(util.HandleNotFound)
	log.Printf("Listening on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), router)
}

func main() {
	setup := cmd.GetSetup("programs")
	u, p := util.ParseAuthInfo(setup.InfluxDbAuth)
	i := util.NewInfluxDbClient(setup.InfluxDbHost, u, p)
	t := NewLODATool(setup.DataDir)
	s := NewProgramsServer(setup.DataDir, i, t)
	s.Run(8081)
}
