package main

import (
	"strings"

	"github.com/loda-lang/loda-api/util"
)

type Sequence struct {
	Id       util.UID
	Name     string
	Keywords []string
	Terms    string
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
