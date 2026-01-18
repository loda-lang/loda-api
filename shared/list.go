package shared

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
	lineRegexp       = regexp.MustCompile(`A([0-9]+): (.+)`)
	continuationLine = regexp.MustCompile(`^  (.+)`)
)

type List struct {
	key     string
	name    string
	dataDir string
	fields  []Field
	mutex   sync.Mutex
}

func NewList(key, name, dataDir string) *List {
	return &List{
		key:     key,
		name:    name,
		dataDir: dataDir,
	}
}

// Name returns the name of the list
func (l *List) Name() string {
	return l.name
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

func (l *List) Flush(deduplicate bool) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	log.Printf("Flushing %s", l.name)
	// Check and sort fields
	if len(l.fields) == 0 {
		return nil
	}
	sort.Slice(l.fields, func(i, j int) bool {
		f := l.fields[i]
		g := l.fields[j]
		return (f.SeqId < g.SeqId) || (f.SeqId == g.SeqId && f.Content < g.Content)
	})
	path := filepath.Join(l.dataDir, l.name)
	// Create file if not exists yet
	if !util.FileExists(path) {
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
	err = mergeLists(l.fields, old, target, deduplicate)
	target.Close()
	old.Close()
	os.Remove(oldPath)
	if err != nil {
		return fmt.Errorf("failed to merge lists: %w", err)
	}
	// Compress new file
	err = exec.Command("gzip", "-f", "-k", path).Run()
	if err != nil {
		return fmt.Errorf("failed to gzip file: %w", err)
	}
	l.fields = nil
	return nil
}

func (l *List) FindMissingIds(maxId int, maxNumIds int) ([]int, int, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	log.Printf("Finding missing %s", l.name)
	path := filepath.Join(l.dataDir, l.name)
	if !util.FileExists(path) {
		log.Printf("No %s available", l.name)
		return nil, 0, nil // not an error
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	ids, numMissing, err := findMissingIds(file, maxId, maxNumIds)
	if err != nil {
		return nil, 0, err
	}
	log.Printf("Found %d/%d missing %s", len(ids), numMissing, l.name)
	return ids, numMissing, nil
}

func formatField(field Field) string {
	return fmt.Sprintf("A%06d: %s", field.SeqId, field.Content)
}

func parseLine(line string) (Field, error) {
	matches := lineRegexp.FindStringSubmatch(line)
	if len(matches) != 3 {
		return Field{}, fmt.Errorf("failed parsing line: %s", line)
	}
	seqId, err := strconv.Atoi(matches[1])
	if err != nil {
		return Field{}, fmt.Errorf("failed parsing seqId: %w", err)
	}
	return Field{
		Key:     "",
		SeqId:   seqId,
		Content: matches[2],
	}, nil
}

func isContinuationLine(line string) bool {
	return continuationLine.MatchString(line)
}

func parseContinuationLine(line string) (string, error) {
	matches := continuationLine.FindStringSubmatch(line)
	if len(matches) != 2 {
		return "", fmt.Errorf("failed parsing continuation line: %s", line)
	}
	return matches[1], nil
}

func mergeLists(fields []Field, old, target *os.File, deduplicate bool) error {
	// Merges fields with old list and writes to target list
	// If deduplicate is true, remove duplicate entries (same SeqId)
	// Outputs in multi-line format: first line has "A000000: content", continuation lines have "  content"

	// Read all old entries grouped by SeqId
	oldEntries := make(map[int][]string)
	scanner := bufio.NewScanner(old)
	var currentSeqId int = -1

	for scanner.Scan() {
		line := scanner.Text()
		if isContinuationLine(line) {
			// This is a continuation line
			if currentSeqId >= 0 {
				content, err := parseContinuationLine(line)
				if err != nil {
					return err
				}
				oldEntries[currentSeqId] = append(oldEntries[currentSeqId], content)
			}
		} else {
			// This is a new entry
			f, err := parseLine(line)
			if err != nil {
				return err
			}
			currentSeqId = f.SeqId
			oldEntries[currentSeqId] = append(oldEntries[currentSeqId], f.Content)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed reading old list: %w", err)
	}

	// Group new fields by SeqId
	newEntries := make(map[int][]string)
	for _, field := range fields {
		newEntries[field.SeqId] = append(newEntries[field.SeqId], field.Content)
	}

	// Merge old and new entries
	allSeqIds := make(map[int]bool)
	for seqId := range oldEntries {
		allSeqIds[seqId] = true
	}
	for seqId := range newEntries {
		allSeqIds[seqId] = true
	}

	// Convert to sorted slice
	var seqIds []int
	for seqId := range allSeqIds {
		seqIds = append(seqIds, seqId)
	}
	sort.Ints(seqIds)

	// Write merged entries in multi-line format
	for _, seqId := range seqIds {
		var entries []string

		// Merge old and new entries for this seqId
		seen := make(map[string]bool)

		// Add new entries first (so they take precedence when deduplicating)
		for _, content := range newEntries[seqId] {
			if !seen[content] {
				entries = append(entries, content)
				seen[content] = true
			}
		}

		// Add old entries
		for _, content := range oldEntries[seqId] {
			if !seen[content] {
				entries = append(entries, content)
				seen[content] = true
			}
		}

		// If deduplicate, keep only one entry
		if deduplicate && len(entries) > 0 {
			entries = entries[:1]
		}

		// Write entries in multi-line format
		if len(entries) > 0 {
			// First entry with full prefix
			_, err := target.WriteString(fmt.Sprintf("A%06d: %s\n", seqId, entries[0]))
			if err != nil {
				return fmt.Errorf("failed writing field: %w", err)
			}

			// Continuation lines with 2-space indentation
			for _, content := range entries[1:] {
				_, err := target.WriteString(fmt.Sprintf("  %s\n", content))
				if err != nil {
					return fmt.Errorf("failed writing continuation: %w", err)
				}
			}
		}
	}

	return nil
}

func findMissingIds(file *os.File, maxId int, maxNumIds int) ([]int, int, error) {
	ids := []int{}
	nextId := 1
	numMissing := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip continuation lines
		if isContinuationLine(line) {
			continue
		}
		f, err := parseLine(line)
		if err != nil {
			return nil, 0, err
		}
		for i := nextId; i < f.SeqId && len(ids) < maxNumIds; i++ {
			ids = append(ids, i)
		}
		if f.SeqId > nextId {
			numMissing += f.SeqId - nextId
		}
		nextId = f.SeqId + 1
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed reading list: %w", err)
	}
	for i := nextId; i <= maxId && len(ids) < maxNumIds; i++ {
		ids = append(ids, i)
	}
	if maxId >= nextId {
		numMissing += maxId + 1 - nextId
	}
	return ids, numMissing, nil
}

func (l *List) ServeGzip(w http.ResponseWriter, r *http.Request) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	util.ServeBinary(w, r, filepath.Join(l.dataDir, l.name+".gz"))
}
