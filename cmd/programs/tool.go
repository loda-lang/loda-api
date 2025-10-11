package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/loda-lang/loda-api/shared"
	"github.com/loda-lang/loda-api/util"
)

// SupportedExportFormats defines the export formats supported by the LODA tool
var SupportedExportFormats = []string{"formula", "pari", "loda", "range"}

type EvalResult struct {
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Terms   []string `json:"terms"`
}

type ExportResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Output  string `json:"output"`
}

type LODATool struct {
	dataDir string
	evalSem chan struct{}
}

func NewLODATool(dataDir string, maxNumParallelEval int) *LODATool {
	evalSem := make(chan struct{}, maxNumParallelEval)
	return &LODATool{
		dataDir: dataDir,
		evalSem: evalSem,
	}
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
	dataBinDir := filepath.Join(t.dataDir, "bin")
	if !util.FileExists(dataBinDir) {
		if err := os.Symlink(binDir, dataBinDir); err != nil && !os.IsExist(err) {
			return fmt.Errorf("failed to create symlink from %s to %s: %w", binDir, dataBinDir, err)
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
		_, err := t.Exec(0, "upgrade")
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

// Exec runs the loda command. If timeout > 0, enforces a timeout. Accepts args as variadic.
func (t *LODATool) Exec(timeout time.Duration, args ...string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	lodaExec := filepath.Join(homeDir, "bin", "loda")
	if !util.FileExists(lodaExec) {
		return "", fmt.Errorf("loda executable not found at: %s", lodaExec)
	}

	var cmd *exec.Cmd
	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		execArgs := args
		execArgs = append(execArgs, "-z", fmt.Sprintf("%d", int(timeout.Seconds())))
		cmd = exec.CommandContext(ctx, lodaExec, execArgs...)
	} else {
		cmd = exec.Command(lodaExec, args...)
	}
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

// writeProgramToTempFile creates a temporary file and writes the program code to it.
// Returns the file path and a cleanup function. The cleanup function should be called
// to remove the temporary file when done.
func (t *LODATool) writeProgramToTempFile(program shared.Program, prefix string) (string, func(), error) {
	tmpfile, err := os.CreateTemp("", prefix+"*.asm")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	cleanup := func() {
		os.Remove(tmpfile.Name())
	}
	if _, err := tmpfile.Write([]byte(program.Code)); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to close temp file: %w", err)
	}
	return tmpfile.Name(), cleanup, nil
}

// Eval evaluates a LODA program and returns a Result with status, message, and terms.
func (t *LODATool) Eval(program shared.Program, numTerms int) EvalResult {
	t.evalSem <- struct{}{}
	defer func() { <-t.evalSem }()
	tmpfilePath, cleanup, err := t.writeProgramToTempFile(program, "loda_eval_")
	if err != nil {
		return EvalResult{
			Status:  "error",
			Message: err.Error(),
			Terms:   nil,
		}
	}
	defer cleanup()
	args := []string{"eval", tmpfilePath, "-t", strconv.Itoa(numTerms)}
	output, execErr := t.Exec(10*time.Second, args...)
	var terms []string
	status := "success"
	message := ""
	if execErr != nil {
		// If error, check if output has two lines: terms and error message
		lines := strings.SplitN(output, "\n", 3)
		if len(lines) >= 2 {
			terms = strings.Split(lines[0], ",")
			for i := range terms {
				terms[i] = strings.TrimSpace(terms[i])
			}
			message = strings.TrimSpace(lines[1])
		} else {
			message = execErr.Error()
		}
		status = "error"
	} else {
		// Success: output is terms (single line)
		terms = strings.Split(strings.TrimSpace(output), ",")
		for i := range terms {
			terms[i] = strings.TrimSpace(terms[i])
		}
	}
	return EvalResult{
		Status:  status,
		Message: message,
		Terms:   terms,
	}
}

// Export exports a LODA program to various formats using the loda export command.
// Supported formats are defined in SupportedExportFormats variable.
func (t *LODATool) Export(program shared.Program, format string) ExportResult {
	t.evalSem <- struct{}{}
	defer func() { <-t.evalSem }()
	// Validate format
	isValid := false
	for _, f := range SupportedExportFormats {
		if f == format {
			isValid = true
			break
		}
	}
	if !isValid {
		return ExportResult{
			Status:  "error",
			Message: fmt.Sprintf("invalid format: %s (supported: %s)", format, strings.Join(SupportedExportFormats, ", ")),
			Output:  "",
		}
	}
	tmpfilePath, cleanup, err := t.writeProgramToTempFile(program, "loda_export_")
	if err != nil {
		return ExportResult{
			Status:  "error",
			Message: err.Error(),
			Output:  "",
		}
	}
	defer cleanup()
	args := []string{"export", "-o", format, tmpfilePath}
	output, execErr := t.Exec(10*time.Second, args...)
	status := "success"
	message := ""
	if execErr != nil {
		status = "error"
		message = execErr.Error()
		if output != "" {
			message = strings.TrimSpace(output)
		}
	}
	return ExportResult{
		Status:  status,
		Message: message,
		Output:  strings.TrimSpace(output),
	}
}
