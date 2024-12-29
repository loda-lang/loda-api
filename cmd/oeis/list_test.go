package main

import (
	"os"
	"testing"

	"github.com/loda-lang/loda-api/util"
	"github.com/stretchr/testify/assert"
)

var (
	testFields = []Field{
		{Key: "S", SeqId: 1, Content: "test1"},
		{Key: "T", SeqId: 2, Content: "test2"},
		{Key: "T", SeqId: 2, Content: "test3"},
		{Key: "T", SeqId: 5, Content: "test5"},
		{Key: "U", SeqId: 7, Content: "test7"},
	}
)

func TestList_Update(t *testing.T) {
	l := NewList("T", "test", ".")
	l.Update(testFields)
	assert.Equal(t, 3, l.Len(), "Unexpected length")
}

func TestList_Flush(t *testing.T) {
	l := NewList("T", "test1", ".")
	l.Update(testFields)
	err := l.Flush()
	assert.Equal(t, nil, err, "Expected no error")
	assert.Equal(t, 0, l.Len(), "Unexpected length")
	assert.True(t, util.FileExists("test1.gz"), "Expected file to exist")
	os.Remove("test1.gz")
}

func testFindMissingIds(t *testing.T, l *List, maxId, maxNumIds, expectedNumMissing int, expected []int) {
	ids, numMissing, err := l.FindMissingIds(maxId, maxNumIds)
	assert.Equal(t, nil, err, "Expected no error")
	assert.Equal(t, expectedNumMissing, numMissing, "Unexpected number of missing ids")
	assert.Equal(t, expected, ids, "Unexpected ids")
}

func TestList_FindMissingIds(t *testing.T) {
	l := NewList("T", "test2", ".")
	l.Update(testFields)
	l.Flush()
	testFindMissingIds(t, l, 5, 2, 3, []int{1, 3})
	testFindMissingIds(t, l, 6, 2, 4, []int{1, 3})
	testFindMissingIds(t, l, 6, 3, 4, []int{1, 3, 4})
	testFindMissingIds(t, l, 6, 4, 4, []int{1, 3, 4, 6})
	testFindMissingIds(t, l, 6, 5, 4, []int{1, 3, 4, 6})
	testFindMissingIds(t, l, 7, 5, 5, []int{1, 3, 4, 6, 7})
	os.Remove("test2.gz")
}
