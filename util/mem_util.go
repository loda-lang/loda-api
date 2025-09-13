package util

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// GetFreeMemoryBytes returns available system memory in bytes (Linux only). Returns 0 on error or non-Linux systems.
func GetFreeMemoryBytes() uint64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if kb, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					return kb * 1024
				}
			}
			break
		}
	}
	return 0
}
