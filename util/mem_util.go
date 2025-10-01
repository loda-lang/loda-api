package util

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// GetFreeMemoryBytes returns available system memory in KB (Linux only). Returns 0 on error or non-Linux systems.
func GetFreeMemoryKB() int {
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
					return int(kb)
				}
			}
			break
		}
	}
	return 0
}

// GetProcessesMemoryUsageKB returns a map of process name to total memory usage (in KB) for all running processes starting with that name.
// Only works on Linux.
func GetProcessesMemoryUsageKB(processNames []string) (map[string]int, error) {
	result := make(map[string]int)
	procEntries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	for _, entry := range procEntries {
		if !entry.IsDir() {
			continue
		}
		pid := entry.Name()
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		cmdlinePath := filepath.Join("/proc", pid, "comm")
		cmdlineBytes, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue
		}
		procName := strings.TrimSpace(string(cmdlineBytes))
		matched := ""
		for _, prefix := range processNames {
			if strings.HasPrefix(procName, prefix) {
				matched = prefix
				break
			}
		}
		if matched == "" {
			continue
		}
		statusPath := filepath.Join("/proc", pid, "status")
		statusBytes, err := os.ReadFile(statusPath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(statusBytes), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					memKB, err := strconv.Atoi(fields[1])
					if err == nil {
						result[matched] += memKB
					}
				}
				break
			}
		}
	}
	return result, nil
}
