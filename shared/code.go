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

// extractOperationTypes extracts the unique operation types from operations.
// For example, "mov $1,$0" -> "mov", "add $1,1" -> "add"
func extractOperationTypes(operations []string) []string {
	seen := make(map[string]bool)
	var opTypes []string
	for _, op := range operations {
		// Get the first word (the operation type)
		parts := strings.Fields(op)
		if len(parts) > 0 {
			opType := parts[0]
			if !seen[opType] {
				seen[opType] = true
				opTypes = append(opTypes, opType)
			}
		}
	}
	return opTypes
}

func extractHeaderComments(code string) []string {
	lines := strings.Split(code, "\n")
	var header []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, ";") {
			comment := strings.TrimSpace(line[1:])
			header = append(header, comment)
		} else {
			break
		}
	}
	return header
}

func extractIdAndName(code string) (util.UID, string) {
	var id util.UID
	var name string
	header := extractHeaderComments(code)
	for _, comment := range header {
		parts := strings.SplitN(comment, ":", 2)
		if len(parts) == 2 {
			idStr := strings.TrimSpace(parts[0])
			uid, err := util.NewUIDFromString(idStr)
			if err == nil {
				id = uid
				name = strings.TrimSpace(parts[1])
				break
			}
		}
	}
	return id, name
}

var submitterPrefix = "Submitted by "

func extractSubmitter(code string) *Submitter {
	header := extractHeaderComments(code)
	for _, comment := range header {
		if after, ok := strings.CutPrefix(comment, submitterPrefix); ok {
			name := strings.TrimSpace(after)
			return &Submitter{Name: name}
		}
	}
	return nil
}

func extractFormula(code string) string {
	header := extractHeaderComments(code)
	for _, comment := range header {
		if after, ok := strings.CutPrefix(comment, "Formula:"); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

func extractMinerProfile(code string) string {
	// Miner profiles are not always in the header
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "; Miner Profile:"); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

func updateIdAndName(code string, id util.UID, name string) string {
	lines := strings.Split(code, "\n")
	isHeader := true
	updated := false
	resultLines := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			resultLines = append(resultLines, line)
			continue
		}
		if !strings.HasPrefix(line, ";") {
			if !updated {
				resultLines = append(resultLines, "; "+id.String()+": "+name)
				updated = true
			}
			isHeader = false
		}
		if isHeader && !updated {
			comment := strings.TrimSpace(line[1:])
			parts := strings.SplitN(comment, ":", 2)
			if len(parts) == 2 {
				idStr := strings.TrimSpace(parts[0])
				_, err := util.NewUIDFromString(idStr)
				if err == nil {
					line = "; " + id.String() + ": " + name
					updated = true
				}
			}
		}
		resultLines = append(resultLines, line)
	}
	return strings.Join(resultLines, "\n")
}

func updateSubmitter(code string, submitter *Submitter) string {
	resultLines := []string{}
	lines := strings.Split(code, "\n")
	isHeader := true
	updated := false
	for _, line := range lines {
		line := strings.TrimSpace(line)
		if len(line) == 0 {
			resultLines = append(resultLines, line)
			continue
		}
		if !strings.HasPrefix(line, ";") {
			if !updated && submitter != nil {
				resultLines = append(resultLines, "; "+submitterPrefix+submitter.Name)
				updated = true
			}
			isHeader = false
		}
		if isHeader && !updated {
			comment := strings.TrimSpace(line[1:])
			if strings.HasPrefix(comment, submitterPrefix) {
				if submitter != nil {
					line = "; " + submitterPrefix + submitter.Name
				} else {
					continue // remove the line
				}
				updated = true
			}
		}
		resultLines = append(resultLines, line)
	}
	return strings.Join(resultLines, "\n")
}
