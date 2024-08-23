package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"sync"

	"github.com/loda-lang/loda-api/util"
)

var (
	lineRegexp = regexp.MustCompile(`A([0-9]+): (.+)`)
)

type List struct {
	key     string
	name    string
	dataDir string
	fields  []Field
	mutex   sync.Mutex
}

func NewList(key, name, dataDir string) List {
	return List{
		key:     key,
		name:    name,
		dataDir: dataDir,
	}
}

func (l *List) Len() int {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return len(l.fields)
}

func (l *List) Update(fields []Field) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for _, field := range fields {
		if field.Key == l.key {
			l.fields = append(l.fields, field)
		}
	}
}

func (l *List) Flush() error {
	log.Printf("Flushing list %s", l.name)
	l.mutex.Lock()
	defer l.mutex.Unlock()
	// Check and sort fields
	if len(l.fields) == 0 {
		return nil
	}
	sort.Slice(l.fields, func(i, j int) bool {
		f := l.fields[i]
		g := l.fields[j]
		return (f.SeqId < g.SeqId) || (f.SeqId == g.SeqId && f.Content < g.Content)
	})
	// Uncompress old file
	path := filepath.Join(l.dataDir, l.name)
	gzPath := path + ".gz"
	if util.FileExists(gzPath) {
		err := exec.Command("gzip", "-d", gzPath).Run()
		if err != nil {
			return fmt.Errorf("failed to gunzip file: %w", err)
		}
	} else {
		file, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		file.Close()
	}
	oldPath := path + "_old"
	os.Rename(path, oldPath)
	// Merge fields with old content
	old, err := os.Open(oldPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	target, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	err = mergeLists(l.fields, old, target)
	target.Close()
	old.Close()
	os.Remove(oldPath)
	if err != nil {
		return fmt.Errorf("failed to merge lists: %w", err)
	}
	// Compress new file
	err = exec.Command("gzip", "-f", path).Run()
	if err != nil {
		return fmt.Errorf("failed to gzip file: %w", err)
	}
	l.fields = nil
	return nil
}

func formatField(field Field) string {
	return fmt.Sprintf("A%06d: %s", field.SeqId, field.Content)
}

func mergeLists(fields []Field, old, target *os.File) error {
	// Merges fields with old list and writes to target list
	i := 0
	scanner := bufio.NewScanner(old)
	for scanner.Scan() {
		// Read and parse old line
		line := scanner.Text()
		matches := lineRegexp.FindStringSubmatch(line)
		if len(matches) != 3 {
			return fmt.Errorf("failed parsing line: %s", line)
		}
		seqId, err := strconv.Atoi(matches[1])
		if err != nil {
			return fmt.Errorf("failed parsing seqId: %w", err)
		}
		content := matches[2]
		// Write all new fields with smaller seqId
		for i < len(fields) && (fields[i].SeqId < seqId || (fields[i].SeqId == seqId && fields[i].Content < content)) {
			_, err := target.WriteString(formatField(fields[i]) + "\n")
			if err != nil {
				return fmt.Errorf("failed writing field: %w", err)
			}
			i++
		}
		// Write old line if it is not the same as the new field
		if i >= len(fields) || fields[i].SeqId != seqId || fields[i].Content != content {
			_, err = target.WriteString(line + "\n")
			if err != nil {
				return fmt.Errorf("failed writing line: %w", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed reading old list: %w", err)
	}
	// Write remaining new fields
	for i < len(fields) {
		_, err := target.WriteString(formatField(fields[i]) + "\n")
		if err != nil {
			return fmt.Errorf("failed writing field: %w", err)
		}
		i++
	}
	return nil
}

func (l *List) ServeGzip(w http.ResponseWriter, r *http.Request) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	util.ServeBinary(w, r, filepath.Join(l.dataDir, l.name+".gz"))
}
