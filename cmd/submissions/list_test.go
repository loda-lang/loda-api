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

func TestList_MultiLineFormat(t *testing.T) {
	// Create a list with duplicate entries
	l := NewList("T", "test_multiline", ".")
	fields := []Field{
		{Key: "T", SeqId: 1, Content: "first entry for A000001"},
		{Key: "T", SeqId: 1, Content: "second entry for A000001"},
		{Key: "T", SeqId: 1, Content: "third entry for A000001"},
		{Key: "T", SeqId: 2, Content: "single entry for A000002"},
		{Key: "T", SeqId: 3, Content: "first entry for A000003"},
		{Key: "T", SeqId: 3, Content: "second entry for A000003"},
	}
	l.Update(fields)
	err := l.Flush(false)
	assert.Equal(t, nil, err, "Expected no error")

	// Read the file and verify format
	content, err := os.ReadFile("test_multiline")
	assert.Equal(t, nil, err, "Expected no error reading file")

	lines := string(content)
	expected := `A000001: first entry for A000001
  second entry for A000001
  third entry for A000001
A000002: single entry for A000002
A000003: first entry for A000003
  second entry for A000003
`
	assert.Equal(t, expected, lines, "Unexpected multi-line format")

	os.Remove("test_multiline")
	os.Remove("test_multiline.gz")
}

func TestList_MultiLineFormatRoundTrip(t *testing.T) {
	// Test that we can read back a multi-line format file
	l := NewList("T", "test_roundtrip", ".")

	// Create initial file with multi-line format
	fields1 := []Field{
		{Key: "T", SeqId: 1, Content: "entry1-a"},
		{Key: "T", SeqId: 1, Content: "entry1-b"},
		{Key: "T", SeqId: 3, Content: "entry3"},
	}
	l.Update(fields1)
	err := l.Flush(false)
	assert.Equal(t, nil, err, "Expected no error")

	// Now add more entries and flush again
	fields2 := []Field{
		{Key: "T", SeqId: 1, Content: "entry1-c"},
		{Key: "T", SeqId: 2, Content: "entry2"},
		{Key: "T", SeqId: 3, Content: "entry3-b"},
	}
	l.Update(fields2)
	err = l.Flush(false)
	assert.Equal(t, nil, err, "Expected no error")

	// Read the file and verify all entries are present
	content, err := os.ReadFile("test_roundtrip")
	assert.Equal(t, nil, err, "Expected no error reading file")

	lines := string(content)
	expected := `A000001: entry1-a
  entry1-b
  entry1-c
A000002: entry2
A000003: entry3
  entry3-b
`
	assert.Equal(t, expected, lines, "Unexpected multi-line format after round trip")

	os.Remove("test_roundtrip")
	os.Remove("test_roundtrip.gz")
}

func TestList_Flush(t *testing.T) {
	l := NewList("T", "test1", ".")
	l.Update(testFields)
	err := l.Flush(false)
	assert.Equal(t, nil, err, "Expected no error")
	assert.Equal(t, 0, l.Len(), "Unexpected length")
	assert.True(t, util.FileExists("test1"), "Expected file to exist")
	assert.True(t, util.FileExists("test1.gz"), "Expected file to exist")
	os.Remove("test1")
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
	l.Flush(false)
	testFindMissingIds(t, l, 5, 2, 3, []int{1, 3})
	testFindMissingIds(t, l, 6, 2, 4, []int{1, 3})
	testFindMissingIds(t, l, 6, 3, 4, []int{1, 3, 4})
	testFindMissingIds(t, l, 6, 4, 4, []int{1, 3, 4, 6})
	testFindMissingIds(t, l, 6, 5, 4, []int{1, 3, 4, 6})
	testFindMissingIds(t, l, 7, 5, 5, []int{1, 3, 4, 6, 7})
	os.Remove("test2")
	os.Remove("test2.gz")
}
