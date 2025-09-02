package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/loda-lang/loda-api/cmd"
	"github.com/loda-lang/loda-api/util"
)

type StatsServer struct {
	openApiSpec    []byte
	influxDbClient *util.InfluxDbClient
	cpuHours       int
	mutex          sync.Mutex
}

func NewStatsServer(influxDbClient *util.InfluxDbClient, openApiSpec []byte) *StatsServer {
	return &StatsServer{
		openApiSpec:    openApiSpec,
		influxDbClient: influxDbClient,
		cpuHours:       0,
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
		s.cpuHours += 1
		util.WriteHttpCreated(w, "Metric received")
	}
	return http.HandlerFunc(f)
}

func (s *StatsServer) publishMetrics() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	labels := make(map[string]string)
	s.influxDbClient.Write("cpuhours", labels, s.cpuHours)
	s.cpuHours = 0
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

func (s *StatsServer) Run(port int) {
	// regularly publish metrics
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			s.publishMetrics()
		}
	}()
	// start web server
	router := mux.NewRouter()
	router.Handle("/v1/cpuhours", newCpuHourHandler(s))
	router.Handle("/v2/openapi", newOpenAPIHandler(s))
	router.Handle("/v2/openapi.yaml", newOpenAPIYAMLHandler(s))
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
	s := NewStatsServer(i, openApiSpec)
	s.Run(8082)
}
