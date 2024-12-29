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

func NewList(key, name, dataDir string) *List {
	return &List{
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

func (l *List) FindMissingIds(maxId int, maxNumIds int) ([]int, int, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	log.Printf("Finding missing %s", l.name)
	path := filepath.Join(l.dataDir, l.name)
	gzPath := path + ".gz"
	if !util.FileExists(gzPath) {
		log.Printf("No %s available", l.name)
		return nil, 0, nil // not an error
	}
	err := exec.Command("gzip", "-d", gzPath).Run()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to gunzip file: %w", err)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file: %w", err)
	}
	ids, numMissing, err := findMissingIds(file, maxId, maxNumIds)
	file.Close()
	if err != nil {
		return nil, 0, err
	}
	err = exec.Command("gzip", "-f", path).Run()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to gzip file: %w", err)
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

func mergeLists(fields []Field, old, target *os.File) error {
	// Merges fields with old list and writes to target list
	i := 0
	scanner := bufio.NewScanner(old)
	for scanner.Scan() {
		// Read and parse old line
		line := scanner.Text()
		f, err := parseLine(line)
		if err != nil {
			return err
		}
		// Write all new fields with smaller seqId
		for i < len(fields) && (fields[i].SeqId < f.SeqId || (fields[i].SeqId == f.SeqId && fields[i].Content < f.Content)) {
			_, err := target.WriteString(formatField(fields[i]) + "\n")
			if err != nil {
				return fmt.Errorf("failed writing field: %w", err)
			}
			i++
		}
		// Write old line if it is not the same as the new field
		if i >= len(fields) || fields[i].SeqId != f.SeqId || fields[i].Content != f.Content {
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

func findMissingIds(file *os.File, maxId int, maxNumIds int) ([]int, int, error) {
	ids := []int{}
	nextId := 1
	numMissing := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		f, err := parseLine(line)
		if err != nil {
			return nil, 0, err
		}
		for i := nextId; i < f.SeqId && len(ids) < maxNumIds; i++ {
			ids = append(ids, i)
		}
		numMissing += f.SeqId - nextId + 1
		nextId = f.SeqId + 1
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed reading list: %w", err)
	}
	for i := nextId; i <= maxId && len(ids) < maxNumIds; i++ {
		ids = append(ids, i)
	}
	numMissing += maxId - nextId - 1
	return ids, numMissing, nil
}

func (l *List) ServeGzip(w http.ResponseWriter, r *http.Request) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	util.ServeBinary(w, r, filepath.Join(l.dataDir, l.name+".gz"))
}
