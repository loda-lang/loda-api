package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func checkField(t *testing.T, f Field, key string, seqId int, content string) {
	assert.Equal(t, key, f.Key, "Unexpected key")
	assert.Equal(t, seqId, f.SeqId, "Unexpected seqId")
	assert.Equal(t, content, f.Content, "Unexpected content")
}

func checkFieldString(t *testing.T, line string, key string, seqId int, content string) {
	f, err := ParseField(line)
	assert.Equal(t, nil, err, "Expected no error")
	checkField(t, f, key, seqId, content)
}

func TestParseField(t *testing.T) {
	checkFieldString(t,
		"%N A000042 Unary representation of natural numbers.",
		"N", 42, "Unary representation of natural numbers.")
}
