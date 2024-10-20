package main

import (
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func checkFieldBasics(t *testing.T, fields []Field) {
	assert.True(t, len(fields) > 0, "Expected some fields")
}

func findField(t *testing.T, fields []Field, key string) Field {
	idx := slices.IndexFunc(fields, func(f Field) bool { return f.Key == key })
	assert.NotEqual(t, -1, idx, "Expected a field with key %s", key)
	return fields[idx]
}

func checkFieldDetails(t *testing.T, fields []Field, key string, seqId int, content string) {
	f := findField(t, fields, key)
	checkField(t, f, key, seqId, content)
}

func TestCrawler_Init(t *testing.T) {
	c := NewCrawler(http.DefaultClient)
	err := c.Init()
	assert.Equal(t, nil, err, "Expected no error")
	assert.True(t, c.maxId > 0, "Unexpected max Id")
	assert.True(t, c.currentId > 0 && c.currentId <= c.maxId, "Unexpected current Id")
	assert.True(t, c.stepSize > 0, "Unexpected step size")
}

func TestCrawler_FetchSeq(t *testing.T) {
	c := NewCrawler(http.DefaultClient)
	fields, err, status := c.FetchSeq(30, false)
	assert.Equal(t, nil, err, "Expected no error")
	assert.Equal(t, http.StatusOK, status, "Expected OK status")
	checkFieldDetails(t, fields, "N", 30, "Initial digit of n.")
	checkFieldDetails(t, fields, "K", 30, "nonn,base,easy,nice,look")
	checkFieldDetails(t, fields, "O", 30, "0,3")
}

func TestCrawler_FetchNext(t *testing.T) {
	c := NewCrawler(http.DefaultClient)
	for i := 0; i < 10; i++ {
		fields, err, status := c.FetchNext()
		assert.Equal(t, http.StatusOK, status, "Expected OK status")
		assert.Equal(t, nil, err, "Expected no error")
		checkFieldBasics(t, fields)
		findField(t, fields, "N")
	}
}
