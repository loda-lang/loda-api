package shared

import (
	"encoding/json"
	"strings"

	"github.com/loda-lang/loda-api/util"
)

type Sequence struct {
	Id        util.UID
	Name      string
	Keywords  uint64 // bitmask of keywords
	Terms     string
	Submitter *Submitter
	Authors   []*Author
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
	var authorNames []string
	for _, a := range s.Authors {
		authorNames = append(authorNames, a.Name)
	}
	return json.Marshal(struct {
		Id       string   `json:"id"`
		Name     string   `json:"name"`
		Keywords []string `json:"keywords"`
		Terms    []string `json:"terms"`
		Authors  []string `json:"authors"`
	}{
		Id:       s.Id.String(),
		Name:     s.Name,
		Keywords: DecodeKeywords(s.Keywords),
		Terms:    s.TermsList(),
		Authors:  authorNames,
	})
}

func (s *Sequence) UnmarshalJSON(data []byte) error {
	var aux struct {
		Id       string   `json:"id"`
		Name     string   `json:"name"`
		Keywords []string `json:"keywords"`
		Terms    []string `json:"terms"`
		Authors  []string `json:"authors"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	uid, err := util.NewUIDFromString(aux.Id)
	if err != nil {
		return err
	}
	keywords, err := EncodeKeywords(aux.Keywords)
	if err != nil {
		return err
	}
	s.Id = uid
	s.Name = aux.Name
	s.Keywords = keywords
	s.Terms = strings.Join(aux.Terms, ",")
	if len(s.Terms) > 0 {
		s.Terms = "," + s.Terms + ","
	}
	// Authors: only set names, not Author pointers
	s.Authors = nil
	for _, name := range aux.Authors {
		s.Authors = append(s.Authors, &Author{Name: name})
	}
	return nil
}
