package main

import (
	"testing"
)

func TestIdDomain(t *testing.T) {
	seq := Sequence{Id: "A123456"}
	got := seq.IdDomain()
	want := byte('A')
	if got != want {
		t.Errorf("IdDomain: got %q, want %q", got, want)
	}
}

func TestIdNumber(t *testing.T) {
	seq := Sequence{Id: "A123456"}
	got := seq.IdNumber()
	var want int64 = 123456
	if got != want {
		t.Errorf("IdNumber: got %d, want %d", got, want)
	}
}

func TestTermsList(t *testing.T) {
	seq := Sequence{
		Id:    "A000001",
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
