package cmd

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/loda-lang/loda-api/util"
)

type LodaSetup struct {
	DataDir        string
	UpdateInterval time.Duration
	InfluxDbHost   string
	InfluxDbAuth   string
}

func GetSetup(app string) LodaSetup {
	if len(os.Args) != 2 {
		log.Fatal("Invalid command-line arguments. Please pass the data directory as argument.")
	}
	dataDir := os.Args[1]
	setup := LodaSetup{
		DataDir:        dataDir,
		UpdateInterval: 24 * time.Hour, // default value
	}
	setupPath := filepath.Join(dataDir, "setup.txt")
	file, err := os.Open(setupPath)
	if err != nil {
		log.Fatalf("Failed to open: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		entry := strings.Split(scanner.Text(), "=")
		if len(entry) != 2 {
			continue
		}
		key := strings.TrimSpace(entry[0])
		value := strings.TrimSpace(entry[1])
		switch {
		case key == "LODA_LOG_DIR":
			util.InitLog(filepath.Join(value, app))
		case key == "LODA_UPDATE_INTERVAL":
			d, err := time.ParseDuration(value)
			if err != nil {
				log.Printf("Invalid duration: %s", value)
			} else {
				setup.UpdateInterval = d
			}
		case key == "LODA_INFLUXDB_HOST":
			setup.InfluxDbHost = value
		case key == "LODA_INFLUXDB_AUTH":
			setup.InfluxDbAuth = value
		}
	}
	return setup
}
