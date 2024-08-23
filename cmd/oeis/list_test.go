package main

import (
	"testing"

	"github.com/loda-lang/loda-api/util"
	"github.com/stretchr/testify/assert"
)

var (
	testFields = []Field{
		{Key: "S", SeqId: 1, Content: "test1"},
		{Key: "T", SeqId: 2, Content: "test2"},
		{Key: "T", SeqId: 2, Content: "test3"},
		{Key: "T", SeqId: 4, Content: "test4"},
		{Key: "U", SeqId: 5, Content: "test5"},
	}
)

func TestList_Update(t *testing.T) {
	l := NewList("T", "test", ".")
	l.Update(testFields)
	assert.Equal(t, 3, l.Len(), "Unexpected length")
}

func TestList_Flush(t *testing.T) {
	l := NewList("T", "test", ".")
	l.Update(testFields)
	err := l.Flush()
	assert.Equal(t, nil, err, "Expected no error")
	assert.Equal(t, 0, l.Len(), "Unexpected length")
	assert.True(t, util.FileExists("test.gz"), "Expected file to exist")
}
