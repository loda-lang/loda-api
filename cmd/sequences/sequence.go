package main

import (
	"strconv"
	"strings"
)

type Sequence struct {
	Id       string
	Name     string
	Keywords []string
	Terms    string
}

func (s *Sequence) IdDomain() byte {
	return s.Id[0]
}

func (s *Sequence) IdNumber() int64 {
	i, err := strconv.ParseInt(s.Id[1:], 10, 64)
	if err != nil {
		return 0
	}
	return i
}

func (s *Sequence) TermsList() []string {
	var terms []string
	for _, t := range strings.Split(s.Terms, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			terms = append(terms, t)
		}
	}
	return terms
}
