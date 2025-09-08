package shared

import (
	"strings"

	"github.com/loda-lang/loda-api/util"
)

func extractOperations(code string) []string {
	var operations []string
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		operations = append(operations, line)
	}
	return operations
}

func extractIdAndName(code string) (util.UID, string) {
	var id util.UID
	var name string
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, ";") {
			parts := strings.SplitN(line[1:], ":", 2)
			if len(parts) == 2 {
				idStr := strings.TrimSpace(parts[0])
				uid, err := util.NewUIDFromString(idStr)
				if err == nil {
					id = uid
				}
				name = strings.TrimSpace(parts[1])
			}
			break
		}
	}
	return id, name
}

func extractSubmitter(code string) *Submitter {
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "; Submitted by "); ok {
			name := strings.TrimSpace(after)
			return &Submitter{Name: name}
		}
	}
	return nil
}
