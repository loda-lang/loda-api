package main

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/loda-lang/loda-api/util"
)

func TestTermsList(t *testing.T) {
	id, _ := util.NewUIDFromString("A000001")
	seq := Sequence{
		Id:    id,
		Name:  "Number of groups of order n.",
		Terms: ",0,1,1,1,2,1,2,1,5,2,2,1,5,1,2,1,14,1,5,1,5,2,2,1,15,2,2,5,4,1,4,1,51,1,2,1,14,1,2,2,14,1,6,1,4,2,2,1,52,2,5,1,5,1,15,2,13,2,2,1,13,1,2,4,267,1,4,1,5,1,4,1,50,1,2,3,4,1,6,1,52,15,2,1,15,1,2,1,12,1,10,1,4,2,",
	}
	got := seq.TermsList()
	want := []string{"0", "1", "1", "1", "2", "1", "2", "1", "5", "2", "2", "1", "5", "1", "2", "1", "14", "1", "5", "1", "5", "2", "2", "1", "15", "2", "2", "5", "4", "1", "4", "1", "51", "1", "2", "1", "14", "1", "2", "2", "14", "1", "6", "1", "4", "2", "2", "1", "52", "2", "5", "1", "5", "1", "15", "2", "13", "2", "2", "1", "13", "1", "2", "4", "267", "1", "4", "1", "5", "1", "4", "1", "50", "1", "2", "3", "4", "1", "6", "1", "52", "15", "2", "1", "15", "1", "2", "1", "12", "1", "10", "1", "4", "2"}
	if len(got) != len(want) {
		t.Fatalf("TermsList: got %d terms, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("TermsList: term %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSequenceMarshalUnmarshalJSON(t *testing.T) {
	uid, _ := util.NewUID('A', 123456)
	seq := Sequence{
		Id:       uid,
		Name:     "Test Sequence",
		Keywords: []string{"easy", "core"},
		Terms:    ",1,2,3,4,5,",
	}

	data, err := json.Marshal(seq)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var got Sequence
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if !got.Id.Equals(seq.Id) {
		t.Errorf("Id: got %v, want %v", got.Id, seq.Id)
	}
	if got.Name != seq.Name {
		t.Errorf("Name: got %q, want %q", got.Name, seq.Name)
	}
	if !reflect.DeepEqual(got.Keywords, seq.Keywords) {
		t.Errorf("Keywords: got %v, want %v", got.Keywords, seq.Keywords)
	}
	if !reflect.DeepEqual(got.TermsList(), seq.TermsList()) {
		t.Errorf("Terms: got %v, want %v", got.TermsList(), seq.TermsList())
	}
}
