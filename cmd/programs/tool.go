package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/loda-lang/loda-api/util"
)

type LODATool struct {
	dataDir string
}

func NewLODATool(dataDir string) *LODATool {
	return &LODATool{dataDir: dataDir}
}

func (t *LODATool) Install() error {
	// Ensure that the setup.txt file exists
	setupFile := filepath.Join(t.dataDir, "setup.txt")
	if !util.FileExists(setupFile) {
		return fmt.Errorf("setup.txt file not found in data directory: %s", t.dataDir)
	}
	// Install the "loda" executable in $HOME/bin
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	binDir := filepath.Join(homeDir, "bin")
	if !util.FileExists(binDir) {
		if err := os.MkdirAll(binDir, 0755); err != nil {
			return fmt.Errorf("failed to create $HOME/bin directory: %w", err)
		}
	}
	lodaExec := filepath.Join(binDir, "loda")
	if !util.FileExists(lodaExec) {
		executable := "loda-linux-x86"
		log.Printf("Downloading %s to: %s", executable, lodaExec)
		cmd := exec.Command("curl", "-fsSLo", "loda", "https://github.com/loda-lang/loda-cpp/releases/latest/download/"+executable)
		cmd.Dir = binDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to download loda executable: %w", err)
		}
		if err := os.Chmod(lodaExec, 0755); err != nil {
			return fmt.Errorf("failed to set executable permission on loda: %w", err)
		}
	} else {
		log.Printf("Checking for new LODA version")
		_, err := t.Exec("upgrade")
		if err != nil {
			return fmt.Errorf("failed to upgrade loda executable: %w", err)
		}
	}
	// Ensure the "programs" directory exists by cloning the repository if necessary
	progsDir := filepath.Join(t.dataDir, "programs")
	if !util.FileExists(progsDir) {
		log.Printf("Cloning loda-programs repository to: %s", progsDir)
		cmd := exec.Command("git", "clone", "https://github.com/loda-lang/loda-programs.git", progsDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone loda-programs: %w", err)
		}
	}
	return nil
}

func (t *LODATool) Exec(args ...string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	lodaExec := filepath.Join(homeDir, "bin", "loda")
	if !util.FileExists(lodaExec) {
		return "", fmt.Errorf("loda executable not found at: %s", lodaExec)
	}
	cmd := exec.Command(lodaExec, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "LODA_HOME="+t.dataDir)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Stream output in real-time
	var outputBuilder strings.Builder
	var wg sync.WaitGroup
	stream := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line + "\n")
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "|", 2)
			if len(parts) == 2 {
				log.Print(parts[1])
			} else {
				log.Print(line)
			}
		}
	}
	wg.Add(2)
	go func() { defer wg.Done(); stream(stdout) }()
	go func() { defer wg.Done(); stream(stderr) }()
	wg.Wait()

	err = cmd.Wait()
	return outputBuilder.String(), err
}
