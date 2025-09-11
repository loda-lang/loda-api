package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/loda-lang/loda-api/cmd"
	"github.com/loda-lang/loda-api/shared"
	"github.com/loda-lang/loda-api/util"
)

type StatsServer struct {
	dataDir            string
	openApiSpec        []byte
	summary            *Summary
	submitters         []*shared.Submitter
	influxDbClient     *util.InfluxDbClient
	cpuHours           int
	cpuHoursByPlatform map[string]map[string]int // platform -> version -> cpuHours
	mutex              sync.Mutex
}

func NewStatsServer(influxDbClient *util.InfluxDbClient, openApiSpec []byte, dataDir string) *StatsServer {
	return &StatsServer{
		dataDir:            dataDir,
		openApiSpec:        openApiSpec,
		influxDbClient:     influxDbClient,
		cpuHours:           0,
		cpuHoursByPlatform: make(map[string]map[string]int),
	}
}

func (s *StatsServer) loadSummary() {
	path := filepath.Join(s.dataDir, "stats", "summary.csv")
	summary, err := LoadSummaryCSV(path)
	if err != nil {
		log.Printf("Failed to load summary: %v", err)
		s.summary = nil
	} else {
		log.Printf("Loaded summary: %d sequences, %d programs, %d formulas",
			summary.NumSequences, summary.NumPrograms, summary.NumFormulas)
		s.summary = summary
	}
}

func (s *StatsServer) loadSubmitters() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	path := filepath.Join(s.dataDir, "stats", "submitters.csv")
	submitters, err := shared.LoadSubmittersCSV(path)
	if err != nil {
		log.Printf("Failed to load submitters: %v", err)
		s.submitters = nil
	} else {
		log.Printf("Loaded %d submitters", len(submitters))
		s.submitters = submitters
	}
}

func newCpuHourHandler(s *StatsServer) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		s.mutex.Lock()
		defer s.mutex.Unlock()

		// Try to parse JSON body
		var payload struct {
			Platform string `json:"platform"`
			Version  string `json:"version"`
			CPUHours int    `json:"cpuHours"`
		}
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&payload)
		if err == nil && (payload.Platform != "" || payload.Version != "" || payload.CPUHours > 0) {
			plat := payload.Platform
			ver := payload.Version
			hours := payload.CPUHours
			if hours <= 0 {
				hours = 1
			}
			if plat == "" {
				plat = "unknown"
			}
			if ver == "" {
				ver = "unknown"
			}
			if s.cpuHoursByPlatform[plat] == nil {
				s.cpuHoursByPlatform[plat] = make(map[string]int)
			}
			s.cpuHoursByPlatform[plat][ver] += hours
			util.WriteHttpCreated(w, "Metric received (json)")
			return
		}
		// Fallback: legacy increment
		s.cpuHours += 1
		util.WriteHttpCreated(w, "Metric received")
	}
	return http.HandlerFunc(f)
}

func (s *StatsServer) publishMetrics() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	labels := make(map[string]string)
	// Publish legacy/global cpuHours
	if s.cpuHours > 0 {
		s.influxDbClient.Write("cpuhours", labels, s.cpuHours)
		s.cpuHours = 0
	}
	// Publish per-platform/version
	for plat, vers := range s.cpuHoursByPlatform {
		for ver, hours := range vers {
			if hours > 0 {
				labels := map[string]string{"platform": plat, "version": ver}
				s.influxDbClient.Write("cpuhours", labels, hours)
				s.cpuHoursByPlatform[plat][ver] = 0
			}
		}
	}
}

func newOpenAPIHandler(s *StatsServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		accept := req.Header.Get("Accept")
		if accept == "application/yaml" || accept == "text/yaml" || accept == "application/x-yaml" {
			w.Header().Set("Content-Type", "application/yaml")
			w.Write(s.openApiSpec)
			return
		}
		// Serve Swagger UI
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
  <title>LODA API v2 - OpenAPI Spec</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.10.5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.10.5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      const ui = SwaggerUIBundle({
        url: '/v2/openapi.yaml',
        dom_id: '#swagger-ui',
      });
    };
  </script>
</body>
</html>`)
	})
}

func newOpenAPIYAMLHandler(s *StatsServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(s.openApiSpec)
	})
}

// Handler for /v2/stats/summary
func newSummaryHandler(s *StatsServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		if s.summary == nil {
			util.WriteHttpInternalServerError(w)
			return
		}
		util.WriteJsonResponse(w, s.summary)
	})
}

// KeywordInfo represents a keyword and its description
type KeywordInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Handler for /v2/stats/keywords
func newKeywordsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		var result []KeywordInfo
		for _, k := range shared.KeywordList {
			result = append(result, KeywordInfo{
				Name:        k,
				Description: shared.GetKeywordDescription(k),
			})
		}
		util.WriteJsonResponse(w, result)
	})
}

// Handler for /v2/stats/submitters
func newSubmittersHandler(s *StatsServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			util.WriteHttpMethodNotAllowed(w)
			return
		}
		if len(s.submitters) == 0 {
			log.Printf("No submitters found")
			util.WriteHttpInternalServerError(w)
			return
		}
		// Remove nils (from sparse array)
		var result []shared.Submitter
		for _, sub := range s.submitters {
			if sub != nil {
				result = append(result, *sub)
			}
		}
		util.WriteJsonResponse(w, result)
	})
}

func (s *StatsServer) Run(port int) {

	// initial data load
	s.loadSummary()
	s.loadSubmitters()

	// schedule background tasks
	reloadTicker := time.NewTicker(24 * time.Hour)
	defer reloadTicker.Stop()
	go func() {
		for range reloadTicker.C {
			s.loadSummary()
			s.loadSubmitters()
		}
	}()
	metricsTicker := time.NewTicker(10 * time.Minute)
	defer metricsTicker.Stop()
	go func() {
		for range metricsTicker.C {
			s.publishMetrics()
		}
	}()

	// start web server
	router := mux.NewRouter()
	router.Handle("/v1/cpuhours", newCpuHourHandler(s))
	router.Handle("/v2/openapi", newOpenAPIHandler(s))
	router.Handle("/v2/openapi.yaml", newOpenAPIYAMLHandler(s))
	router.Handle("/v2/stats/summary", newSummaryHandler(s))
	router.Handle("/v2/stats/keywords", newKeywordsHandler())
	router.Handle("/v2/stats/submitters", newSubmittersHandler(s))
	router.NotFoundHandler = http.HandlerFunc(util.HandleNotFound)
	log.Printf("Listening on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), router)
}

func main() {
	setup := cmd.GetSetup("stats")
	util.MustDirExist(setup.DataDir)
	openApiPath := filepath.Join(setup.DataDir, "openapi.v2.yaml")
	openApiSpec, err := os.ReadFile(openApiPath)
	if err != nil {
		log.Fatalf("Failed to read OpenAPI spec: %v", err)
	}
	u, p := util.ParseAuthInfo(setup.InfluxDbAuth)
	i := util.NewInfluxDbClient(setup.InfluxDbHost, u, p)
	s := NewStatsServer(i, openApiSpec, setup.DataDir)
	s.Run(8082)
}
