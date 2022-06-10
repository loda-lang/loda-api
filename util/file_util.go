package util

import (
	"log"
	"os"
	"time"
)

func MustDirExist(path string) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		log.Fatalf("Directory not found: %s", path)
	}
}

func IsFileRecent(path string, maxAge time.Duration) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	d := time.Since(info.ModTime())
	return d.Minutes() <= maxAge.Minutes()
}
