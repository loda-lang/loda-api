package main

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	fieldRegexp = regexp.MustCompile(`%([A-Za-z])\s+A([0-9]+)\s+(.+)`)
)

type Field struct {
	Key     string
	SeqId   int
	Content string
}

func ParseField(line string) (Field, error) {
	matches := fieldRegexp.FindStringSubmatch(line)
	if len(matches) != 4 {
		return Field{}, fmt.Errorf("Field parse error")
	}
	seqId, err := strconv.Atoi(matches[2])
	if err != nil {
		return Field{}, fmt.Errorf("Field seqId conversion error")
	}
	return Field{
		Key:     matches[1],
		SeqId:   seqId,
		Content: matches[3],
	}, nil
}
