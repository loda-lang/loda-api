package main

import (
	"encoding/json"
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

func (s Sequence) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id       string   `json:"id"`
		Name     string   `json:"name"`
		Keywords []string `json:"keywords"`
		Terms    []string `json:"terms"`
	}{
		Id:       s.Id.String(),
		Name:     s.Name,
		Keywords: s.Keywords,
		Terms:    s.TermsList(),
	})
}

func (s *Sequence) UnmarshalJSON(data []byte) error {
	var aux struct {
		Id       string   `json:"id"`
		Name     string   `json:"name"`
		Keywords []string `json:"keywords"`
		Terms    []string `json:"terms"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	uid, err := util.NewUIDFromString(aux.Id)
	if err != nil {
		return err
	}
	s.Id = uid
	s.Name = aux.Name
	s.Keywords = aux.Keywords
	s.Terms = strings.Join(aux.Terms, ",")
	if len(s.Terms) > 0 {
		s.Terms = "," + s.Terms + ","
	}
	return nil
}
