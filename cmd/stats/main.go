package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/loda-lang/loda-api/cmd"
	"github.com/loda-lang/loda-api/util"
)

type StatsServer struct {
	influxDbClient *util.InfluxDbClient
	cpuHours       int
	mutex          sync.Mutex
}

func NewStatsServer(influxDbClient *util.InfluxDbClient) *StatsServer {
	return &StatsServer{
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
	router.NotFoundHandler = http.HandlerFunc(util.HandleNotFound)
	log.Printf("Listening on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), router)
}

func main() {
	setup := cmd.GetSetup("stats")
	u, p := util.ParseAuthInfo(setup.InfluxDbAuth)
	i := util.NewInfluxDbClient(setup.InfluxDbHost, u, p)
	s := NewStatsServer(i)
	s.Run(8082)
}
